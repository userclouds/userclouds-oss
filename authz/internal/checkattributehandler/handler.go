package checkattributehandler

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/tenantmap"
)

type handler struct {
	tenantsHandled map[uuid.UUID]bool
	tenantStateMap *tenantmap.StateMap
}

// NewHandler returns a new AuthZ API handler
func NewHandler(tenantStateMap *tenantmap.StateMap, tenants []uuid.UUID) http.Handler {
	h := &handler{tenantStateMap: tenantStateMap}
	h.tenantsHandled = make(map[uuid.UUID]bool)
	for _, tenantID := range tenants {
		h.tenantsHandled[tenantID] = true
	}

	hb := builder.NewHandlerBuilder()
	hb.CollectionHandler("/").
		GetOne(h.checkAttribute).
		WithAuthorizer(uchttp.NewAllowAllAuthorizer())

	return hb.Build()
}

func (h *handler) checkAttribute(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	if len(h.tenantsHandled) > 0 && !h.tenantsHandled[tenantID] {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("tenant %s not handled by this checkattribute instance", tenantID), jsonapi.Code(http.StatusNotFound))
		return
	}

	sourceObjectID, err := uuid.FromString(r.URL.Query().Get("source_object_id"))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}
	targetObjectID, err := uuid.FromString(r.URL.Query().Get("target_object_id"))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}
	attributeName := r.URL.Query().Get("attribute")
	if len(attributeName) == 0 {
		jsonapi.MarshalError(ctx, w, ucerr.New("missing 'attribute' query parameter"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	ts, err := h.tenantStateMap.GetTenantStateForID(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}
	s := internal.NewStorage(ctx, ts.ID, ts.TenantDB, ts.CacheConfig)

	hasAttribute, path, err := internal.CheckAttributeBFS(ctx, s, tenantID, sourceObjectID, targetObjectID, attributeName, false)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
		return
	}

	jsonapi.Marshal(w, authz.CheckAttributeResponse{
		HasAttribute: hasAttribute,
		Path:         path,
	}, jsonapi.Code(http.StatusOK))

}
