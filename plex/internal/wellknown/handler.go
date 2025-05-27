package wellknown

import (
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"
	jose "gopkg.in/square/go-jose.v2"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/workerclient"
	acmeHandler "userclouds.com/plex/internal/acme"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
}

// NewHandler returns a new wellknown handler
func NewHandler(acmeCfg *acme.Config, wc workerclient.Client) http.Handler {
	h := &handler{}

	hb := builder.NewHandlerBuilder()
	hb.HandleFunc("/openid-configuration", h.OpenIDConfiguration)
	// TODO: changed this to be consistent with Auth0 until we do proper
	// JWKS path discovery.
	hb.HandleFunc("/jwks.json", h.JSONWebKeySet)
	if acmeCfg != nil {
		hb.Handle("/acme-challenge", acmeHandler.NewHandler(acmeCfg, wc))
	}
	return hb.Build()
}

// OpenIDProviderJSON defines the schema for the JSON manifest returned by the OIDC discovery endpoint.
// See https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims for claims
type OpenIDProviderJSON struct {
	Issuer        string   `json:"issuer"`
	AuthURL       string   `json:"authorization_endpoint"`
	TokenURL      string   `json:"token_endpoint"`
	JWKSURL       string   `json:"jwks_uri"`
	UserInfoURL   string   `json:"userinfo_endpoint"`
	Algorithms    []string `json:"id_token_signing_alg_values_supported"`
	SubjectTypes  []string `json:"subject_types_supported"`
	Scopes        []string `json:"scopes_supported"`
	Claims        []string `json:"claims_supported"`
	ResponseTypes []string `json:"response_types_supported"`
}

// OpenIDConfiguration returns the "well-known" OIDC config info
func (h *handler) OpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	// TODO: need to test this with ALB.
	urlScheme := "http://"
	fwdProto := r.Header.Get(headers.XForwardedProto)
	if fwdProto == "https" || fwdProto == "http" {
		urlScheme = fwdProto + "://"
	} else if r.TLS != nil {
		urlScheme = "https://"
	}

	baseURL := strings.TrimSuffix(urlScheme+r.Host, "/")
	providerJSON := OpenIDProviderJSON{
		Issuer:        baseURL,
		AuthURL:       baseURL + "/oidc/authorize",
		TokenURL:      baseURL + "/oidc/token",
		JWKSURL:       baseURL + "/.well-known/jwks.json",
		UserInfoURL:   baseURL + "/oidc/userinfo",
		Algorithms:    []string{"RS256"},
		SubjectTypes:  []string{"public"},
		Scopes:        []string{"openid", "profile"},
		Claims:        []string{"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "name", "email"},
		ResponseTypes: []string{"id_token"},
	}

	jsonapi.Marshal(w, providerJSON)
}

// JSONWebKeySet returns the set of JWKs that we honor tokens from
func (h *handler) JSONWebKeySet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tc := tenantconfig.MustGet(ctx)
	pubKey, err := ucjwt.LoadRSAPublicKey([]byte(tc.Keys.PublicKey))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToLoadPublicKey")
		return
	}

	keyset := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Algorithm: "RS256",
				Key:       pubKey,
				Use:       "sig",
				KeyID:     tc.Keys.KeyID,
			},
		},
	}

	jsonapi.Marshal(w, keyset)
}
