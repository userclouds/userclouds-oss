package api

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/worker/storage"
)

type listRunsResponse struct {
	pagination.ResponseFields

	Runs []storage.IDPSyncRun `json:"data"`
}

func (h *handler) listRuns(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	ten, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.ensureCompanyEmployee(r, ten.CompanyID); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	// TODO: console doesn't normally read tenant DBs but uses cross-service calls, but in this
	// case worker isn't a "service" that is accessible synchronously (yet?)
	tdb, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	s := storage.New(tdb)

	var opts []pagination.Option

	// I think type always gets used in our current designs (but maybe not), but make it optional
	// since in theory we might show everything?
	if typ := storage.SyncRunType(r.URL.Query().Get("type")); typ != "" {
		if err := typ.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
		opts = append(opts, pagination.Filter(fmt.Sprintf("('type',EQ,'%v')", typ)))
	}

	pager, err := storage.NewIDPSyncRunPaginatorFromRequest(r, opts...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	runs, pr, err := s.ListIDPSyncRunsPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	res := listRunsResponse{
		ResponseFields: *pr,
		Runs:           runs,
	}

	jsonapi.Marshal(w, res)
}

type listRecordsResponse struct {
	pagination.ResponseFields

	Records []storage.IDPSyncRecord `json:"data"`
}

func (h *handler) getRun(w http.ResponseWriter, r *http.Request, tenantID, runID uuid.UUID) {
	ctx := r.Context()

	ten, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.ensureCompanyEmployee(r, ten.CompanyID); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	// TODO: console doesn't normally read tenant DBs but uses cross-service calls, but in this
	// case worker isn't a "service" that is accessible synchronously (yet?)
	tdb, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	s := storage.New(tdb)

	pager, err := storage.NewIDPSyncRecordPaginatorFromRequest(
		r,
		pagination.Filter(fmt.Sprintf("('sync_run_id',EQ,'%v')", runID)),
	)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	recs, pr, err := s.ListIDPSyncRecordsPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	res := listRecordsResponse{
		ResponseFields: *pr,
		Records:        recs,
	}

	jsonapi.Marshal(w, res)
}
