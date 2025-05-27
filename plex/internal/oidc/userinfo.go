package oidc

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
)

// UserInfoHandler handles /userinfo
func (h Handler) UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bearerToken, err := ucjwt.ExtractBearerToken(&r.Header)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusUnauthorized))
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	token, err := s.GetPlexTokenForAccessToken(ctx, bearerToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	// return a skeleton when used with client credentials
	// this is undefined behavior in the spec so shouldn't hurt anything
	// https://openid.net/specs/openid-connect-core-1_0.html#UserInfo
	if token.IDPSubject == "" {
		pc := tenantconfig.MustGet(ctx)
		app, _, err := pc.PlexMap.FindAppForClientID(token.ClientID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}

		response := oidc.UCTokenClaims{
			StandardClaims: oidc.StandardClaims{
				RegisteredClaims: jwt.RegisteredClaims{Subject: token.ClientID}},
			Name:        app.Name,
			SubjectType: "client",
		}

		jsonapi.Marshal(w, response)
		return
	}

	// Service the user info request from the primary IDP'S data store. TODO: fallback to follower if needed.
	mgmtClient, err := provider.NewActiveManagementClient(ctx, h.factory, token.ClientID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	user, err := mgmtClient.GetUser(ctx, token.IDPSubject)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tu := tenantconfig.MustGetTenantURLString(ctx)

	scopes := oidc.SplitTokens(token.Scopes)
	var response oidc.UCTokenClaims
	for i := range scopes {
		switch scopes[i] {
		case "openid":
			response.Subject = user.ID
			response.Audience = []string{token.ClientID, tu} // TODO: shouldn't we load these from somewhere?
			response.Issuer = tu                             // TODO: check MR support
			response.UpdatedAt = user.UpdatedAt
		case "profile":
			response.Name = user.Name
			response.Nickname = user.Nickname
			response.Picture = user.Picture
		case "email":
			response.Email = user.Email
			response.EmailVerified = user.EmailVerified
			// TODO: support others
		}
	}

	jsonapi.Marshal(w, response)
}
