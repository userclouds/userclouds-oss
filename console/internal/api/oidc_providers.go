package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/plex/manager"
)

// CreateOIDCProviderRequest is a request for creating an OIDC provider
type CreateOIDCProviderRequest struct {
	OIDCProvider oidc.ProviderConfig `json:"oidc_provider"`
}

//go:generate genvalidate CreateOIDCProviderRequest

func (h *handler) createOIDCProviderHandler(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req CreateOIDCProviderRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// load company for ACL checks
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if err := h.ensureCompanyAdmin(r, tenant.CompanyID); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	tp.PlexConfig.OIDCProviders.Providers =
		append(tp.PlexConfig.OIDCProviders.Providers, req.OIDCProvider)
	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, req.OIDCProvider)
}

// DeleteOIDCProviderRequest is a request for deleting an OIDC provider
type DeleteOIDCProviderRequest struct {
	OIDCProviderName string `json:"oidc_provider_name" validate:"notempty"`
}

//go:generate genvalidate DeleteOIDCProviderRequest

func (h *handler) deleteOIDCProviderHandler(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req DeleteOIDCProviderRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// load company for ACL checks
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if err := h.ensureCompanyAdmin(r, tenant.CompanyID); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	foundProvider := false
	providers := []oidc.ProviderConfig{}
	for _, provider := range tp.PlexConfig.OIDCProviders.Providers {
		if provider.Name == req.OIDCProviderName {
			foundProvider = true
			continue
		}
		providers = append(providers, provider)
	}

	if !foundProvider {
		jsonapi.MarshalError(ctx, w,
			ucerr.Errorf("no OIDC provider with name '%s'", req.OIDCProviderName),
			jsonapi.Code(http.StatusBadRequest))
		return
	}

	tp.PlexConfig.OIDCProviders.Providers = providers
	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
