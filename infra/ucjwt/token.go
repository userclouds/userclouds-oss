package ucjwt

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
)

var (
	// TimeFunc is a function that returns the current time. used by JWT to get the current time.
	TimeFunc = time.Now
)

// CreateToken creates a new JWT
func CreateToken(ctx context.Context,
	privateKey *rsa.PrivateKey,
	keyID string,
	tokenID uuid.UUID,
	claims oidc.UCTokenClaims,
	jwtIssuerURL string,
	validFor int64) (string, error) {
	if jwtIssuerURL == "" {
		return "", ucerr.Errorf("jwtIssuerURL must be set")
	}
	// Augment claims with special fields.
	now := time.Now().UTC()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.Issuer = jwtIssuerURL

	// we could use a default value here, but seems more useful to catch data errors
	if validFor == 0 {
		return "", ucerr.Errorf("validFor must be > 0")
	}
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(validFor) * time.Second))
	// Put unique token ID in claims so we can track tokens back to any context around their issuance.
	// As a side effect, we also get unique tokens which is currently required since we want to be able to look
	// up each token issuance uniquely by the token.
	claims.ID = tokenID.String()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	token.Header["kid"] = keyID
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return signedToken, nil
}

// ParseUCClaimsUnverified extracts the claims as UCTokenClaims from a token without validating
// its signature or anything else.
func ParseUCClaimsUnverified(token string) (*oidc.UCTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, ucerr.Errorf("malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ucerr.Errorf("malformed jwt payload: %v", err)
	}
	var claims oidc.UCTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ucerr.Errorf("failed to unmarshal claims: %v", err)
	}

	return &claims, nil
}

// ParseJWTClaimsUnverified extracts the claims as MapClaims from a token without validating
// its signature or anything else.
func ParseJWTClaimsUnverified(token string) (jwt.MapClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, ucerr.Errorf("malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ucerr.Errorf("malformed jwt payload: %v", err)
	}
	var claims jwt.MapClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ucerr.Errorf("failed to unmarshal claims: %v", err)
	}

	return claims, nil
}

// ParseUCClaimsVerified extracts the claims as UCTokenClaims from a token and verifies the signature, expiration, etc.
func ParseUCClaimsVerified(token string, key *rsa.PublicKey) (*oidc.UCTokenClaims, error) {
	var claims oidc.UCTokenClaims
	_, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithTimeFunc(TimeFunc))
	return &claims, ucerr.Wrap(err)
}

// IsExpired returns `true, nil` if the supplied JWT has valid claims and is expired,
// `false, nil` if it has valid claims and is unexpired, and `true, err` if the claims
// aren't parseable.
// NOTE: It does NOT validate the token's signature!
func IsExpired(jwt string) (bool, error) {
	claims, err := ParseUCClaimsUnverified(jwt)
	if err != nil {
		return true, ucerr.Wrap(err)
	}
	if claims.ExpiresAt == nil {
		return true, ucerr.New("jwt has no expiration time")
	}
	if claims.ExpiresAt.After(time.Now().UTC()) {
		return false, nil
	}
	return true, nil
}

// ExtractBearerToken extracts a bearer token from an HTTP request or returns an error
// if none is found or if it's malformed.
// NOTE: this doesn't enforce that it's a JWT, much less a valid one.
func ExtractBearerToken(h *http.Header) (string, error) {
	bearerToken := h.Get(headers.Authorization)
	if bearerToken == "" {
		return "", ucerr.New("authorization header required")
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(bearerToken, bearerPrefix) {
		return "", ucerr.New("authorization header requires bearer token")
	}

	bearerToken = strings.TrimPrefix(bearerToken, bearerPrefix)
	return bearerToken, nil
}
