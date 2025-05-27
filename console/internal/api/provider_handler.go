package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
	"userclouds.com/worker"
)

func (h *handler) importProviderAppHandler(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	ten, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	for _, p := range tp.PlexConfig.PlexMap.Providers {
		if p.Type == tenantplex.ProviderTypeAuth0 {
			msg := worker.CreateImportAuth0AppsMessage(tenantID, ten.TenantURL, p.ID)
			if err := h.workerClient.Send(ctx, msg); err != nil {
				uchttp.Error(ctx, w, err, http.StatusInternalServerError)
				return
			}
			break
		}
	}
}
