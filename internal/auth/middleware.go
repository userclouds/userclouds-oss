package auth

import (
	"context"
	"net/http"
	"strings"

	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/multitenant"
	internaloidc "userclouds.com/internal/oidc"
)

type contextKey int

const (
	ctxJWT contextKey = iota // key for a the raw (but verified) JWT parsed from a bearer token
	ctxClaims
	ctxSubjectUUID
	ctxSubjectType
	ctxOrganizationUUID
)

// Verifier provides a mechanism to validate & decode a raw JWT.
type Verifier interface {
	VerifyAndDecode(ctx context.Context, rawJWT string) (*goidc.IDToken, error)
}

var externalIssuerPaths = set.NewStringSet("/userstore/api/mutators", "/userstore/api/accessors", "/tokenizer/tokens/actions/resolve")

// AddTokenToContext extracts the claims, subject, and organization from the token and adds them to the context (including the raw token)
func AddTokenToContext(ctx context.Context, token string, jwtVerifier Verifier, onlyAllowUCIssued bool) (context.Context, error) {

	idToken, err := jwtVerifier.VerifyAndDecode(ctx, token)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	claims := oidc.UCTokenClaims{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// save both the raw JWT and the parsed claims
	ctx = context.WithValue(
		context.WithValue(ctx, ctxJWT, token),
		ctxClaims,
		claims)

	if internaloidc.IsUsercloudsIssued(idToken.Issuer) {
		subject, err := uuid.FromString(claims.Subject)
		if err != nil {
			subject = uuid.Nil
		}
		ctx = context.WithValue(context.WithValue(ctx, ctxSubjectType, claims.SubjectType), ctxSubjectUUID, subject)

		organizationID, err := uuid.FromString(claims.OrganizationID)
		if err != nil {
			organizationID = uuid.Nil
		}
		ctx = context.WithValue(ctx, ctxOrganizationUUID, organizationID)

		// TODO (sgarrity 8/23): once we have full JSON-payload logging, we can just attach this to each event
		uclog.Debugf(ctx, "JWT subject=%v organizaton=%v", subject, organizationID)
	} else if onlyAllowUCIssued {
		uclog.Warningf(ctx, "Unauthorized call using JWT with issuer: %s", idToken.Issuer)
		return nil, ucerr.Errorf("Unauthorized call using JWT with issuer: %s", idToken.Issuer)
	}

	return ctx, nil
}

// Middleware is used where a JWT bearer token is required. It ensures that
// one is present and adds it to the request context.
func Middleware(jwtVerifier Verifier, consoleTenantID uuid.UUID) middleware.Middleware {
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			token, err := ucjwt.ExtractBearerToken(&r.Header)
			if err == nil {
				onlyAllowUCIssued := false
				// TODO (jwang 2/24): This is crude but effective for now. Longer term solution would be to do checks in each of the roleBasedAuthorizers, but currently
				// a bunch of those are no-ops (e.g. both authz's and idp's handler.ensureTenantMember)
				requestURIParts := strings.Split(r.RequestURI, "?")
				if !externalIssuerPaths.Contains(requestURIParts[0]) || r.Method != "POST" {
					onlyAllowUCIssued = true
				}

				ctxToken, err := AddTokenToContext(ctx, token, jwtVerifier, onlyAllowUCIssued)
				if err != nil {
					uchttp.ErrorL(ctx, w, err, http.StatusUnauthorized, "BearerTokenFail")
					return
				}

				next.ServeHTTP(w, r.WithContext(ctxToken))
				return
			}

			token, err = ExtractAccessToken(&r.Header)
			if err == nil {
				ts := multitenant.GetTenantState(ctx)
				if ts == nil {
					// can't use the access token without a tenant ... as usual, logserver makes this hard :)
					uchttp.Error(ctx, w, ucerr.New("no tenant in context"), http.StatusUnauthorized)
					return
				}

				tenantID := ts.ID
				if err := m2m.ValidateM2MSecret(ctx, tenantID, token); err != nil {
					// allow console to bypass this check as usual
					if tenantID != consoleTenantID {
						if err := m2m.ValidateM2MSecret(ctx, consoleTenantID, token); err != nil {
							uchttp.ErrorL(ctx, w, ucerr.Wrap(err), http.StatusUnauthorized, "AccessTokenFail")
							return
						}
					} else {
						uchttp.ErrorL(ctx, w, ucerr.Wrap(err), http.StatusUnauthorized, "AccessTokenFail")
						return
					}
				}

				ctx = context.WithValue(
					context.WithValue(context.WithValue(
						ctx,
						ctxSubjectType,
						m2m.SubjectTypeM2M),
						ctxOrganizationUUID,
						ts.CompanyID),
					ctxSubjectUUID,
					tenantID)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// TODO: return JSON error?
			uchttp.ErrorL(ctx, w, ucerr.Wrap(err), http.StatusUnauthorized, "TokenFail")
		})
	}))
}

// ExtractAccessToken returns the "Authorization: AccessToken <token>" header value or error
func ExtractAccessToken(header *http.Header) (string, error) {
	accessToken := header.Get("Authorization")
	if accessToken == "" {
		return "", ucerr.Errorf("no authorization header")
	}

	parts := strings.Split(accessToken, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "accesstoken" {
		return "", ucerr.Errorf("invalid access token format")
	}

	return parts[1], nil
}

// GetAuditLogActor returns a string for the audit log
func GetAuditLogActor(ctx context.Context) string {
	actor := GetSubjectUUID(ctx)
	if actor.IsNil() {
		subject := ctx.Value(ctxSubjectUUID)
		uclog.Warningf(ctx, "subject in context not a valid uuid: '%v' (type: %T)", subject, subject)
		return MustGetClaims(ctx).Subject
	}
	return actor.String()
}

// GetSubjectUUID returns the subject from the context, assuming it is a UUID, uuid.Nil otherwise
func GetSubjectUUID(ctx context.Context) uuid.UUID {
	val := ctx.Value(ctxSubjectUUID)
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// GetOrganizationUUID returns the organization ID from the context, assuming it is a UUID, uuid.Nil otherwise
func GetOrganizationUUID(ctx context.Context) uuid.UUID {
	val := ctx.Value(ctxOrganizationUUID)
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// GetSubjectType returns the subject type from the context, assuming it is a string, "" otherwise
func GetSubjectType(ctx context.Context) string {
	val := ctx.Value(ctxSubjectType)
	subjectType, ok := val.(string)
	if !ok {
		return ""
	}
	return subjectType
}

// GetRawJWT extracts the ID token from the context, which is parsed & validated
// by middleware in the prologue of the request.
func GetRawJWT(ctx context.Context) string {
	val := ctx.Value(ctxJWT)
	token, ok := val.(string)
	if !ok {
		return ""
	}
	return token
}

// MustGetClaims extracts the claims from the context
func MustGetClaims(ctx context.Context) oidc.UCTokenClaims {
	val := ctx.Value(ctxClaims)
	claims, ok := val.(oidc.UCTokenClaims)
	if !ok {
		panic(ucerr.New("claims not set in context, did you forget to use auth.Middleware?"))
	}
	return claims
}

// SetSubjectTypeAndUUID sets the subject type and UUID in the context -- this is only used for situations
// where the subject is not going to be available from a JWT, but we still want to be able to, e.g., write to audit log
func SetSubjectTypeAndUUID(ctx context.Context, subjectID uuid.UUID, subjectType string) context.Context {
	return context.WithValue(context.WithValue(ctx, ctxSubjectUUID, subjectID), ctxSubjectType, subjectType)
}
