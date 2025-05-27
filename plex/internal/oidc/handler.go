package oidc

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/token"
)

// Handler handles Plex OIDC requests
type Handler struct {
	*uchttp.ServeMux

	factory   provider.Factory
	consolePK *rsa.PublicKey
}

// NewHandler returns an OIDC http handler, along with specific authorize & userinfo handlers to multihome them for auth0 compat
func NewHandler(factory provider.Factory, consolePK *rsa.PublicKey) (http.Handler, func(http.ResponseWriter, *http.Request), func(http.ResponseWriter, *http.Request)) {
	h := &Handler{factory: factory, consolePK: consolePK}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	hb.HandleFunc("/authorize", h.Authorize)
	hb.HandleFunc("/token", h.TokenExchange)
	hb.HandleFunc("/employeetoken", h.m2mEmployeeTokenExchange)
	hb.HandleFunc("/userinfo", h.UserInfoHandler)

	h.ServeMux = hb.Build()

	return h, h.Authorize, h.UserInfoHandler
}

//go:generate genhandler /oidc --public

func validateScopes(scopes []string) error {
	openIDFound := false
	for i := range scopes {
		if scopes[i] == "openid" {
			openIDFound = true
		}
	}
	if !openIDFound {
		return ucerr.Errorf("scope 'openid' not found in OIDC request")
	}

	return nil
}

// parseGrantType parses & validates the "grant_type" request parameter
// and returns a single, valid OIDC grant type.
func parseGrantType(grantTypeParam string) (tenantplex.GrantType, error) {
	grantTypes := oidc.SplitTokens(grantTypeParam)

	if len(grantTypes) == 0 {
		return "", ucerr.NewUnsupportedGrantError("not specified")
	}

	if len(grantTypes) != 1 {
		return "", ucerr.NewUnsupportedGrantError(grantTypeParam)
	}

	grantType := tenantplex.GrantType(grantTypes[0])

	if !tenantplex.SupportedGrantTypes.Contains(grantType) {
		return "", ucerr.NewUnsupportedGrantError(grantTypeParam)
	}

	return grantType, nil
}

