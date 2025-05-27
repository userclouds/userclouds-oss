package oidc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

// AuthCodeOptionGetter is a function that returns a set of auth code options
// for a passed in http request
type AuthCodeOptionGetter func(*http.Request) []oauth2.AuthCodeOption

var defaultAuthCodeOptionGetter AuthCodeOptionGetter = func(*http.Request) []oauth2.AuthCodeOption {
	return nil
}

// Authenticator abstracts away different OIDC providers.
type Authenticator struct {
	Provider                 *gooidc.Provider
	Config                   oauth2.Config
	alternateSubjectClaimKey string
	AuthCodeOptionGetter     AuthCodeOptionGetter

	OverrideValidateIssuer func(string) error
}

// DefaultScopes defines the default OAuth/OIDC scopes requested for basic authentication.
var DefaultScopes = fmt.Sprintf("%s %s %s", gooidc.ScopeOpenID, "profile", "email")
var defaultScopeTokens = SplitTokens(DefaultScopes)

// NewAuthenticator creates a new Authenticator with default scope via OIDC Discovery.
func NewAuthenticator(ctx context.Context, idpURL string, clientID string, clientSecret secret.String, redirectURL string) (*Authenticator, error) {
	return newAuthenticatorViaDiscovery(ctx, idpURL, clientID, clientSecret, redirectURL, defaultScopeTokens)
}

func newAuthenticatorViaDiscovery(
	ctx context.Context, idpURL string, clientID string, clientSecret secret.String,
	redirectURL string, scopes []string) (*Authenticator, error) {

	cs, err := clientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	p, err := gooidc.NewProvider(ctx, idpURL)
	if err != nil {
		return nil, ucerr.Friendlyf(err, "Failed to create OIDC provider for issuer %v. This may be due to network issues, an incorrect issuer URL, or unexpected response content. Please verify the URL and your network connection.", idpURL)
	}

	return newAuthenticator(p, clientID, cs, redirectURL, scopes), nil
}

func newAuthenticatorViaConfiguration(
	ctx context.Context, idpURL string, authURL string, tokenURL string,
	userInfoURL string, jwksURL string, alternateSubjectClaimKey string,
	clientID string, clientSecret secret.String, redirectURL string, scopes []string) (*Authenticator, error) {

	cs, err := clientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	pc := gooidc.ProviderConfig{
		IssuerURL:   idpURL,
		AuthURL:     authURL,
		TokenURL:    tokenURL,
		UserInfoURL: userInfoURL,
		JWKSURL:     jwksURL,
	}
	p := pc.NewProvider(ctx)

	a := newAuthenticator(p, clientID, cs, redirectURL, scopes)
	a.alternateSubjectClaimKey = alternateSubjectClaimKey
	return a, nil
}

func newAuthenticator(p *gooidc.Provider, clientID string, clientSecret string, redirectURL string, scopes []string) *Authenticator {
	c := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       scopes,
	}

	return &Authenticator{Provider: p, Config: c, AuthCodeOptionGetter: defaultAuthCodeOptionGetter}
}

func typedValue[T any](m map[string]any, key string, defaultValue T) T {
	if m[key] != nil {
		if value, ok := m[key].(T); ok {
			return value
		}
	}

	return defaultValue
}

func boolValue(m map[string]any, key string) bool {
	return typedValue(m, key, false)
}

func stringValue(m map[string]any, key string) string {
	return typedValue(m, key, "")
}

