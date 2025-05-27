package auth0

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// App represents an Auth0 application.
// defined here: https://auth0.com/docs/api/management/v2#!/Clients/get_clients
// TODO: current order matches the docs but is hard to reason about, should we do our own grouping?
type App struct {
	ClientID                    string                       `json:"client_id"`
	Tenant                      string                       `json:"tenant"`
	Name                        string                       `json:"name"`
	Description                 string                       `json:"description"`
	Global                      bool                         `json:"global"`
	ClientSecret                string                       `json:"client_secret"`
	AppType                     string                       `json:"app_type"`
	LogoURI                     string                       `json:"logo_uri"`
	IsFirstParty                bool                         `json:"is_first_party"`
	OIDCConformant              bool                         `json:"oidc_conformant"`
	Callbacks                   []string                     `json:"callbacks"`
	AllowedOrigins              []string                     `json:"allowed_origins"`
	WebOrigins                  []string                     `json:"web_origins"`
	ClientAliases               []string                     `json:"client_aliases"`
	AllowedClients              []string                     `json:"allowed_clients"`
	AllowedLogoutURLs           []string                     `json:"allowed_logout_urls"`
	OIDCBackchannelLogout       map[string]any               `json:"oidc_backchannel_logout"`
	GrantTypes                  []string                     `json:"grant_types"`
	JWTConfiguration            JWTConfiguration             `json:"jwt_configuration"`
	SigningKeys                 []any                        `json:"signing_keys"`
	EncryptionKey               any                          `json:"encryption_key"`
	SSO                         bool                         `json:"sso"`
	SSODisabled                 bool                         `json:"sso_disabled"`
	CrossOriginAuthentication   bool                         `json:"cross_origin_auth"`
	CrossOriginLoc              string                       `json:"cross_origin_loc"`
	CustomLoginPageOn           bool                         `json:"custom_login_page_on"`
	CustomLoginPage             string                       `json:"custom_login_page"`
	CustomLoginPagePreview      string                       `json:"custom_login_page_preview"`
	FormTemplate                string                       `json:"form_template"`
	AddOns                      map[string]any               `json:"addons"`
	TokenEndpointAuthMethod     string                       `json:"token_endpoint_auth_method"`
	ClientMetadata              map[string]any               `json:"client_metadata"`
	Mobile                      map[string]any               `json:"mobile"`
	InitiateLoginURI            string                       `json:"initiate_login_uri"`
	NativeSocialLogin           map[string]NativeSocialLogin `json:"native_social_login"`
	RefreshToken                RefreshToken                 `json:"refresh_token"`
	OrganizationUsage           string                       `json:"organization_usage"`
	OrganizationRequireBehavior string                       `json:"organization_require_behavior"`

	CallbackURLTemplate bool `json:"callback_url_template"` // undocumented?
}

// NativeSocialLogin is used by App
type NativeSocialLogin struct {
	Enabled bool `json:"enabled"`
}

// JWTConfiguration is used by App
type JWTConfiguration struct {
	Alg               string `json:"alg"`
	LifetimeInSeconds int    `json:"lifetime_in_seconds"`
	SecretEncoded     bool   `json:"secret_encoded"`
}

// DefaultJWTConfiguration specifies the auth0 default
var DefaultJWTConfiguration = JWTConfiguration{
	Alg:               "RS256",
	LifetimeInSeconds: 36000,
	SecretEncoded:     false,
}

// RefreshToken is used by App
type RefreshToken struct {
	ExpirationType            string `json:"expiration_type"`
	Leeway                    int    `json:"leeway"`
	InfiniteTokenLifetime     bool   `json:"infinite_token_lifetime"`
	InfiniteIdleTokenLifetime bool   `json:"infinite_idle_token_lifetime"`
	TokenLifetime             int    `json:"token_lifetime"`
	IdleTokenLifetime         int    `json:"idle_token_lifetime"`
	RotationType              string `json:"rotation_type"`
}

// DefaultRefreshTokenSettings specifies the auth0 default
var DefaultRefreshTokenSettings = RefreshToken{
	ExpirationType:            "non-expiring",
	Leeway:                    0,
	InfiniteTokenLifetime:     true,
	InfiniteIdleTokenLifetime: true,
	TokenLifetime:             31557600,
	IdleTokenLifetime:         2592000,
	RotationType:              "non-rotating",
}

// ListApps returns all apps in the Auth0 tenant.
func (mc MgmtClient) ListApps(ctx context.Context) ([]App, error) {
	var apps []App
	page := 0

	for {
		uclog.Debugf(ctx, "auth0 list apps page %d", page)
		vals := url.Values{
			// Auth0 doesn't seem to support Lucene > queries on updated_at, so we're using this janky workaround
			"per_page": []string{strconv.Itoa(pageSize)}, // Auth0 max, just to be explicit
			"page":     []string{strconv.Itoa(page)},
		}

		pathURL := &url.URL{
			Path:     "/api/v2/clients",
			RawQuery: vals.Encode(),
		}

		var resp []App
		if err := mc.client.Get(ctx, pathURL.String(), &resp); err != nil {
			uclog.Debugf(ctx, "auth0 list apps error: %+v", err)
			return nil, ucerr.Wrap(err)
		}

		apps = append(apps, resp...)

		// TODO: I think the page size and limits are probably different on this
		// API from ListUsers, but 1500 apps seems sufficient for now? As of now
		// I haven't found any documentation on limiting maxPages for non-Lucene backed queries
		if len(resp) == pageSize && page < (maxPages-1) {
			// try another page and see?
			time.Sleep(perRequestDelay) // TODO: this is super naive and assumes only one user of this API
			page++
			continue
		}

		break
	}

	return apps, nil
}