// Authorize handles the OIDC authorization request
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	opts := []storage.SessionOption{}

	query := r.URL.Query()
	clientID := query.Get("client_id")
	if len(clientID) == 0 {
		uchttp.ErrorL(ctx, w, ucerr.New("required query parameter 'client_id' missing or malformed"), http.StatusBadRequest, "InvalidClientID")
		return
	}

	// validate that this is a registered client ID
	tc := tenantconfig.MustGet(ctx)
	plexApp, _, err := tc.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.New("invalid client_id"), http.StatusBadRequest)
		return
	}

	redirectURIParam := query.Get("redirect_uri")
	if len(redirectURIParam) == 0 {
		uchttp.ErrorL(ctx, w, ucerr.New("required query parameter 'redirect_uri' missing or malformed"), http.StatusBadRequest, "InvalidRedirect")
		return
	}

	redirectURL, err := plexApp.ValidateRedirectURI(ctx, redirectURIParam)
	if err != nil {
		uchttp.ErrorL(ctx, w, ucerr.Wrap(err), http.StatusBadRequest, "InvalidRedirectURI")
		return
	}

	// The relevant part of the OAuth spec (https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2.1) says:
	//
	// "If the request fails due to a missing, invalid, or mismatching
	// redirection URI, or if the client identifier is missing or invalid,
	// the authorization server SHOULD inform the resource owner of the
	// error and MUST NOT automatically redirect the user-agent to the
	// invalid redirection URI.
	// If the resource owner denies the access request or if the request
	// fails for reasons other than a missing or invalid redirection URI,
	// the authorization server informs the client by adding the following
	// parameters to the query component of the redirection URI using the
	// "application/x-www-form-urlencoded" format, per Appendix B"
	scope := query.Get("scope")
	scopes := oidc.SplitTokens(scope)
	if err := validateScopes(scopes); err != nil {
		uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(err), "InvalidScope")
		return
	}
	responseTypes, err := storage.NewResponseTypes(query.Get("response_type"))
	if err != nil {
		uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(err), "InvalidResponseType")
		return
	}

	// TODO: handle "response_mode" parameter which can be optionally specified to control how the code/tokens are returned.
	// See https://auth0.com/docs/authorization/protocols/protocol-oauth2#authorization-endpoint for useful docs.
	// Default to 'query' for 'code' response_type requests and 'fragment' for 'token' response_type requests, but recommend
	// 'form_post' for token & hybrid flows.
	//responseMode := query.Get("response_mode")

	// Handle "nonce" parameter and ensure it gets encoded in JWTs: https://openid.net/specs/openid-connect-core-1_0.html#IDToken
	nonce := query.Get("nonce")
	if len(nonce) > 0 {
		opts = append(opts, storage.Nonce(nonce))
	}

	state := query.Get("state")
	if len(state) == 0 {
		uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(ucerr.New("required query parameter 'state' missing or malformed")), "InvalidState")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)

	codeChallengeMethod := query.Get("code_challenge_method")
	// Check if using PKCE
	if responseTypes.Contains(storage.AuthorizationCodeResponseType) && len(codeChallengeMethod) > 0 {
		method, err := crypto.NewCodeChallengeMethod(codeChallengeMethod)
		if err != nil {
			uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(err), "InvalidCodeChallengeMethod")
			return
		}

		codeChallenge := query.Get("code_challenge")
		if err := crypto.ValidateCodeChallenge(method, codeChallenge); err != nil {
			uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(err), "InvalidCodeChallenge")
			return
		}

		opts = append(opts, storage.CodeChallenge(ctx, s, method, codeChallenge))
	} else {
		// Fail loudly if some parameters are present when not expected because it means that the client thinks we are using PKCE,
		// but the server does not.
		queryMap := map[string][]string(r.URL.Query())
		if _, ok := queryMap["code_challenge"]; ok {
			uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(ucerr.New("'code_challenge' unexpectedly specified")), "UnexpectedCodeChallenge")
			return
		}
		if _, ok := queryMap["code_challenge_method"]; ok {
			uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewRequestError(ucerr.New("'code_challenge_method' unexpectedly specified")), "UnexpectedCodeChallengeMethod")
			return
		}
	}

	sessionID, err := storage.CreateOIDCLoginSession(ctx, s, clientID, responseTypes, redirectURL, state, strings.Join(scopes, " "), opts...)
	if err != nil {
		uchttp.RedirectOAuthError(w, r, redirectURL, ucerr.NewServerError(err), "FailedLoginSession")
		return
	}

	amc, err := provider.NewActiveClient(ctx, h.factory, clientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	loginURL, err := amc.LoginURL(ctx, sessionID, plexApp)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	uchttp.Redirect(w, r, loginURL.String(), http.StatusFound)
}

// takes both request & postForm because we've already validated postform earlier
// RFC6749 section 2.3 outlines acceptable forms of client authorization, mostly either
// by passing client ID & secret in the POST body, or via Basic Auth
func validateClient(ctx context.Context, r *http.Request, postForm *url.Values) (*tenantplex.App, error) {
	clientID, clientSecret, ok := r.BasicAuth()
	if ok {
		// this is a strange bug I ran into while building a nodejs sample, but TL;DR RFC 6749 says
		// that if you're using basicauth as a client auth method, you must encode the client ID & secret
		// according to application/x-www-form-urlencoded, which means we need to unescape them here
		// https://www.rfc-editor.org/rfc/rfc6749#section-2.3.1
		var err error
		clientID, err = url.QueryUnescape(clientID)
		if err != nil {
			return nil, ucerr.NewInvalidClientError(err)
		}
		clientSecret, err = url.QueryUnescape(clientSecret)
		if err != nil {
			// TODO: why isn't invalid client secret a wrapped error like invalid client error?
			return nil, ucerr.Wrap(ucerr.ErrInvalidClientSecret)
		}
	} else {
		// NB: postForm already un-url-encodes these params for us, so we don't need to do it again
		// (and notably RFC 6749 doesn't specify that they need to be double-encoded, unlike basic auth
		// which needs to be both b64 and urlencoded :eyeroll:)
		clientID = postForm.Get("client_id")
		clientSecret = postForm.Get("client_secret")
	}

	// TODO: validate scope; right now it's unused
	//scope := postForm.Get("scope")

	tc := tenantconfig.MustGet(ctx)
	plexApp, _, err := tc.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		return nil, ucerr.NewInvalidClientError(err)
	}

	cs, err := plexApp.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(ucerr.ErrInvalidClientSecret) // TODO (sgarrity 6/24): technically 500?
	}

	if cs != clientSecret {
		return nil, ucerr.Wrap(ucerr.ErrInvalidClientSecret)
	}

	return plexApp, nil
}

