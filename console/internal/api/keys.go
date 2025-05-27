package api

import (
	"fmt"
	"net/http"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/provisioning"
	"userclouds.com/plex/manager"
)

type listTenantPublicKeysResponse struct {
	PublicKeys []string `json:"public_keys"`
}

// naming this list instead of get since someday (soon?) we'll support a list of keypairs
func (h *handler) listTenantPublicKeys(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

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

	resp := listTenantPublicKeysResponse{
		PublicKeys: []string{tp.PlexConfig.Keys.PublicKey},
	}
	jsonapi.Marshal(w, resp)
}

func (h *handler) getTenantPrivateKey(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// extra safety check here to protect the private key
	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "User must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
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

	pk, err := tp.PlexConfig.Keys.PrivateKey.Resolve(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pem", tenantID))
	w.Header().Set(headers.ContentType, "application/octet-stream")
	fmt.Fprint(w, pk)
}

func (h *handler) rotateKeys(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

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

	keys, err := provisioning.GeneratePlexKeys(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tp.PlexConfig.Keys = *keys

	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// plex's tenantconfig cache will timeout and update soon

	w.WriteHeader(http.StatusNoContent)
}
