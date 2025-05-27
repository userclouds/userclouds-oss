package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

func (h *handler) getLoginApp(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, appID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := h.ensureEmployeeAccessToTenant(r, tenant); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	app, err := mgr.GetLoginApp(ctx, tenantID, appID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, app)
}

// AddLoginAppRequest is the request body for adding a login app.
type AddLoginAppRequest struct {
	AppID        uuid.UUID `json:"app_id" yaml:"app_id"`
	Name         string    `json:"name" yaml:"name"`
	ClientID     string    `json:"client_id" yaml:"client_id"`
	ClientSecret string    `json:"client_secret" yaml:"client_secret"`
}

func (h *handler) addLoginApp(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := h.ensureEmployeeAccessToTenant(r, tenant); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req AddLoginAppRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	cs, err := crypto.CreateClientSecret(ctx, req.AppID.String(), req.ClientSecret)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	app := tenantplex.App{
		ID:           req.AppID,
		Name:         req.Name,
		ClientID:     req.ClientID,
		ClientSecret: *cs,
		GrantTypes:   []tenantplex.GrantType{tenantplex.GrantTypeAuthorizationCode, tenantplex.GrantTypeRefreshToken, tenantplex.GrantTypeClientCredentials},
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	if err := mgr.AddLoginApp(ctx, tenantID, authZClient, app); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, tp.PlexConfig)
}

func (h *handler) deleteLoginApp(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, appID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := h.ensureEmployeeAccessToTenant(r, tenant); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	if err := mgr.DeleteLoginApp(ctx, tenantID, authZClient, appID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, tp.PlexConfig)
}

func (h *handler) listLoginApps(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var organizationID uuid.UUID
	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		var err error
		organizationID, err = uuid.FromString(orgID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	apps, err := mgr.GetLoginApps(ctx, tenantID, organizationID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, apps)
}

// UpdateLoginAppRequest is the request body for updating a login app.
type UpdateLoginAppRequest struct {
	App tenantplex.App `json:"app" yaml:"app"`
}

func (h *handler) updateLoginApp(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, appID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := h.ensureEmployeeAccessToTenant(r, tenant); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req UpdateLoginAppRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	if err := mgr.UpdateLoginApp(ctx, tenantID, req.App); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, req.App)
}
