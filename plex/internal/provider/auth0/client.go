package auth0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/token"
)

type client struct {
	iface.BaseClient
	id      uuid.UUID
	name    string
	p       tenantplex.Auth0Provider
	app     tenantplex.Auth0App
	client  *sdkclient.Client
	baseURL *url.URL
}

// NewClient creates an Auth0 provider client that implements iface.Client.
func NewClient(id uuid.UUID, name string, auth0 tenantplex.Auth0Provider, appID uuid.UUID) (iface.Client, error) {
	// appID has been validated to exist already
	var app *tenantplex.Auth0App
	for i := range auth0.Apps {
		if auth0.Apps[i].ID == appID {
			app = &auth0.Apps[i]
			break
		}
	}
	if app == nil {
		return nil, ucerr.Errorf("app ID %v not found in provider %s (%v) despite validation", appID, name, id)
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   auth0.Domain,
		Path:   "/",
	}
	jc := sdkclient.New(baseURL.String(), "plex-auth0")

	return &client{
		id:      id,
		name:    name,
		p:       auth0,
		app:     *app,
		client:  jc,
		baseURL: baseURL,
	}, nil
}

// auth0LoginRequest describes the JSON request to send to Auth0's
// `/oauth/token` end point for a Resource Owner Password Credentials grant.
// https://auth0.com/docs/api/authentication#resource-owner-password
type auth0LoginRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	GrantType    string `json:"grant_type"`
	Scope        string `json:"scope"`
}

