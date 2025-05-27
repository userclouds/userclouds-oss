package acme

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/acmestorage"
	"userclouds.com/internal/multitenant"
	"userclouds.com/worker"
)

type handler struct {
	acmeThumbprint string
	qc             workerclient.Client
}

// NewHandler returns a new ACME handler
func NewHandler(cfg *acme.Config, qc workerclient.Client) http.Handler {
	mux := uchttp.NewServeMux()
	h := &handler{qc: qc}
	ctx := context.Background()

	pk, err := cfg.PrivateKey.Resolve(ctx)
	if err != nil {
		// TODO: in theory we could keep running in a degraded state with this failure,
		// so over time I think we should log/alert, but for now, if we have bad config
		// I'd rather just fatal and notice it immediately?
		uclog.Fatalf(ctx, "failed to resolve private key: %v", err)
	}

	tb, err := acme.ComputeThumbprint(ctx, pk)
	if err != nil {
		// TODO: same fatal error as above...could continue but need to make sure we notice
		uclog.Errorf(ctx, "failed to compute thumbprint: %v", err)
	}
	h.acmeThumbprint = tb

	// regardless of what token LE asks for, we handle it the same place
	mux.HandleFunc("/", h.handleWellKnown)

	return mux
}

func (h *handler) handleWellKnown(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.qc == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "not currently supported"), jsonapi.Code(http.StatusNotImplemented))
		return
	}

	// this might be overly paranoid but in case we change pathing or something
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 2 || parts[0] != "" {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "invalid path"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	token := parts[1]

	ts := multitenant.MustGetTenantState(ctx)
	as := acmestorage.New(ts.TenantDB)

	// ensure we expected this challenge
	order, err := as.GetOrderByToken(ctx, token)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	if order.Status != acmestorage.OrderStatusPending && order.Status != acmestorage.OrderStatusValidated {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "order status not pending"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	w.Header().Add(headers.ContentType, "text/plain")
	w.Write(fmt.Appendf(nil, "%s.%s", token, h.acmeThumbprint))

	// LE at least (maybe all ACME CAs) will send the same challenge multiple times
	// don't reprocess it if we've already done so (but obviously return correct result above)
	if order.Status == acmestorage.OrderStatusValidated {
		return
	}

	// start the finalize process
	msg := worker.CreateFinalizeTenantCNAMEMessage(ts.ID, order.ID)
	if err := h.qc.Send(ctx, msg); err != nil {
		// don't blow up the ACME verification by returning this error over HTTP, just log it
		uclog.Errorf(ctx, "failed to send %+v: %v", msg, err)
	}

	order.Status = acmestorage.OrderStatusValidated
	if err := as.SaveOrder(ctx, order); err != nil {
		// don't blow up the ACME verification by returning this error over HTTP, just log it
		uclog.Errorf(ctx, "failed to update order: %v", err)
	}
}