// TokenExchange is the OAuth/OIDC-compliant endpoint for all things related to token exchange.
// At the moment it supports:
// 1. Authorization Code flow (exchange a code for a token).
// 2. Client Credentials flow (exchange client ID + client credentials for a token).
// TODO:
// 3. PKCE ("pixie") support for Authorization Code flow
// 4. Refresh token
func (h *Handler) TokenExchange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseForm()
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(err), "TokenParseError")
		return
	}

	grantType, err := parseGrantType(r.PostForm.Get("grant_type"))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(err), "InvalidGrants")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)

	plexApp, err := validateClient(ctx, r, &r.PostForm)
	if err != nil {
		// Note that 5.2 of RFC6749 prefers 400s except in invalid_client, where 401 MAY be used for
		// form POST auth, and MUST be used for BasicAuth, so we use it for both
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidClient", jsonapi.Code(http.StatusUnauthorized))
		return
	}

	if !plexApp.GrantTypes.Contains(grantType) {
		jsonapi.MarshalError(ctx, w,
			ucerr.Friendlyf(nil, "Client is not authorized to use grant type %s", grantType),
			jsonapi.Code(http.StatusBadRequest))
		return
	}

	uclog.Debugf(ctx, "tokenExchange grant type: %v", grantType)

	switch grantType {
	case tenantplex.GrantTypeAuthorizationCode:
		h.authorizationCodeTokenExchange(w, r, s, &r.PostForm, plexApp)
		return
	case tenantplex.GrantTypeClientCredentials:
		h.clientCredentialsTokenExchange(w, r, s, &r.PostForm, plexApp)
		return
	case tenantplex.GrantTypeRefreshToken:
		h.refreshTokenTokenExchange(w, r, &r.PostForm)
		return
	case tenantplex.GrantTypePassword:
		h.passwordTokenExchange(w, r, s, &r.PostForm, plexApp)
		return
	}

	// This can't be reached but we'll guard against it anyways
	jsonapi.MarshalErrorL(ctx, w, ucerr.New("failed to validate grants"), "FailedToValidateGrants")
}

const m2mEmployeeTokenValidity = 86400 // 24 hours
func (h *Handler) m2mEmployeeTokenExchange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	m2mToken, err := auth.ExtractAccessToken(&r.Header)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToExtractAccessToken")
		return
	}

	tenant := multitenant.MustGetTenantState(ctx)
	if err := m2m.ValidateM2MSecret(ctx, tenant.ID, m2mToken); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToValidateM2MSecret")
		return
	}

	if err := r.ParseForm(); err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(err), "TokenParseError")
		return
	}

	subjectJWT := r.PostForm.Get("subject_jwt")
	if subjectJWT == "" {
		jsonapi.MarshalErrorL(ctx, w, ucerr.New("required parameter 'subject_jwt' missing"), "MissingSubjectJWT")
		return
	}

	tu := tenantconfig.MustGetTenantURLString(ctx)
	audiences := []string{tu}

	// if the actually-used host is different from the primary, include both
	tenantURL := tenant.GetTenantURL()
	if tu != tenantURL {
		audiences = append(audiences, tenantURL)
	}

	// Parse the subject JWT, using the console public key (console tenant is required to have issued the JWT)
	claims, err := ucjwt.ParseUCClaimsVerified(subjectJWT, h.consolePK)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToParseClaims", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// Create a JWT access token (but don't store it to the DB)
	// TODO: should these be stored in the DB as a PlexToken? They are currently each only used for one API call
	// and only by our Console service so there's not much risk of these tokens being abused
	tokenID := uuid.Must(uuid.NewV4())
	tc := tenantconfig.MustGet(ctx)
	accessToken, err := token.CreateAccessTokenJWT(ctx, &tc, tokenID, claims.Subject, claims.SubjectType, claims.OrganizationID, tenantURL, audiences, m2mEmployeeTokenValidity)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGenerateAccessToken")
		return
	}

	jsonapi.Marshal(w, oidc.TokenResponse{
		TokenType:   "Bearer",
		AccessToken: accessToken,
	})
}