// auth0TokenResponse describes the JSON response from Auth0's
// `/oauth/token` endpoint, and extends the OIDC standard response.
type auth0TokenResponse struct {
	oidc.TokenResponse

	MFAToken     string `json:"mfa_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	RecoveryCode string `json:"recovery_code,omitempty"`
}

func (c *client) UsernamePasswordLogin(ctx context.Context, username, password string) (*iface.LoginResponseWithClaims, error) {
	cs, err := c.app.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	auth0Req := auth0LoginRequest{
		ClientID:     c.app.ClientID,
		ClientSecret: cs,
		Username:     username,
		Password:     password,
		GrantType:    "password",
		Scope:        "openid profile email",
	}

	var idpResp auth0TokenResponse
	// NB (sgarrity 5/9/23): we used to pass the redirect URL in a header here (jsonclient.Header("Origin", redirectURL)),
	// but as I implemented the Plex ROPC flow, there is no valid redirect URL to use. Rather than pass something silly,
	// I started trying to figure out why it was here (and never figured it out).
	//
	// 1) auth0 docs don't require it: https://auth0.com/docs/get-started/authentication-and-authorization-flow/call-your-api-using-resource-owner-password-flow
	// 2) it was originally introduced here with no comment: `056b6e0e0`
	// 3) @kutta.s, who wrote this originally, doesn't remember why it's there (although we both agree it probably had a purpose)
	// 4) it works fine today without it.
	//
	// so I am leaving this comment in case someone in the future is diagnosing a weird failure with Auth0's ROPC and this helps.
	err = c.client.Post(ctx, "/oauth/token", auth0Req, &idpResp, jsonclient.UnmarshalOnError())
	if err == nil {
		claims, err := token.ExtractClaimsFromJWT(ctx, c.baseURL.String(), c.app.ClientID, idpResp.IDToken)
		if err != nil {
			return nil, ucerr.Errorf("error extracting claims from Auth0 login JWT: %w", err)
		}

		return &iface.LoginResponseWithClaims{Status: idp.LoginStatusSuccess, Claims: claims}, nil
	}

	if len(idpResp.ErrorType) == 0 {
		return nil, ucerr.Wrap(err)
	}

	if idpResp.ErrorType != "mfa_required" {
		return nil, ucerr.Errorf("Error: %s, Description: %s", idpResp.ErrorType, idpResp.ErrorDesc)
	}

	// mfa is required

	mfaChannels, err := c.getSupportedMFAChannels(ctx, idpResp.MFAToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &iface.LoginResponseWithClaims{
		Status:               idp.LoginStatusMFARequired,
		MFAProvider:          c.id,
		MFAToken:             idpResp.MFAToken,
		SupportedMFAChannels: mfaChannels,
	}, nil
}

type auth0MFAAuthenticator struct {
	ID                string `json:"id"`
	AuthenticatorType string `json:"authenticator_type"`
	Active            bool   `json:"active"`
	OOBChannel        string `json:"oob_channel,omitempty"`
	Name              string `json:"name,omitempty"`
}

func (a auth0MFAAuthenticator) getChannelType() oidc.MFAChannelType {
	switch a.AuthenticatorType {
	case "otp":
		return oidc.MFAAuth0AuthenticatorChannel
	case "oob":
		switch a.OOBChannel {
		case "sms":
			return oidc.MFAAuth0SMSChannel
		case "email":
			return oidc.MFAAuth0EmailChannel
		}
	case "recovery-code":
		return oidc.MFARecoveryCodeChannel
	}

	return oidc.MFAInvalidChannel
}

var preferredChannelTypes = []oidc.MFAChannelType{oidc.MFAAuth0AuthenticatorChannel, oidc.MFAAuth0SMSChannel, oidc.MFAAuth0EmailChannel}

func (c *client) getSupportedMFAChannels(ctx context.Context, mfaToken string) (channels oidc.MFAChannels, err error) {
	var auths []auth0MFAAuthenticator
	if err := c.client.Get(ctx, "/mfa/authenticators", &auths, jsonclient.HeaderAuthBearer(mfaToken)); err != nil {
		return channels, ucerr.Wrap(err)
	}

	channelIDsByChannelType := map[oidc.MFAChannelType]uuid.UUID{}
	for _, a := range auths {
		if !a.Active {
			continue
		}

		channelType := a.getChannelType()
		if channelType == oidc.MFAInvalidChannel {
			continue
		}

		if channels.ChannelTypes[channelType] {
			continue
		}

		channel, err := channels.AddChannel(channelType, a.ID, a.Name, true)
		if err != nil {
			return channels, ucerr.Wrap(err)
		}

		channelIDsByChannelType[channelType] = channel.ID
		channels.ChannelTypes[channelType] = true
	}

	// make the most preferred channel primary
	for _, ct := range preferredChannelTypes {
		channelID, found := channelIDsByChannelType[ct]
		if found {
			if err := channels.SetPrimary(channelID); err != nil {
				return channels, ucerr.Wrap(err)
			}

			if err := channels.Validate(); err != nil {
				return channels, ucerr.Wrap(err)
			}

			return channels, nil
		}
	}

	return channels, ucerr.New("there is no primary MFA channel")
}

type auth0MFAChallengeRequest struct {
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	ChallengeType   string `json:"challenge_type"`
	MFAToken        string `json:"mfa_token"`
	AuthenticatorID string `json:"authenticator_id"`
}

type auth0MFAChallengeResponse struct {
	ChallengeType string `json:"challenge_type"`
	BindingMethod string `json:"binding_method"`
	OOBCode       string `json:"oob_code"`
}

// Validate implements the Validatable interface
func (r auth0MFAChallengeResponse) Validate() error {
	switch r.ChallengeType {
	case "otp":
	case "oop":
		if r.BindingMethod != "prompt" {
			return ucerr.Errorf("invalid binding_method:'%v'", r)
		}
		if r.OOBCode == "" {
			return ucerr.Errorf("missing oob_code: '%v'", r)
		}
	default:
		return ucerr.Errorf("unsupported chalenge_type: '%v'", r)
	}

	return nil
}

// MFAChallenge issues the auth0 MFA challenge for the specified MFA token
func (c *client) MFAChallenge(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	// recovery code login does not require a challenge
	if channel.ChannelType == oidc.MFARecoveryCodeChannel {
		return &channel, nil
	}

	cs, err := c.app.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	req := auth0MFAChallengeRequest{
		ClientID:        c.app.ClientID,
		ClientSecret:    cs,
		MFAToken:        mfaToken,
		AuthenticatorID: channel.ChannelTypeID,
	}

	switch channel.ChannelType {
	case oidc.MFAAuth0AuthenticatorChannel:
		req.ChallengeType = "otp"
	case oidc.MFAAuth0EmailChannel, oidc.MFAAuth0SMSChannel:
		req.ChallengeType = "oob"
	default:
		return nil, ucerr.Errorf("unsupported MFA channel type '%v'", channel.ChannelType)
	}

	var resp auth0MFAChallengeResponse
	if err := c.client.Post(ctx, "/mfa/challenge", req, &resp, jsonclient.HeaderAuthBearer(mfaToken)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := resp.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if channel.ChannelType != oidc.MFAAuth0AuthenticatorChannel {
		// this must be the email or sms channel - capture
		// the OOBCode for use in the subsequent MFALogin
		channel.ChallengeKey = resp.OOBCode
	}

	return &channel, nil
}

func (c *client) MFALogin(ctx context.Context, mfaToken string, challengeCode string, channel oidc.MFAChannel) (*iface.LoginResponseWithClaims, error) {
	cs, err := c.app.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// TODO: do we need the header bearer token?
	data := url.Values{
		"client_id":     {c.app.ClientID},
		"client_secret": {cs},
		"mfa_token":     {mfaToken},
	}
	auth0TokenURL := fmt.Sprintf("https://%s/oauth/token", c.p.Domain)

	switch channel.ChannelType {
	case oidc.MFAAuth0AuthenticatorChannel:
		data.Add("grant_type", "http://auth0.com/oauth/grant-type/mfa-otp")
		data.Add("otp", challengeCode)
	case oidc.MFAAuth0EmailChannel, oidc.MFAAuth0SMSChannel:
		data.Add("grant_type", "http://auth0.com/oauth/grant-type/mfa-oob")
		data.Add("oob_code", channel.ChallengeKey)
		data.Add("binding_code", challengeCode)
	case oidc.MFARecoveryCodeChannel:
		data.Add("grant_type", "http://auth0.com/oauth/grant-type/mfa-recovery-code")
		data.Add("recovery_code", challengeCode)
	default:
		return nil, ucerr.Errorf("unsupported channel type '%v'", channel.ChannelType)
	}

	res, err := http.PostForm(auth0TokenURL, data)
	if err != nil {
		// TODO: distinguish between errors - see:
		//   https://auth0.com/docs/api/authentication#standard-error-responses
		return nil, ucerr.Wrap(err)
	}

	var idpResp auth0TokenResponse
	if err := json.NewDecoder(res.Body).Decode(&idpResp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	claims, err := token.ExtractClaimsFromJWT(ctx, c.baseURL.String(), c.app.ClientID, idpResp.IDToken)
	if err != nil {
		return nil, ucerr.Errorf("error extracting claims from Auth0 login JWT: %w", err)
	}

	resp := iface.LoginResponseWithClaims{
		Status: idp.LoginStatusSuccess,
		Claims: claims,
	}

	if channel.ChannelType == oidc.MFARecoveryCodeChannel {
		if idpResp.RecoveryCode == "" {
			return nil, ucerr.Errorf("no new recovery code provided after recovery code login")
		}

		resp.NewRecoveryCode = idpResp.RecoveryCode
	}

	return &resp, nil
}

func (c *client) auth0RedirectURL(ctx context.Context, sessionID uuid.UUID, plexApp *tenantplex.App) (*url.URL, error) {
	tenantURL := tenantconfig.MustGetTenantURLString(ctx)

	redirectURL := fmt.Sprintf("%s/delegation%s", tenantURL, paths.Auth0RedirectCallbackPath)
	auth0App, err := c.p.FindProviderApp(plexApp)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	a, err := oidc.NewAuthenticator(ctx, fmt.Sprintf("https://%s/", c.p.Domain), auth0App.ClientID, auth0App.ClientSecret, redirectURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	redirectTo, err := url.Parse(a.Config.AuthCodeURL(sessionID.String()))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return redirectTo, nil
}

func (c *client) LoginURL(ctx context.Context, sessionID uuid.UUID, plexApp *tenantplex.App) (*url.URL, error) {
	if c.p.Redirect {
		return c.auth0RedirectURL(ctx, sessionID, plexApp)
	}

	return paths.LoginURL(ctx, sessionID)
}

func (c *client) Logout(ctx context.Context, redirectURL string) (string, error) {
	query := url.Values{}
	query.Add("client_id", c.app.ClientID)
	query.Add("returnTo", redirectURL)
	logoutURL := "https://" + c.p.Domain + "/v2/logout?" + query.Encode()
	return logoutURL, nil
}

func (c client) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v', app name '%s', app id: '%s'",
		tenantplex.ProviderTypeAuth0, c.name, c.id, c.app.Name, c.app.ID)
}