// ExtractClaims ensures that certain common properties are available
// in claims returned in a JWT and/or /userinfo OIDC endpoint.
// TODO: 'email' is probably not required longer term? But 'sub' probably should be?
func ExtractClaims(claims map[string]any) (*UCTokenClaims, error) {
	var commonClaims UCTokenClaims

	commonClaims.Subject = stringValue(claims, "sub")
	if commonClaims.Subject == "" {
		return nil, ucerr.Errorf("failed to get valid 'sub' from user info claims profile")
	}
	commonClaims.Email = stringValue(claims, "email")
	if commonClaims.Email == "" {
		return nil, ucerr.Errorf("failed to get 'email' from user info claims profile")
	}

	commonClaims.EmailVerified = boolValue(claims, "email_verified")
	commonClaims.Name = stringValue(claims, "name")
	commonClaims.Nickname = stringValue(claims, "nickname")
	commonClaims.Picture = stringValue(claims, "picture")
	commonClaims.OrganizationID = stringValue(claims, "organization_id")
	return &commonClaims, nil
}

// UserInfoProfile is a mapping from claim type key to claim for an oidc UserInfo
type UserInfoProfile map[string]any

// TokenInfo contains Authenticated information (token, profile) about a user.
type TokenInfo struct {
	RawIDToken   string
	AccessToken  string
	RefreshToken string
	Profile      UserInfoProfile

	// Claims are derived from Profile but do not include all possible claims.
	Claims UCTokenClaims
}

func (authr *Authenticator) userInfoProfileForToken(ctx context.Context, token *oauth2.Token) (UserInfoProfile, error) {
	var uip UserInfoProfile

	// TODO: do we need to hit the user info endpoint or can we get what we need
	// from the ID token in most/all cases?
	userInfo, err := authr.Provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return uip, ucerr.Wrap(err)
	}

	if err = userInfo.Claims(&uip); err != nil {
		return uip, ucerr.Wrap(err)
	}

	// if the subject is passed under an alternate claim key, make sure it can be found
	if authr.alternateSubjectClaimKey != "" {
		if subject := stringValue(uip, authr.alternateSubjectClaimKey); subject != "" {
			uip["sub"] = subject
		}
	}

	return uip, nil
}

// ProcessAuthCodeCallback handles a standard OIDC authorization code callback, assuming
// `code` and `state` are supplied query params.
// It extracts/gets/returns the raw ID token, access token, and user profile and returns
// an appropriate HTTP status code.
func (authr *Authenticator) ProcessAuthCodeCallback(r *http.Request, state string) (*TokenInfo, int, error) {
	ctx := r.Context()
	if r.URL.Query().Get("state") != state {
		return nil, http.StatusBadRequest, ucerr.New("invalid state parameter value")
	}

	token, err := authr.Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		return nil, http.StatusUnauthorized, ucerr.Wrap(err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, http.StatusBadRequest, ucerr.New("no id_token field in oauth2 token")
	}

	oidcConfig := &gooidc.Config{
		ClientID:        authr.Config.ClientID,
		SkipIssuerCheck: authr.OverrideValidateIssuer != nil, // SkipIssuer if we have our own validator, eg. MSFT
	}

	// TODO: It's not clear if 'internal server error' is actually the right error
	// in many of these cases, since it could either be network failures OR
	// a bad token from the IDP. Hard to say without deeper analysis.
	verifiedToken, err := authr.Provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	userInfoProfile, err := authr.userInfoProfileForToken(ctx, token)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	claims, err := ExtractClaims(userInfoProfile)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if authr.OverrideValidateIssuer != nil {
		if err := authr.OverrideValidateIssuer(verifiedToken.Issuer); err != nil {
			return nil, http.StatusUnauthorized, ucerr.Wrap(err)
		}
	}

	return &TokenInfo{
		RawIDToken:   rawIDToken,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Profile:      userInfoProfile,
		Claims:       *claims,
	}, http.StatusOK, nil
}

func trimEmpty(strs []string) []string {
	var trimmedStrs []string
	for _, v := range strs {
		v = strings.TrimSpace(v)
		if v != "" {
			trimmedStrs = append(trimmedStrs, v)
		}
	}
	return trimmedStrs
}

// SplitTokens splits an OIDC space-delimited string (e.g. scopes, grant types,
// response types) into an array of non-empty scopes.
func SplitTokens(scopes string) []string {
	return trimEmpty(strings.Split(scopes, " "))
}
