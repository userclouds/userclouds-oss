package uc

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/token"
)

type authClient struct {
	iface.BaseClient
	id              uuid.UUID
	name            string
	p               tenantplex.UCProvider
	app             tenantplex.UCApp
	loginChallenger *loginChallenger

	authIDP *idp.AuthClient
	mgmtIDP *idp.ManagementClient
}

func newJWT(ctx context.Context, tc *tenantplex.TenantConfig, tenantURL string, appID uuid.UUID, appOrgID uuid.UUID) (string, error) {
	// We use appID as the subject, and the audience field is the URL of the tenant for now
	return token.CreateAccessTokenJWT(ctx,
		tc,
		uuid.Must(uuid.NewV4()),
		appID.String(),
		authz.ObjectTypeLoginApp,
		appOrgID.String(),
		tenantURL,
		[]string{tenantURL},
		ucjwt.DefaultValidityAccess,
	) // since this is an internal UC-to-UC communication, we use our own defaults, not the customer plex app

}

func newAuthClient(ctx context.Context, tenantURL string, plexClientID string, appID uuid.UUID, appOrgID uuid.UUID) (*idp.AuthClient, *idp.ManagementClient, error) {
	// NB: this implicitly adds a dependency on tenantconfig.Middleware, which is fine
	// because all of the code in plex/internal/provider/factory.go (which creates these clients)
	// depends on it too.
	tc := tenantconfig.MustGet(ctx)

	j, err := newJWT(ctx, &tc, tenantURL, appID, appOrgID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	authClient, err := idp.NewAuthClient(tenantURL, plexClientID, jsonclient.HeaderAuthBearer(j), security.PassXForwardedFor())
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	mgmtClient, err := idp.NewManagementClient(tenantURL, jsonclient.HeaderAuthBearer(j), security.PassXForwardedFor())
	return authClient, mgmtClient, ucerr.Wrap(err)
}

// NewClient creates a UC provider client that implements iface.Client.
func NewClient(ctx context.Context,
	id uuid.UUID,
	name string,
	uc tenantplex.UCProvider,
	providerAppID uuid.UUID,
	plexClientID string,
	emailClient *email.Client) (iface.Client, error) {
	// appID has been validated to exist already
	var app *tenantplex.UCApp
	for i := range uc.Apps {
		if uc.Apps[i].ID == providerAppID {
			app = &uc.Apps[i]
			break
		}
	}
	if app == nil {
		return nil, ucerr.Errorf("app ID %v not found in provider %s (%v) despite validation", providerAppID, name, id)
	}

	tc := tenantconfig.MustGet(ctx)
	loginApp, _, err := tc.PlexMap.FindAppForClientID(plexClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	idpAuthClient, mc, err := newAuthClient(ctx, uc.IDPURL, plexClientID, loginApp.ID, loginApp.OrganizationID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	ac := &authClient{
		id:      id,
		name:    name,
		p:       uc,
		app:     *app,
		authIDP: idpAuthClient,
		mgmtIDP: mc,
	}
	if emailClient != nil {
		ac.loginChallenger = &loginChallenger{clientID: plexClientID, emailClient: *emailClient}
	}
	return ac, nil
}

// claimsFromUserID gets the user profile data from UC IDP (which is a strongly typed struct
// containing many fields which are also OIDC-compliant claims) and copies it into
// a generic dict (jwt.MapClaims) which is used to create a signed token. Since JWTs can have
// any claims in addition to 'standard' ones, this is the simplest way to get the user profile
// encoded as claims in tokens without manually copying each field or possibly forgetting new
// ones that get added to the user profile but left out here.
// TODO: Codegen is probably a better way to generate this method, and explicitly name
// each field in a searchable manner without accidentally leaving some out.
func (c *authClient) claimsFromUserID(ctx context.Context, userID uuid.UUID) (jwt.MapClaims, error) {
	resp, err := c.mgmtIDP.GetUserBaseProfileAndAuthN(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	claims := jwt.MapClaims{}
	claims["email"] = resp.UserBaseProfile.Email
	claims["email_verified"] = resp.UserBaseProfile.EmailVerified
	if resp.UserBaseProfile.Name != "" {
		claims["name"] = resp.UserBaseProfile.Name
	}
	if resp.UserBaseProfile.Nickname != "" {
		claims["nickname"] = resp.UserBaseProfile.Nickname
	}
	if resp.UserBaseProfile.Picture != "" {
		claims["picture"] = resp.UserBaseProfile.Picture
	}
	claims["sub"] = userID.String()
	return claims, nil
}

func (c *authClient) UsernamePasswordLogin(ctx context.Context, username, password string) (*iface.LoginResponseWithClaims, error) {
	idpResp, err := c.authIDP.Login(ctx, username, password)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	resp := iface.LoginResponseWithClaims{
		Status:                       idpResp.Status,
		EvaluateSupportedMFAChannels: idpResp.EvaluateSupportedMFAChannels,
	}

	switch idpResp.Status {
	case idp.LoginStatusSuccess:
		claims, err := c.claimsFromUserID(ctx, idpResp.UserID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		resp.Claims = claims
	case idp.LoginStatusMFARequired:
		resp.MFAProvider = c.id
		resp.MFAToken = idpResp.MFAToken
		resp.SupportedMFAChannels = idpResp.SupportedMFAChannels
	default:
		return nil, ucerr.Errorf("unexpected LoginStatus: '%s'", idpResp.Status)
	}

	return &resp, nil
}

func (c *authClient) MFAChallenge(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	if c.loginChallenger == nil {
		return nil, ucerr.Friendlyf(nil, "MFA is not supported since email is disabled")
	}
	idpResp, err := c.authIDP.GetMFACode(ctx, mfaToken, channel)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &idpResp.MFAChannel, ucerr.Wrap(c.loginChallenger.issueChallenge(ctx, idpResp.MFACode, idpResp.MFAChannel, authChallengeType))
}

func (c *authClient) MFAClearPrimaryChannel(ctx context.Context, mfaToken string) (*oidc.MFAChannels, error) {
	idpResp, err := c.authIDP.MFAClearPrimaryChannel(ctx, mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &idpResp.MFAChannels, nil
}

func (c *authClient) MFACreateChannel(ctx context.Context, mfaToken string, channelType oidc.MFAChannelType, channelTypeID string) (*oidc.MFAChannel, error) {
	if c.loginChallenger == nil {
		return nil, ucerr.Friendlyf(nil, "MFA is not supported since email is disabled")
	}
	idpResp, err := c.authIDP.MFACreateChannel(ctx, mfaToken, channelType, channelTypeID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &idpResp.MFAChannel, ucerr.Wrap(c.loginChallenger.issueChallenge(ctx, idpResp.MFACode, idpResp.MFAChannel, verifyChallengeType))
}

func (c *authClient) MFADeleteChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) {
	idpResp, err := c.authIDP.MFADeleteChannel(ctx, mfaToken, channel)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &idpResp.MFAChannels, nil
}

func (c *authClient) MFAGetChannels(ctx context.Context, userID string) (*iface.MFAGetChannelsResponse, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	idpResp, err := c.authIDP.GetMFAChannels(ctx, userUUID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	resp := iface.MFAGetChannelsResponse{
		MFAToken:             idpResp.MFAToken.String(),
		MFAProvider:          c.id,
		SupportedMFAChannels: idpResp.MFAChannels,
	}

	return &resp, nil
}

func (c *authClient) MFALogin(ctx context.Context, mfaToken string, challengeCode string, channel oidc.MFAChannel) (*iface.LoginResponseWithClaims, error) {
	idpResp, err := c.authIDP.LoginWithMFA(ctx, mfaToken, challengeCode)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	resp := iface.LoginResponseWithClaims{
		Status: idpResp.Status,
	}

	switch idpResp.Status {
	case idp.LoginStatusMFACodeInvalid:
	case idp.LoginStatusMFACodeExpired:
	case idp.LoginStatusSuccess:
		claims, err := c.claimsFromUserID(ctx, idpResp.UserID)
		if err != nil {
			return nil, ucerr.Errorf("error getting user info after logging in to UC IDP: %w", err)
		}
		if channel.ChannelType == oidc.MFAEmailChannel {
			extractedClaims, err := oidc.ExtractClaims(claims)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			if !extractedClaims.EmailVerified && extractedClaims.Email == channel.ChannelTypeID {
				if err := setEmailVerified(ctx, c.mgmtIDP, idpResp.UserID, true); err != nil {
					return nil, ucerr.Wrap(err)
				}
				claims["email_verified"] = true
			}
		}
		resp.Claims = claims
		resp.NewRecoveryCode = idpResp.NewRecoveryCode
		return &resp, nil
	default:
		return nil, ucerr.Errorf("unexpected LoginStatus: '%s'", idpResp.Status)
	}

	resp.MFAProvider = c.id
	resp.MFAToken = mfaToken
	return &resp, nil
}

func (c *authClient) MFAMakePrimaryChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) {
	idpResp, err := c.authIDP.MFAMakePrimaryChannel(ctx, mfaToken, channel)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &idpResp.MFAChannels, nil
}

func (c *authClient) MFAReissueRecoveryCode(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	idpResp, err := c.authIDP.MFAReissueRecoveryCode(ctx, mfaToken, channel)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &idpResp.MFAChannel, nil
}

func (authClient) SupportsMFAConfiguration() bool {
	return true
}

func (c *authClient) LoginURL(ctx context.Context, sessionID uuid.UUID, app *tenantplex.App) (*url.URL, error) {
	tc := tenantconfig.MustGet(ctx)
	authMethods, err := tenantplex.GetAuthenticationMethods(&tc, app)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	switch len(authMethods) {
	case 1:
		if authMethods[0] == "passwordless" {
			return paths.PasswordlessLoginURL(ctx, sessionID)
		}
		// fallthrough to normal login URL
	case 0:
		return nil, ucerr.New("no authentication methods specified")
	default:
		// fall through to normal login URL
	}

	return paths.LoginURL(ctx, sessionID)
}

func (c *authClient) Logout(ctx context.Context, redirectURL string) (string, error) {
	// No-op for now. UC IDP & Plex don't yet set cookies so there's nothing to clear.
	// TODO: validate redirect URI is allowed for this app.
	return redirectURL, nil
}

func (c authClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v', app name '%s', app id: '%s'",
		tenantplex.ProviderTypeUC, c.name, c.id, c.app.Name, c.app.ID)
}
