// NOTE: automatically generated file -- DO NOT EDIT

package datamapping

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/datamapping"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/datasource").
		GetOne(h.getDataSourceGenerated).
		Post(h.createDataSourceGenerated).
		Put(h.updateDataSourceGenerated).
		Delete(h.deleteDataSourceGenerated).
		GetAll(h.listDataSourcesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/element").
		GetOne(h.getDataSourceElementGenerated).
		Post(h.createDataSourceElementGenerated).
		Put(h.updateDataSourceElementGenerated).
		Delete(h.deleteDataSourceElementGenerated).
		GetAll(h.listDataSourceElementsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

}

func (h *handler) createDataSourceGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req datamapping.CreateDataSourceRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *datamapping.DataSource
	res, code, entries, err := h.createDataSource(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteDataSourceGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteDataSource(ctx, id, urlValues)
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

func (h *handler) getDataSourceGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *datamapping.DataSource
	res, code, entries, err := h.getDataSource(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listDataSourcesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listDataSourcesParams{}
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

	var res *datamapping.ListDataSourcesResponse
	res, code, entries, err := h.listDataSources(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateDataSourceGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req datamapping.UpdateDataSourceRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *datamapping.DataSource
	res, code, entries, err := h.updateDataSource(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createDataSourceElementGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req datamapping.CreateDataSourceElementRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *datamapping.DataSourceElement
	res, code, entries, err := h.createDataSourceElement(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteDataSourceElementGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteDataSourceElement(ctx, id, urlValues)
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

func (h *handler) getDataSourceElementGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *datamapping.DataSourceElement
	res, code, entries, err := h.getDataSourceElement(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listDataSourceElementsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listDataSourceElementsParams{}
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

	var res *datamapping.ListDataSourceElementsResponse
	res, code, entries, err := h.listDataSourceElements(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateDataSourceElementGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req datamapping.UpdateDataSourceElementRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *datamapping.DataSourceElement
	res, code, entries, err := h.updateDataSourceElement(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
