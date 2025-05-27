package storage

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/authz"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/token"
)

// GenerateUserPlexToken generates a user-centric set of tokens (ID token, access token) and stores it in the DB.
func GenerateUserPlexToken(ctx context.Context,
	tenantURL string, // tenant.TenantURL, not the actually-used host
	tc *tenantplex.TenantConfig,
	storage *Storage,
	profile *iface.UserProfile,
	session *OIDCLoginSession,
	underlyingToken *oidc.TokenInfo) error {
	return ucerr.Wrap(GenerateImpersonatedUserPlexToken(ctx, tenantURL, tc, storage, profile, session, underlyingToken, ""))
}

// GenerateImpersonatedUserPlexToken generates a user-centric set of tokens etc, but on behalf of someone else
// TODO: there's probably a better factoring, but this is the minimal diff set of changes
func GenerateImpersonatedUserPlexToken(ctx context.Context,
	tenantURL string, // tenant.TenantURL, not the actually-used host
	tc *tenantplex.TenantConfig,
	storage *Storage,
	profile *iface.UserProfile,
	session *OIDCLoginSession,
	underlyingToken *oidc.TokenInfo,
	impersonator string) error {

	claims := oidc.UCTokenClaims{
		StandardClaims: oidc.StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: profile.ID},
			Audience:         []string{session.ClientID, tenantURL},
			AuthorizedParty:  session.ClientID, // https://stackoverflow.com/questions/41231018/openid-connect-standard-authorized-party-azp-contradiction
		},
		Name:           profile.Name,
		Nickname:       profile.Nickname,
		Email:          profile.Email,
		EmailVerified:  profile.EmailVerified,
		Picture:        profile.Picture,
		Nonce:          session.Nonce,
		OrganizationID: profile.OrganizationID,
		SubjectType:    authz.ObjectTypeUser,
		ImpersonatedBy: impersonator,
	}

	// NB: (and maybe TODO) we can't use the TenantURL from the plex tenant config, since that is
	// currently normalized to the primary. Instead we need to grab the actually-used tenant URL
	// for this request from the context so we'll issue a JWT for eg. auth.foo.com instead of foo.tenant.userclouds.com
	iss := multitenant.MustGetTenantState(ctx).GetTenantURL()

	// we always include the primary one but will also include the actual tenant URL if it's different
	if iss != tenantURL {
		claims.StandardClaims.Audience = append(claims.StandardClaims.Audience, iss)
	}

	plexApp, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	idValidFor := plexApp.TokenValidity.Access
	accessValidFor := plexApp.TokenValidity.Access
	refreshValidFor := plexApp.TokenValidity.Refresh

	if underlyingToken != nil && impersonator != "" {
		idValidFor = plexApp.TokenValidity.ImpersonateUser
		accessValidFor = plexApp.TokenValidity.ImpersonateUser
		refreshValidFor = plexApp.TokenValidity.ImpersonateUser
	}

	keyText, err := tc.Keys.PrivateKey.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	privKey, err := ucjwt.LoadRSAPrivateKey([]byte(keyText))
	if err != nil {
		return ucerr.Wrap(err)
	}
	tokenID := uuid.Must(uuid.NewV4())
	rawIDToken, err := ucjwt.CreateToken(ctx, privKey, tc.Keys.KeyID, tokenID, claims, iss, idValidFor) // TODO: check MR support
	if err != nil {
		return ucerr.Wrap(err)
	}

	var uts string
	if underlyingToken != nil {
		bs, err := json.Marshal(underlyingToken)
		if err != nil {
			return ucerr.Wrap(err)
		}
		uts = string(bs)
	}
	accessToken, err := token.CreateAccessTokenJWT(ctx, tc, tokenID, profile.ID, authz.UserObjectTypeID.String(), profile.OrganizationID, iss, []string{iss}, accessValidFor)
	if err != nil {
		return ucerr.Wrap(err)
	}
	refreshToken, err := token.CreateRefreshTokenJWT(ctx, tc, tokenID, profile.ID, authz.UserObjectTypeID.String(), profile.OrganizationID, []string{iss}, refreshValidFor)
	if err != nil {
		return ucerr.Wrap(err)
	}

	plexToken := &PlexToken{
		BaseModel:       ucdb.NewBaseWithID(tokenID),
		AuthCode:        crypto.GenerateOpaqueAccessToken(),
		ClientID:        session.ClientID,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		IDToken:         rawIDToken,
		IDPSubject:      profile.ID,
		Scopes:          session.Scopes,
		SessionID:       session.ID,
		UnderlyingToken: uts,
	}

	if err := storage.SavePlexToken(ctx, plexToken); err != nil {
		return ucerr.Wrap(err)
	}

	session.PlexTokenID = plexToken.ID
	if err := storage.SaveOIDCLoginSession(ctx, session); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// GenerateUserPlexTokenWithoutSession generates a user-centric set of tokens (access token, refresh token) and stores it in the DB.
