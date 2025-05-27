package loginapp

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex"
	"userclouds.com/plex/manager"
)

type handler struct {
}

// NewHandler returns a new loginapp handler for plex
func NewHandler() http.Handler {
	h := &handler{}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	return hb.Build()
}

//go:generate genhandler /loginapp collection,LoginApp,h.newRoleBasedAuthorizer(),/register

func (h *handler) newRoleBasedAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetOneF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantAdmin(r))
		},
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantAdmin(r))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantAdmin(r))
		},
		PutF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantAdmin(r))
		},
		DeleteF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantAdmin(r))
		},
	}
}

func (h *handler) ensureTenantAdmin(r *http.Request) error {

	ctx := r.Context()

	if subjectType := auth.GetSubjectType(ctx); subjectType == authz.ObjectTypeLoginApp {
		return ucerr.Friendlyf(nil, "Must use credentials of a user for this endpoint, not a login app")
	}

	ts := multitenant.MustGetTenantState(ctx)

	authzClient, err := authz.NewClient(ts.GetTenantURL(), authz.PassthroughAuthorization(), authz.JSONClient(security.PassXForwardedFor()))
	if err != nil {
		return ucerr.Wrap(err)
	}

	subjectID := auth.GetSubjectUUID(ctx)
	if subjectID.IsNil() {
		return ucerr.Friendlyf(nil, "No subject ID available for verification")
	}

	resp, err := authzClient.CheckAttribute(ctx, subjectID, ts.CompanyID, ucauthz.AdminRole)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if !resp.HasAttribute {
		return ucerr.Friendlyf(nil, "User is not an admin of the tenant")
	}

	return nil
}

// OpenAPI Summary: Create Login Application
// OpenAPI Tags: Login Application
// OpenAPI Description: This endpoint creates a new login application for the tenant. Only tenant admins can create login applications.
func (h *handler) createLoginApp(ctx context.Context, req plex.LoginAppRequest) (*plex.LoginAppResponse, int, []auditlog.Entry, error) {

	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	id := uuid.Must(uuid.NewV4())
	cs, err := crypto.GenerateClientSecret(ctx, id.String())
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	app := tenantplex.App{
		ID:             id,
		OrganizationID: ts.CompanyID, // TODO: enable organization ID to be specified in the request
		Name:           req.ClientName,
		ClientID:       crypto.MustRandomHex(32),
		ClientSecret:   *cs,
		GrantTypes:     []tenantplex.GrantType{tenantplex.GrantTypeAuthorizationCode, tenantplex.GrantTypeRefreshToken, tenantplex.GrantTypeClientCredentials},
	}

	authzClient, err := authz.NewClient(ts.GetTenantURL(), authz.PassthroughAuthorization(), authz.JSONClient(security.PassXForwardedFor()))
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := mgr.AddLoginApp(ctx, ts.ID, authzClient, app); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp, err := loginAppResponseFromApp(ctx, &app)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	resp.ClientIDIssuedAt = time.Now().UTC()

	return &resp, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Update Login Application
// OpenAPI Tags: Login Application
// OpenAPI Description: This endpoint updates a login application for the tenant. Only tenant admins can update login applications.
func (h *handler) updateLoginApp(ctx context.Context, id uuid.UUID, req plex.LoginAppRequest) (*plex.LoginAppResponse, int, []auditlog.Entry, error) {
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	app, err := mgr.GetLoginApp(ctx, ts.ID, id)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	app.Name = req.ClientName
	app.AllowedRedirectURIs = req.RedirectURIs
	app.GrantTypes = []tenantplex.GrantType{}
	for _, g := range req.GrantTypes {
		app.GrantTypes = append(app.GrantTypes, tenantplex.GrantType(g))
	}

	if err := mgr.UpdateLoginApp(ctx, ts.ID, *app); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resp, err := loginAppResponseFromApp(ctx, app)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &resp, http.StatusOK, nil, nil
}

// OpenAPI Summary: Delete Login Application
// OpenAPI Tags: Login Application
// OpenAPI Description: This endpoint deletes a login application for the tenant. Only tenant admins can delete login applications.
func (h *handler) deleteLoginApp(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	authzClient, err := authz.NewClient(ts.GetTenantURL(), authz.PassthroughAuthorization(), authz.JSONClient(security.PassXForwardedFor()))
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := mgr.DeleteLoginApp(ctx, ts.ID, authzClient, id); err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Login Application
// OpenAPI Tags: Login Application
// OpenAPI Description: This endpoint retrieves a login application for the tenant. Only tenant admins can get login applications.
func (h *handler) getLoginApp(ctx context.Context, id uuid.UUID, _ url.Values) (*plex.LoginAppResponse, int, []auditlog.Entry, error) {
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	app, err := mgr.GetLoginApp(ctx, ts.ID, id)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	resp, err := loginAppResponseFromApp(ctx, app)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &resp, http.StatusOK, nil, nil
}

type listLoginAppsParams struct {
	OrganizationID *string `description:"Optional organization ID filter" query:"organization_id"`
}

// OpenAPI Summary: List Login Applications
// OpenAPI Tags: Login Application
// OpenAPI Description: This endpoint lists all login applications for the tenant. Only tenant admins can list login applications.
func (h *handler) listLoginApps(ctx context.Context, req listLoginAppsParams) ([]plex.LoginAppResponse, int, []auditlog.Entry, error) {
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	orgID := uuid.Nil
	if req.OrganizationID != nil && *req.OrganizationID != "" {
		var err error
		orgID, err = uuid.FromString(*req.OrganizationID)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
	}

	apps, err := mgr.GetLoginApps(ctx, ts.ID, orgID)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	resps := make([]plex.LoginAppResponse, len(apps))
	for i, app := range apps {
		r, err := loginAppResponseFromApp(ctx, &app)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		resps[i] = r
	}

	return resps, http.StatusOK, nil, nil
}

func loginAppResponseFromApp(ctx context.Context, app *tenantplex.App) (plex.LoginAppResponse, error) {
	grantTypes := []string{}
	responseTypes := []string{}
	for _, gt := range app.GrantTypes {
		grantTypes = append(grantTypes, string(gt))
		if gt == tenantplex.GrantTypeAuthorizationCode {
			responseTypes = append(responseTypes, "code")
		}
		if gt == tenantplex.GrantTypeImplicit {
			responseTypes = append(responseTypes, "token")
		}
	}

	cs, err := app.ClientSecret.Resolve(ctx)
	if err != nil {
		return plex.LoginAppResponse{}, ucerr.Wrap(err)
	}

	return plex.LoginAppResponse{
		AppID:                 app.ID,
		ClientID:              app.ClientID,
		ClientSecret:          cs,
		ClientSecretExpiresAt: time.Time{},
		OrganizationID:        app.OrganizationID,
		Metadata: plex.LoginAppRequest{
			RedirectURIs:            app.AllowedRedirectURIs,
			TokenEndpointAuthMethod: "client_secret_basic",
			GrantTypes:              grantTypes,
			ResponseTypes:           responseTypes,
			ClientName:              app.Name,
		},
	}, nil
}
