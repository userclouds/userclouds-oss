package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/datamapping"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
)

func (h *handler) listDataSources(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := []datamapping.Option{datamapping.Pagination(pager.GetOptions()...)}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := dmClient.ListDataSources(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getDataSource(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.GetDataSource(ctx, dsID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}

func (h *handler) updateDataSource(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	var req datamapping.UpdateDataSourceRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.UpdateDataSource(ctx, req.DataSource)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}

func (h *handler) deleteDataSource(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := dmClient.DeleteDataSource(ctx, dsID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) createDataSource(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req datamapping.CreateDataSourceRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.CreateDataSource(ctx, req.DataSource)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}

func (h *handler) listDataSourceElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := []datamapping.Option{datamapping.Pagination(pager.GetOptions()...)}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := dmClient.ListDataSourceElements(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getDataSourceElement(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.GetDataSourceElement(ctx, dsID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}

func (h *handler) updateDataSourceElement(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	var req datamapping.UpdateDataSourceElementRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.UpdateDataSourceElement(ctx, req.DataSourceElement)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}

func (h *handler) deleteDataSourceElement(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, dsID uuid.UUID) {
	ctx := r.Context()

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := dmClient.DeleteDataSourceElement(ctx, dsID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) createDataSourceElement(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req datamapping.CreateDataSourceElementRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	dmClient, err := h.newDatamappingClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ds, err := dmClient.CreateDataSourceElement(ctx, req.DataSourceElement)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, ds)
}