// This is specifically used for the ROPC flow (password grant type) where there's no real "login session" to speak of.
func GenerateUserPlexTokenWithoutSession(ctx context.Context,
	tc *tenantplex.TenantConfig,
	s *Storage,
	profile *iface.UserProfile,
	scopes []string,
	plexApp *tenantplex.App) (*PlexToken, error) {

	tokenID := uuid.Must(uuid.NewV4())

	// NB: (and maybe TODO) we can't use the TenantURL from the plex tenant config, since that is
	// currently normalized to the primary. Instead we need to grab the actually-used tenant URL
	// for this request from the context so we'll issue a JWT for eg. auth.foo.com instead of foo.tenant.userclouds.com
	iss := multitenant.MustGetTenantState(ctx).GetTenantURL()

	accessToken, err := token.CreateAccessTokenJWT(ctx, tc, tokenID, profile.ID, authz.UserObjectTypeID.String(), profile.OrganizationID, iss, []string{iss}, plexApp.TokenValidity.Access)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	refreshToken, err := token.CreateRefreshTokenJWT(ctx, tc, tokenID, profile.ID, authz.UserObjectTypeID.String(), profile.OrganizationID, []string{iss}, plexApp.TokenValidity.Refresh)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	plexToken := &PlexToken{
		BaseModel:    ucdb.NewBaseWithID(tokenID),
		AuthCode:     crypto.GenerateOpaqueAccessToken(),
		ClientID:     plexApp.ClientID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDPSubject:   profile.ID,
		Scopes:       strings.Join(scopes, ","),
		SessionID:    NonInteractiveSessionID,
	}

	if err := s.SavePlexToken(ctx, plexToken); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return plexToken, nil
}

// GenerateM2MPlexToken generates a non-user-centric set of ID & Access tokens for M2M auth.
func GenerateM2MPlexToken(ctx context.Context, tc *tenantplex.TenantConfig, storage *Storage, clientID, scope string, audience []string) (*PlexToken, error) {
	plexApp, _, err := tc.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ts := multitenant.MustGetTenantState(ctx)

	tokenID := uuid.Must(uuid.NewV4())
	accessToken, err := token.CreateAccessTokenJWT(ctx, tc, tokenID, plexApp.ID.String(), authz.ObjectTypeLoginApp, plexApp.OrganizationID.String(), ts.GetTenantURL(), audience, plexApp.TokenValidity.Access)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	refreshToken, err := token.CreateRefreshTokenJWT(ctx, tc, tokenID, plexApp.ID.String(), authz.ObjectTypeLoginApp, plexApp.OrganizationID.String(), audience, plexApp.TokenValidity.Refresh)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Auth code isn't used, but we need a unique value here because of the unique constraint on the column.
	authCode := crypto.GenerateOpaqueAccessToken()

	plexToken := &PlexToken{
		BaseModel:    ucdb.NewBaseWithID(tokenID),
		AuthCode:     authCode,
		ClientID:     clientID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      "",
		IDPSubject:   "",
		Scopes:       scope,
		SessionID:    NonInteractiveSessionID,
	}

	if err := storage.SavePlexToken(ctx, plexToken); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return plexToken, nil
}
