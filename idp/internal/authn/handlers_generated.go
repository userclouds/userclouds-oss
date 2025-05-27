// NOTE: automatically generated file -- DO NOT EDIT

package authn

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/users").
		GetOne(h.getUserGenerated).
		Post(h.createUserGenerated).
		Put(h.updateUserGenerated).
		Delete(h.deleteUserGenerated).
		GetAll(h.listUsersGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/baseprofiles").
		GetAll(h.listUserBaseProfilesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

}

func (h *handler) createUserGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateUserAndAuthnRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.UserResponse
	res, code, entries, err := h.createUser(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteUserGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteUser(ctx, id, urlValues)
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

func (h *handler) getUserGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.UserResponse
	res, code, entries, err := h.getUser(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listUsersGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := ListUsersParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("organization_id") && urlValues.Get("organization_id") != "null" {
		v := urlValues.Get("organization_id")
		req.OrganizationID = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *idp.ListUsersResponse
	res, code, entries, err := h.listUsers(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateUserGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.UserResponse
	res, code, entries, err := h.updateUser(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listUserBaseProfilesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := ListUserBaseProfilesParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("organization_id") && urlValues.Get("organization_id") != "null" {
		v := urlValues.Get("organization_id")
		req.OrganizationID = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}
	if urlValues.Has("user_ids") {
		req.UserIDs = urlValues["user_ids"]
	}

	var res *idp.ListUserBaseProfilesResponse
	res, code, entries, err := h.listUserBaseProfiles(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
