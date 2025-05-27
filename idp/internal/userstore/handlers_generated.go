// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/config/accessors").
		GetOne(h.getAccessorGenerated).
		Post(h.createAccessorGenerated).
		Put(h.updateAccessorGenerated).
		Delete(h.deleteAccessorGenerated).
		GetAll(h.listAccessorsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	handlerColumn := builder.CollectionHandler("/config/columns").
		GetOne(h.getColumnGenerated).
		Post(h.createColumnGenerated).
		Put(h.updateColumnGenerated).
		Delete(h.deleteColumnGenerated).
		GetAll(h.listColumnsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/config/datatypes").
		GetOne(h.getDataTypeGenerated).
		Post(h.createDataTypeGenerated).
		Put(h.updateDataTypeGenerated).
		Delete(h.deleteDataTypeGenerated).
		GetAll(h.listDataTypesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/config/liveretentiondurations").
		GetOne(h.getLiveRetentionDurationGenerated).
		Post(h.createLiveRetentionDurationGenerated).
		Put(h.updateLiveRetentionDurationGenerated).
		Delete(h.deleteLiveRetentionDurationGenerated).
		GetAll(h.listLiveRetentionDurationsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/config/mutators").
		GetOne(h.getMutatorGenerated).
		Post(h.createMutatorGenerated).
		Put(h.updateMutatorGenerated).
		Delete(h.deleteMutatorGenerated).
		GetAll(h.listMutatorsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	handlerPurpose := builder.CollectionHandler("/config/purposes").
		GetOne(h.getPurposeGenerated).
		Post(h.createPurposeGenerated).
		Put(h.updatePurposeGenerated).
		Delete(h.deletePurposeGenerated).
		GetAll(h.listPurposesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/config/softdeletedretentiondurations").
		GetOne(h.getSoftDeletedRetentionDurationGenerated).
		Post(h.createSoftDeletedRetentionDurationGenerated).
		Put(h.updateSoftDeletedRetentionDurationGenerated).
		Delete(h.deleteSoftDeletedRetentionDurationGenerated).
		GetAll(h.listSoftDeletedRetentionDurationsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/config/searchindices").
		GetOne(h.getUserSearchIndexGenerated).
		Post(h.createUserSearchIndexGenerated).
		Put(h.updateUserSearchIndexGenerated).
		Delete(h.deleteUserSearchIndexGenerated).
		GetAll(h.listUserSearchIndexesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/api/users").
		Post(h.createUserstoreUserGenerated).
		Delete(h.deleteUserstoreUserGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	handlerColumn.NestedCollectionHandler("/liveretentiondurations").
		GetOne(h.getLiveRetentionDurationOnColumnGenerated).
		Post(h.createLiveRetentionDurationOnColumnGenerated).
		Put(h.updateLiveRetentionDurationOnColumnGenerated).
		Delete(h.deleteLiveRetentionDurationOnColumnGenerated).
		GetAll(h.listLiveRetentionDurationsOnColumnGenerated).
		WithAuthorizer(h.newNestedRoleBasedAuthorizer())

	handlerPurpose.NestedCollectionHandler("/liveretentiondurations").
		GetOne(h.getLiveRetentionDurationOnPurposeGenerated).
		Post(h.createLiveRetentionDurationOnPurposeGenerated).
		Put(h.updateLiveRetentionDurationOnPurposeGenerated).
		Delete(h.deleteLiveRetentionDurationOnPurposeGenerated).
		GetAll(h.listLiveRetentionDurationsOnPurposeGenerated).
		WithAuthorizer(h.newNestedRoleBasedAuthorizer())

	handlerColumn.NestedCollectionHandler("/softdeletedretentiondurations").
		GetOne(h.getSoftDeletedRetentionDurationOnColumnGenerated).
		Post(h.createSoftDeletedRetentionDurationOnColumnGenerated).
		Put(h.updateSoftDeletedRetentionDurationOnColumnGenerated).
		Delete(h.deleteSoftDeletedRetentionDurationOnColumnGenerated).
		GetAll(h.listSoftDeletedRetentionDurationsOnColumnGenerated).
		WithAuthorizer(h.newNestedRoleBasedAuthorizer())

	handlerPurpose.NestedCollectionHandler("/softdeletedretentiondurations").
		GetOne(h.getSoftDeletedRetentionDurationOnPurposeGenerated).
		Post(h.createSoftDeletedRetentionDurationOnPurposeGenerated).
		Put(h.updateSoftDeletedRetentionDurationOnPurposeGenerated).
		Delete(h.deleteSoftDeletedRetentionDurationOnPurposeGenerated).
		GetAll(h.listSoftDeletedRetentionDurationsOnPurposeGenerated).
		WithAuthorizer(h.newNestedRoleBasedAuthorizer())

	builder.MethodHandler("/api/accessors").Post(h.executeAccessorHandlerGenerated)

	builder.MethodHandler("/api/consentedpurposes").Post(h.getConsentedPurposesForUserGenerated)

	builder.MethodHandler("/api/mutators").Post(h.executeMutatorHandlerGenerated)

}

func (h *handler) executeAccessorHandlerGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var req executeAccessorHandlerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
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

	var res *idp.ExecuteAccessorResponse
	res, code, entries, err := h.executeAccessorHandler(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) executeMutatorHandlerGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.ExecuteMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ExecuteMutatorResponse
	res, code, entries, err := h.executeMutatorHandler(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) getConsentedPurposesForUserGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.GetConsentedPurposesForUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.GetConsentedPurposesForUserResponse
	res, code, entries, err := h.getConsentedPurposesForUser(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createAccessorGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateAccessorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Accessor
	res, code, entries, err := h.createAccessor(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteAccessorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteAccessor(ctx, id, urlValues)
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

func (h *handler) getAccessorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := GetAccessorParams{}
	if urlValues.Has("accessor_version") && urlValues.Get("accessor_version") != "null" {
		v := urlValues.Get("accessor_version")
		req.Version = &v
	}

	var res *userstore.Accessor
	res, code, entries, err := h.getAccessor(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listAccessorsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listAccessorsParams{}
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
	if urlValues.Has("versioned") && urlValues.Get("versioned") != "null" {
		v := urlValues.Get("versioned")
		req.Versioned = &v
	}

	var res *idp.ListAccessorsResponse
	res, code, entries, err := h.listAccessors(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateAccessorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateAccessorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Accessor
	res, code, entries, err := h.updateAccessor(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createColumnGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateColumnRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Column
	res, code, entries, err := h.createColumn(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteColumnGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteColumn(ctx, id, urlValues)
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

func (h *handler) getColumnGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *userstore.Column
	res, code, entries, err := h.getColumn(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listColumnsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listColumnsParams{}
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

	var res *idp.ListColumnsResponse
	res, code, entries, err := h.listColumns(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateColumnGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Column
	res, code, entries, err := h.updateColumn(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createDataTypeGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateDataTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.ColumnDataType
	res, code, entries, err := h.createDataType(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteDataTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteDataType(ctx, id, urlValues)
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

func (h *handler) getDataTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *userstore.ColumnDataType
	res, code, entries, err := h.getDataType(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listDataTypesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listDataTypesParams{}
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

	var res *idp.ListDataTypesResponse
	res, code, entries, err := h.listDataTypes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateDataTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateDataTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.ColumnDataType
	res, code, entries, err := h.updateDataType(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createLiveRetentionDurationGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.createLiveRetentionDuration(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteLiveRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteLiveRetentionDuration(ctx, id, urlValues)
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

func (h *handler) getLiveRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getLiveRetentionDuration(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listLiveRetentionDurationsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.listLiveRetentionDurations(ctx, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateLiveRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateLiveRetentionDuration(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createLiveRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationsResponse
	res, code, entries, err := h.createLiveRetentionDurationOnColumn(ctx, parentID, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteLiveRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteLiveRetentionDurationOnColumn(ctx, parentID, id, urlValues)
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

func (h *handler) getLiveRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getLiveRetentionDurationOnColumn(ctx, parentID, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listLiveRetentionDurationsOnColumnGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationsResponse
	res, code, entries, err := h.listLiveRetentionDurationsOnColumn(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateLiveRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateLiveRetentionDurationOnColumn(ctx, parentID, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createLiveRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.createLiveRetentionDurationOnPurpose(ctx, parentID, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteLiveRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteLiveRetentionDurationOnPurpose(ctx, parentID, id, urlValues)
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

func (h *handler) getLiveRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getLiveRetentionDurationOnPurpose(ctx, parentID, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listLiveRetentionDurationsOnPurposeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.listLiveRetentionDurationsOnPurpose(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateLiveRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateLiveRetentionDurationOnPurpose(ctx, parentID, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createMutatorGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Mutator
	res, code, entries, err := h.createMutator(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteMutatorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteMutator(ctx, id, urlValues)
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

func (h *handler) getMutatorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := GetMutatorParams{}
	if urlValues.Has("mutator_version") && urlValues.Get("mutator_version") != "null" {
		v := urlValues.Get("mutator_version")
		req.Version = &v
	}

	var res *userstore.Mutator
	res, code, entries, err := h.getMutator(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listMutatorsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listMutatorsParams{}
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
	if urlValues.Has("versioned") && urlValues.Get("versioned") != "null" {
		v := urlValues.Get("versioned")
		req.Versioned = &v
	}

	var res *idp.ListMutatorsResponse
	res, code, entries, err := h.listMutators(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateMutatorGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Mutator
	res, code, entries, err := h.updateMutator(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createPurposeGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreatePurposeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Purpose
	res, code, entries, err := h.createPurpose(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deletePurposeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deletePurpose(ctx, id, urlValues)
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

func (h *handler) getPurposeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *userstore.Purpose
	res, code, entries, err := h.getPurpose(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listPurposesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listPurposesParams{}
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

	var res *idp.ListPurposesResponse
	res, code, entries, err := h.listPurposes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updatePurposeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdatePurposeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *userstore.Purpose
	res, code, entries, err := h.updatePurpose(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createSoftDeletedRetentionDurationGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.createSoftDeletedRetentionDuration(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteSoftDeletedRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteSoftDeletedRetentionDuration(ctx, id, urlValues)
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

func (h *handler) getSoftDeletedRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getSoftDeletedRetentionDuration(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listSoftDeletedRetentionDurationsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.listSoftDeletedRetentionDurations(ctx, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateSoftDeletedRetentionDurationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateSoftDeletedRetentionDuration(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createSoftDeletedRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationsResponse
	res, code, entries, err := h.createSoftDeletedRetentionDurationOnColumn(ctx, parentID, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteSoftDeletedRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteSoftDeletedRetentionDurationOnColumn(ctx, parentID, id, urlValues)
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

func (h *handler) getSoftDeletedRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getSoftDeletedRetentionDurationOnColumn(ctx, parentID, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listSoftDeletedRetentionDurationsOnColumnGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationsResponse
	res, code, entries, err := h.listSoftDeletedRetentionDurationsOnColumn(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateSoftDeletedRetentionDurationOnColumnGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateSoftDeletedRetentionDurationOnColumn(ctx, parentID, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createSoftDeletedRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.createSoftDeletedRetentionDurationOnPurpose(ctx, parentID, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteSoftDeletedRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteSoftDeletedRetentionDurationOnPurpose(ctx, parentID, id, urlValues)
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

func (h *handler) getSoftDeletedRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.getSoftDeletedRetentionDurationOnPurpose(ctx, parentID, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listSoftDeletedRetentionDurationsOnPurposeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.listSoftDeletedRetentionDurationsOnPurpose(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateSoftDeletedRetentionDurationOnPurposeGenerated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateColumnRetentionDurationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *idp.ColumnRetentionDurationResponse
	res, code, entries, err := h.updateSoftDeletedRetentionDurationOnPurpose(ctx, parentID, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createUserSearchIndexGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateUserSearchIndexRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *search.UserSearchIndex
	res, code, entries, err := h.createUserSearchIndex(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteUserSearchIndexGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteUserSearchIndex(ctx, id, urlValues)
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

func (h *handler) getUserSearchIndexGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *search.UserSearchIndex
	res, code, entries, err := h.getUserSearchIndex(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listUserSearchIndexesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listUserSearchIndicesParams{}
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

	var res *idp.ListUserSearchIndicesResponse
	res, code, entries, err := h.listUserSearchIndexes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateUserSearchIndexGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req idp.UpdateUserSearchIndexRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *search.UserSearchIndex
	res, code, entries, err := h.updateUserSearchIndex(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createUserstoreUserGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.CreateUserWithMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res uuid.UUID
	res, code, entries, err := h.createUserstoreUser(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteUserstoreUserGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteUserstoreUser(ctx, id, urlValues)
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
