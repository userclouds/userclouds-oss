package handler

import (
	"context"
	"fmt"
	"net/http"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/multitenant"
	"userclouds.com/userevent"
	"userclouds.com/userevent/storage"
)

type handler struct {
}

// New returns a User Event handler
func New() http.Handler {
	h := &handler{}

	// Auth on these APIs is handled by the JWT middleware on all IDP routes.
	// TODO: this shouldn't live on IDP longer term.
	// TODO: this seems easy to mess up, we may want some kind of assertion that the JWT is valid?
	hb := builder.NewHandlerBuilder()

	hb.CollectionHandler("/events").
		GetAll(h.listEvents).
		Post(h.reportEvents).
		WithAuthorizer(uchttp.NewAllowAllAuthorizer())

	return hb.Build()
}

func getStorage(ctx context.Context) (*storage.Storage, error) {
	tenant := multitenant.MustGetTenantState(ctx)

	logDB, err := tenant.GetLogDB(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return storage.New(logDB), nil
}

func (h *handler) listEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uestorage, err := getStorage(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var options []pagination.Option
	if userAlias := r.URL.Query().Get("user_alias"); userAlias != "" {
		options = append(options, pagination.Filter(fmt.Sprintf("('user_alias',EQ,'%s')", userAlias)))
	}

	pager, err := userevent.NewUserEventPaginatorFromRequest(r, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	events, respFields, err := uestorage.ListUserEventsPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, userevent.ListEventsResponse{
		Data:           events,
		ResponseFields: *respFields,
	})
}

func (h *handler) reportEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uestorage, err := getStorage(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req userevent.ReportEventsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for _, event := range req.Events {
		if err := event.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}

		if err := uestorage.SaveUserEvent(ctx, &event); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
