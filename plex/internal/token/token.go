package token

import (
	"context"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
)

// ExtractClaimsFromJWT extracts claims from a JWT and verifies signatures, etc.
func ExtractClaimsFromJWT(ctx context.Context, idpURL, clientID, rawIDToken string) (jwt.MapClaims, error) {
	provider, err := gooidc.NewProvider(ctx, idpURL)
	if err != nil {
		uclog.Debugf(ctx, "failed to get provider from URL '%s', error: %v", idpURL, err)
		return nil, ucerr.Wrap(err)
	}

	oidcConfig := &gooidc.Config{ClientID: clientID}
	idToken, err := provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
	if err != nil {
		uclog.Debugf(ctx, "failed to get verify ID token from IDP URL '%s', error: %v", idpURL, err)
		return nil, ucerr.Wrap(err)
	}

	var claims jwt.MapClaims
	if err := idToken.Claims(&claims); err != nil {
		uclog.Debugf(ctx, "failed to extract claims from ID token from IDP URL '%s', error: %v", idpURL, err)
		return nil, ucerr.Wrap(err)
	}

	return claims, nil
}

// CreateAccessTokenJWT creates & signs a JWT for an access token. Used by M2M as well as user-centric tokens.
func CreateAccessTokenJWT(ctx context.Context, tc *tenantplex.TenantConfig, tokenID uuid.UUID, subject string, subjectType string, organizationID string, issuer string, audiences []string, validFor int64) (string, error) {
	// NOTE: most other user-centric claims (as well as 'nonce') are not applicable here.
	claims := oidc.UCTokenClaims{
		StandardClaims: oidc.StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: subject},
			Audience:         audiences,
		},
		SubjectType:    subjectType,
		OrganizationID: organizationID,
	}

	keyText, err := tc.Keys.PrivateKey.Resolve(ctx)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	privKey, err := ucjwt.LoadRSAPrivateKey([]byte(keyText))
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	j, err := ucjwt.CreateToken(ctx, privKey, tc.Keys.KeyID, tokenID, claims, issuer, validFor) // TODO check MR support
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return j, nil
}

// CreateRefreshTokenJWT creates & signs a JWT for a refresh token. Looks identical to Access token except 'aud' claim is moved to 'refresh_aud'
// so this token can't be used as a regular access token but can be used to generate an access token with that audience.
func CreateRefreshTokenJWT(ctx context.Context, tc *tenantplex.TenantConfig, tokenID uuid.UUID, subject string, subjectType string, organizationID string, audiences []string, validFor int64) (string, error) {
	claims := oidc.UCTokenClaims{
		StandardClaims: oidc.StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: subject},
		},
		SubjectType:     subjectType,
		OrganizationID:  organizationID,
		RefreshAudience: audiences,
	}

	keyText, err := tc.Keys.PrivateKey.Resolve(ctx)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	privKey, err := ucjwt.LoadRSAPrivateKey([]byte(keyText))
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	// we issue tokens using the URL that the user used to access the tenant, so that we can use the same URL to verify the token
	ts := multitenant.MustGetTenantState(ctx)

	j, err := ucjwt.CreateToken(ctx, privKey, tc.Keys.KeyID, tokenID, claims, ts.GetTenantURL(), validFor)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return j, nil
}
