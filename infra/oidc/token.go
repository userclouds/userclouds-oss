package oidc

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/infra/ucerr"
)

// StandardClaims is forked from golang-jwt/jwt.StandardClaims,
// except Audience is an array here per the actual spec:
//
//	In the general case, the "aud" value is an array of case-sensitive strings, each containing
//	a StringOrURI value.  In the special case when the JWT has one audience, the "aud" value MAY
//	be a single case-sensitive string containing a StringOrURI value.  The interpretation of
//	audience values is generally application specific. Use of this claim is OPTIONAL.
//
// https://tools.ietf.org/html/rfc7519#section-4.1
//
// AZP is also added here, per the OIDC spec, which is slightly ambiguous:
//
// From 2 https://openid.net/specs/openid-connect-core-1_0.html#IDToken:
// OPTIONAL. Authorized party - the party to which the ID Token was issued.
// If present, it MUST contain the OAuth 2.0 Client ID of this party. This Claim
// is only needed when the ID Token has a single audience value and that audience
// is different than the authorized party. It MAY be included even when the
// authorized party is the same as the sole audience. The azp value is a case
// sensitive string containing a StringOrURI value.
//
// From 3.1.3.7 https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation
// 4. If the ID Token contains multiple audiences, the Client SHOULD verify that an azp Claim is present.
// 5. If an azp (authorized party) Claim is present, the Client SHOULD verify that its client_id is the Claim Value.
type StandardClaims struct {
	jwt.RegisteredClaims
	Audience        []string `json:"aud,omitempty" yaml:"aud,omitempty"`
	AuthorizedParty string   `json:"azp,omitempty" yaml:"azp,omitempty"`
}

// private class to allow audience to be an array or a string
type standardClaims struct {
	jwt.RegisteredClaims
	Audience        any    `json:"aud,omitempty" yaml:"aud,omitempty"`
	AuthorizedParty string `json:"azp,omitempty" yaml:"azp,omitempty"`
}

// UCTokenClaims represents the UserClouds claims made by a token, and is also used by the UserInfo
// endpoint to return standard OIDC user claims.
type UCTokenClaims struct {
	StandardClaims

	Name            string   `json:"name,omitempty" yaml:"name,omitempty"`
	Nickname        string   `json:"nickname,omitempty" yaml:"nickname,omitempty"`
	Email           string   `json:"email,omitempty" yaml:"email,omitempty"`
	EmailVerified   bool     `json:"email_verified,omitempty" yaml:"email_verified,omitempty"`
	Picture         string   `json:"picture,omitempty" yaml:"picture,omitempty"`
	Nonce           string   `json:"nonce,omitempty" yaml:"nonce,omitempty"`
	UpdatedAt       int64    `json:"updated_at,omitempty" yaml:"updated_at,omitempty"` // NOTE: Auth0 treats this as a string, but OIDC says this is seconds since the Unix Epoch
	RefreshAudience []string `json:"refresh_aud,omitempty" yaml:"refresh_aud,omitempty"`
	SubjectType     string   `json:"subject_type,omitempty" yaml:"subject_type,omitempty"`
	OrganizationID  string   `json:"organization_id,omitempty" yaml:"organization_id,omitempty"`
	ImpersonatedBy  string   `json:"impersonated_by,omitempty" yaml:"impersonated_by,omitempty"`
}

// private class needed for json unmarshal
type tokenclaims struct {
	standardClaims

	Name            string   `json:"name,omitempty" yaml:"name,omitempty"`
	Nickname        string   `json:"nickname,omitempty" yaml:"nickname,omitempty"`
	Email           string   `json:"email,omitempty" yaml:"email,omitempty"`
	EmailVerified   bool     `json:"email_verified,omitempty" yaml:"email_verified,omitempty"`
	Picture         string   `json:"picture,omitempty" yaml:"picture,omitempty"`
	Nonce           string   `json:"nonce,omitempty" yaml:"nonce,omitempty"`
	UpdatedAt       int64    `json:"updated_at,omitempty" yaml:"updated_at,omitempty"` // NOTE: Auth0 treats this as a string, but OIDC says this is seconds since the Unix Epoch
	RefreshAudience []string `json:"refresh_aud,omitempty" yaml:"refresh_aud,omitempty"`
	SubjectType     string   `json:"subject_type,omitempty" yaml:"subject_type,omitempty"`
	OrganizationID  string   `json:"organization_id,omitempty" yaml:"organization_id,omitempty"`
	ImpersonatedBy  string   `json:"impersonated_by,omitempty" yaml:"impersonated_by,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler, we need this to handle the audience field being either an array or a string
func (t *UCTokenClaims) UnmarshalJSON(b []byte) error {
	var tc tokenclaims
	if err := json.Unmarshal(b, &tc); err != nil {
		return ucerr.Wrap(err)
	}

	audience := []string{}
	if arr, ok := tc.Audience.([]any); ok {
		for _, a := range arr {
			if s, ok := a.(string); ok {
				audience = append(audience, s)
			}
		}
	} else if s, ok := tc.Audience.(string); ok {
		audience = append(audience, s)
	}

	t.Audience = audience
	t.AuthorizedParty = tc.AuthorizedParty
	t.ExpiresAt = tc.ExpiresAt
	t.ID = tc.ID
	t.IssuedAt = tc.IssuedAt
	t.Issuer = tc.Issuer
	t.NotBefore = tc.NotBefore
	t.Subject = tc.Subject
	t.Name = tc.Name
	t.Nickname = tc.Nickname
	t.Email = tc.Email
	t.EmailVerified = tc.EmailVerified
	t.Picture = tc.Picture
	t.Nonce = tc.Nonce
	t.UpdatedAt = tc.UpdatedAt
	t.RefreshAudience = tc.RefreshAudience
	t.SubjectType = tc.SubjectType
	t.OrganizationID = tc.OrganizationID
	t.ImpersonatedBy = tc.ImpersonatedBy
	return nil
}

// TokenResponse is an OIDC-compliant response from a token endpoint.
// (either token exchange or resource owner password credential flow).
// See https://datatracker.ietf.org/doc/html/rfc6749#section-5.1.
// ErrorType will be non-empty if error.
type TokenResponse struct {
	AccessToken  string `json:"access_token,omitempty" yaml:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty" yaml:"token_type,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty" yaml:"expires_in,omitempty"`
	IDToken      string `json:"id_token,omitempty" yaml:"id_token,omitempty"`

	ErrorType string `json:"error,omitempty" yaml:"error,omitempty"`
	ErrorDesc string `json:"error_description,omitempty" yaml:"error_description,omitempty"`
}
