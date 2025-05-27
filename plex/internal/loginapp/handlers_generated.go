// NOTE: automatically generated file -- DO NOT EDIT

package loginapp

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
	"userclouds.com/plex"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/register").
		GetOne(h.getLoginAppGenerated).
		Post(h.createLoginAppGenerated).
		Put(h.updateLoginAppGenerated).
		Delete(h.deleteLoginAppGenerated).
		GetAll(h.listLoginAppsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

}

func (h *handler) createLoginAppGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req plex.LoginAppRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *plex.LoginAppResponse
	res, code, entries, err := h.createLoginApp(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteLoginAppGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteLoginApp(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getLoginAppGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *plex.LoginAppResponse
	res, code, entries, err := h.getLoginApp(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listLoginAppsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listLoginAppsParams{}
	if urlValues.Has("organization_id") && urlValues.Get("organization_id") != "null" {
		v := urlValues.Get("organization_id")
		req.OrganizationID = &v
	}

	var res []plex.LoginAppResponse
	res, code, entries, err := h.listLoginApps(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateLoginAppGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req plex.LoginAppRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *plex.LoginAppResponse
	res, code, entries, err := h.updateLoginApp(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
