package oidc

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/plex/internal/tenantconfig"
)

func (h *Handler) refreshTokenTokenExchange(w http.ResponseWriter, r *http.Request, postForm *url.Values) {
	ctx := r.Context()
	tc := tenantconfig.MustGet(ctx)
	tu := tenantconfig.MustGetTenantURLString(ctx)

	refreshToken := (*postForm).Get("refresh_token")
	if refreshToken == "" {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("No refresh token provided"), "NoRefreshToken", jsonapi.Code(http.StatusBadRequest))
		return
	}

	pk, err := ucjwt.LoadRSAPublicKey([]byte(tc.Keys.PublicKey))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToLoadPublicKey", jsonapi.Code(http.StatusBadRequest))
		return
	}

	keyText, err := tc.Keys.PrivateKey.Resolve(ctx)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToFindPrivateKey", jsonapi.Code(http.StatusBadRequest))
		return
	}

	privKey, err := ucjwt.LoadRSAPrivateKey([]byte(keyText))
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToLoadPrivateKey", jsonapi.Code(http.StatusBadRequest))
		return
	}

	claims, err := ucjwt.ParseUCClaimsVerified(refreshToken, pk)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToParseClaims", jsonapi.Code(http.StatusBadRequest))
		return
	}

	tokenID, err := uuid.FromString(claims.ID) // grab 'jti' so that we can re-use it in the new token
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidJTI", jsonapi.Code(http.StatusBadRequest))
		return
	}

	app, err := validateClient(ctx, r, postForm)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "InvalidClient", jsonapi.Code(http.StatusUnauthorized))
		return
	}

	// don't let the new token expire later than the refresh token
	validFor := min(app.TokenValidity.Access, int64(claims.ExpiresAt.Sub(time.Now().UTC()).Seconds()))
	claims.Audience = claims.RefreshAudience // move the refresh audience to the audience
	claims.RefreshAudience = nil

	accessToken, err := ucjwt.CreateToken(ctx, privKey, tc.Keys.KeyID, tokenID, *claims, tu, validFor)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToCreateToken", jsonapi.Code(http.StatusBadRequest))
		return
	}

	response := oidc.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}

	jsonapi.Marshal(w, response)
}
