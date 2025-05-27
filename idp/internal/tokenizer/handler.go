package tokenizer

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/paths"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/security"
	logServerClient "userclouds.com/logserver/client"
)

// NewHandler returns a new tokenizer handler
func NewHandler(m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo, enableLogServer bool) (*uchttp.ServeMux, error) {
	h := &handler{}
	if enableLogServer {
		lsc, err := logServerClient.NewClientForTenantAuth(consoleTenantInfo.TenantURL, consoleTenantInfo.TenantID, m2mAuth, security.PassXForwardedFor())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		h.logServerClient = lsc
	}
	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)
	hb.HandleFunc("/dbstats", h.dbStatsHandler)
	hb.HandleFunc(paths.StripTokenizerBase(paths.TestAccessPolicy), h.testAccessPolicy)
	hb.HandleFunc(paths.StripTokenizerBase(paths.TestAccessPolicyTemplate), h.testAccessPolicyTemplate)
	hb.HandleFunc(paths.StripTokenizerBase(paths.TestTransformer), h.testTransformer)
	return hb.Build(), nil
}

//go:generate genhandler /tokenizer POST,inspectToken,/tokens/actions/inspect POST,lookupTokens,/tokens/actions/lookup POST,lookupOrCreateTokens,/tokens/actions/lookuporcreate POST,resolveToken,/tokens/actions/resolve method,Token,/tokens collection,AccessPolicyTemplate,h.newAccessPolicyTemplateAuthorizer(),/policies/accesstemplate collection,AccessPolicy,h.newTokenizerAuthorizer(),/policies/access collection,Transformer,h.newTokenizerAuthorizer(),/policies/transformation method,Token,/tokens collection,Secret,h.newSecretAuthorizer(),/policies/secret

type handler struct {
	logServerClient *logServerClient.Client
}

// dbStatsHandler returns a simple http.Handler to return the db stats
// NB: requires use of the multitenant.Middleware
func (h *handler) dbStatsHandler(w http.ResponseWriter, r *http.Request) {
	ts := multitenant.MustGetTenantState(r.Context())
	jsonapi.Marshal(w, ts.TenantDB.Stats())
}

func (h handler) newAccessPolicyTemplateAuthorizer() uchttp.CollectionAuthorizer {
	return uchttp.NewAllowAllAuthorizer()
}

func (h handler) newSecretAuthorizer() uchttp.CollectionAuthorizer {
	return uchttp.NewAllowAllAuthorizer()
}

func (h handler) newTokenizerAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.canUserAccessPolicy(r, idpAuthz.PoliciesObjectID, idpAuthz.AttributePolicyRead))
		},
		GetOneF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(h.canUserAccessPolicy(r, id, idpAuthz.AttributePolicyRead))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.canUserAccessPolicy(r, idpAuthz.PoliciesObjectID, idpAuthz.AttributePolicyCreate))
		},
		PutF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(h.canUserAccessPolicy(r, id, idpAuthz.AttributePolicyUpdate))
		},
		DeleteF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(h.canUserAccessPolicy(r, id, idpAuthz.AttributePolicyDelete))
		},
	}
}

// canUserAccessPolicy returns nil if the user in the context has access to the policy, or an error if not
func (h handler) canUserAccessPolicy(req *http.Request, policyID uuid.UUID, attributeName string) error {
	ctx := req.Context()

	subjectType := auth.GetSubjectType(ctx)
	if subjectType == authz.ObjectTypeLoginApp || subjectType == m2m.SubjectTypeM2M {
		return nil
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)

	if err != nil {
		return ucerr.Wrap(err)
	}

	subjectID := auth.GetSubjectUUID(ctx)
	if subjectID.IsNil() {
		// TODO: this shouldn't happen, except maybe for tests
		uclog.Errorf(ctx, "no subject id found in context, skipping authz check")
		return nil
	}

	object, err := authzClient.GetObject(ctx, subjectID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if object.TypeID == authz.LoginAppObjectTypeID {
		// TODO: implement permissions for login apps
		return nil
	}

	var r *authz.CheckAttributeResponse

	// Look for a path from the user to the specified policy granting the user full access
	if r, err = authzClient.CheckAttribute(ctx, subjectID, policyID, attributeName); err != nil {
		return ucerr.Wrap(err)
	}
	if r.HasAttribute {
		return nil
	}
	return ucerr.Friendlyf(nil, "user not authorized")
}
