package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

type handler struct {
}

// New returns a AuditLog handler
func New() http.Handler {
	h := &handler{}

	// Auth on these APIs is handled by the JWT middleware on all AuthZ routes.

	hb := builder.NewHandlerBuilder()
	hb.CollectionHandler("/entries").
		GetAll(h.listAuditLogEntries).
		GetOne(h.getAuditLogEntry).
		Post(h.createAuditLogEntry).
		WithAuthorizer(uchttp.NewAllowAllAuthorizer())

	return hb.Build()
}

func mustGetStorage(ctx context.Context) *auditlog.Storage {
	tenant := multitenant.MustGetTenantState(ctx)
	return auditlog.NewStorage(tenant.TenantDB)
}

func handleUpsertError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	if ucdb.IsUniqueViolation(err) {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
		return
	} else if errors.Is(err, sql.ErrNoRows) {
		// This error occurs if certain fields are changed in a Save* operation
		// that should not be changed.
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	jsonapi.MarshalError(ctx, w, err)
}

func (h *handler) listAuditLogEntries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := mustGetStorage(ctx)

	pager, err := auditlog.NewEntryPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	es, respFields, err := s.ListEntriesPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, auditlog.ListEntriesResponse{Data: es, ResponseFields: *respFields}, jsonapi.Code(http.StatusOK))

}

func (h *handler) getAuditLogEntry(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := mustGetStorage(ctx)

	typ, err := s.GetEntry(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, typ)
}

func (h *handler) createAuditLogEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := mustGetStorage(ctx)

	var auditLogEntry auditlog.Entry
	if err := jsonapi.Unmarshal(r, &auditLogEntry); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := s.SaveEntry(ctx, &auditLogEntry); err != nil {
		handleUpsertError(ctx, w, r, err)
		return
	}

	jsonapi.Marshal(w, auditLogEntry, jsonapi.Code(http.StatusCreated))
}
