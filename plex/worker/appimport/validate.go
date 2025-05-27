package appimport

import (
	"strings"

	"userclouds.com/infra/ucerr"
	"userclouds.com/plex/internal/provider/auth0"
)

func validateAuth0App(app auth0.App) error {
	var es error // aggregated

	// translate Auth0 grant types to UserClouds grant types
	// (they should be the same, but we'll validate them)
	if _, err := mapAuth0GrantTypesToUC(app.GrantTypes); err != nil {
		es = ucerr.Combine(es, ucerr.Errorf("invalid grant type %v for a0app %s (%v): %w", app.GrantTypes, app.Name, app.ClientID, err))
	}

	// https://auth0.com/docs/get-started/applications/confidential-and-public-applications/first-party-and-third-party-applications
	if !app.IsFirstParty {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) is not a first-party app", app.Name, app.ClientID))
	}

	if !app.OIDCConformant {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) is not OIDC conformant", app.Name, app.ClientID))
	}

	for _, ao := range app.AllowedOrigins {
		if !strings.HasPrefix(ao, "http://localhost") {
			es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has non-localhost allowed origins set", app.Name, app.ClientID))
			break
		}
	}

	for _, wo := range app.WebOrigins {
		if !strings.HasPrefix(wo, "http://localhost") {
			es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has non-localhost web origins set", app.Name, app.ClientID))
			break
		}
	}

	if len(app.ClientAliases) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has client aliases set", app.Name, app.ClientID))
	}

	if len(app.AllowedClients) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has allowed clients set", app.Name, app.ClientID))
	}

	if len(app.OIDCBackchannelLogout) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has OIDC backchannel logout set", app.Name, app.ClientID))
	}

	if app.JWTConfiguration != auth0.DefaultJWTConfiguration {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has non-default JWT configuration: %+v", app.Name, app.ClientID, app.JWTConfiguration))
	}

	if len(app.SigningKeys) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has signing keys set", app.Name, app.ClientID))
	}

	if app.EncryptionKey != nil {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has encryption key set", app.Name, app.ClientID))
	}

	if app.SSO {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has SSO set", app.Name, app.ClientID))
	}

	// we can probably live without this for a while (at least until we support SSO :) ), but warn
	if app.SSO && !app.SSODisabled {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has SSO enabled", app.Name, app.ClientID))
	}

	// this should be redundant to the CORS array above?
	if app.CrossOriginAuthentication {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has cross-origin authentication set", app.Name, app.ClientID))
	}

	if app.CustomLoginPageOn {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has custom login page set", app.Name, app.ClientID))
	}

	// explicitly not checking app.CustomLoginPage or ...Preview since it's only used if the above is true

	if app.FormTemplate != "" {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has form template set", app.Name, app.ClientID))
	}

	if len(app.AddOns) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has add-ons set", app.Name, app.ClientID))
	}

	// explicitly ignoring TokenEndpointAuthMethod since we support them all (but don't yet restrict, and don't see a reason to?)

	if len(app.ClientMetadata) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has client metadata set", app.Name, app.ClientID))
	}

	if len(app.Mobile) > 0 {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has mobile set", app.Name, app.ClientID))
	}

	if app.InitiateLoginURI != "" {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has initiate login URI set", app.Name, app.ClientID))
	}

	for s, n := range app.NativeSocialLogin {
		if n.Enabled {
			es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has native social login for %v enabled", app.Name, app.ClientID, s))
			break
		}
	}

	if app.RefreshToken != auth0.DefaultRefreshTokenSettings {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has non-default refresh token settings: %+v", app.Name, app.ClientID, app.RefreshToken))
	}

	if app.OrganizationUsage != "" {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has organization usage set", app.Name, app.ClientID))
	}

	// explicitly skipping app.OrganizationRequireBehavior until we support OrganizationUsage above

	if app.CallbackURLTemplate {
		es = ucerr.Combine(es, ucerr.Errorf("auth0 app %s (%v) has callback URL template set", app.Name, app.ClientID))
	}

	return ucerr.Wrap(es)
}
