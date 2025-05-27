package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/worker"
)

type createTenantURLRequest struct {
	TenantURL companyconfig.TenantURL `json:"tenant_url"`
}

type createTenantURLResponse struct {
	TenantURL companyconfig.TenantURL `json:"tenant_url"`
}

func (h *handler) createTenantURL(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req createTenantURLRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// TODO unify validation logic with update?
	if req.TenantURL.Validated || req.TenantURL.System {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot create validated/system tenant URLs"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// validate tenantIDs while we're here
	if req.TenantURL.TenantID != tenantID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "wrong tenant ID in tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// if this doesn't include a scheme, add one
	if !strings.Contains(req.TenantURL.TenantURL, "://") {
		req.TenantURL.TenantURL = fmt.Sprintf("%s://%s", h.tenantsProtocol, req.TenantURL.TenantURL)
	}

	u, err := url.Parse(req.TenantURL.TenantURL)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "failed to parse tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}
	// disallow explicit ports not in dev
	if u.Port() != "" && !universe.Current().IsDev() {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot specify port for tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}
	// you can't use eg. HTTP in non-dev
	if u.Scheme != h.tenantsProtocol {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "tenant URL must use scheme %v", h.tenantsProtocol), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// disallow new userclouds.com URLs
	if strings.HasSuffix(req.TenantURL.TenantURL, "userclouds.com") {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot add new userclouds.com URLs"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// there's a package called https://github.com/jpillora/go-tld that is supposed to do this
	// but it's designed for URIs with a scheme, not just host-optional-port, so I'm doing it manually
	// the objective is to grab just the last part of the domain to match, since
	// Suffix(..., "tenant.userclouds.com") will miss tenant-aws-us-west-2.userclouds.com,
	// and HasSuffix( ..., "userclouds.com") won't work in dev
	// this is pretty custom for this use case (eg. it wouldn't go high enough to catch
	// bbc.co.uk, and just check .co.uk, but it seems sufficient for our use case right now)
	fqdn := h.tenantsSubDomain
	if strings.Contains(fqdn, ":") {
		fqdn, _, err = net.SplitHostPort(fqdn)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}
	parts := strings.Split(fqdn, ".")
	lastTwo := parts[len(parts)-2:]
	suffix := strings.Join(lastTwo, ".")

	if err := h.storage.SaveTenantURL(ctx, &req.TenantURL); err != nil {
		if ucdb.IsUniqueViolation(err) {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "duplicate tenant URL value"), jsonapi.Code(http.StatusConflict))
			return
		}
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// TODO: better place to kick off this logic?
	if !strings.HasSuffix(u.Hostname(), suffix) {
		if err := h.workerClient.Send(ctx, worker.CreateNewTenantCNAMEMessage(tenantID, req.TenantURL.TenantURL)); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, createTenantURLResponse(req), jsonapi.Code(http.StatusCreated))
}

type updateTenantURLRequest struct {
	TenantURL companyconfig.TenantURL `json:"tenant_url"`
}

type updateTenantURLResponse struct {
	TenantURL companyconfig.TenantURL `json:"tenant_url"`
}

// NB: this only affects the "secondary" tenant URL table, not the main tenant record / primary URL
func (h *handler) updateTenantURL(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, tenantURLID uuid.UUID) {
	ctx := r.Context()

	var req updateTenantURLRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	e, err := h.storage.GetTenantURL(ctx, tenantURLID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// validate incoming tenantURLs

	// these can't be edited so they should never be passed up
	if req.TenantURL.System || e.System {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot set system tenant URLs"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// validate tenantIDs while we're here
	if req.TenantURL.TenantID != tenantID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "wrong tenant ID in tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// if this doesn't include a scheme, add one
	if !strings.Contains(req.TenantURL.TenantURL, "://") {
		req.TenantURL.TenantURL = fmt.Sprintf("%s://%s", h.tenantsProtocol, req.TenantURL.TenantURL)
	}

	u, err := url.Parse(req.TenantURL.TenantURL)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "failed to parse tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}
	// disallow explicit ports not in dev
	if u.Port() != "" && !universe.Current().IsDev() {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot specify port for tenant URL"), jsonapi.Code(http.StatusBadRequest))
		return
	}
	// you can't use eg. HTTP in non-dev
	if u.Scheme != h.tenantsProtocol {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "tenant URL must use scheme %v", h.tenantsProtocol), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// disallow new userclouds.com URLs
	if strings.HasSuffix(req.TenantURL.TenantURL, "userclouds.com") {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot add new userclouds.com URLs"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	// mark all request TenantURLs as not-validated unless proven otherwise
	if req.TenantURL.TenantURL == e.TenantURL {
		// only mark as validated if it's already been validated,
		// so eg. two saves don't mark it as validated
		req.TenantURL.Validated = e.Validated
	} else {
		// if the URL is changing, we need to re-validate it
		req.TenantURL.Validated = false
	}

	// there's a package called https://github.com/jpillora/go-tld that is supposed to do this
	// but it's designed for URIs with a scheme, not just host-optional-port, so I'm doing it manually
	// the objective is to grab just the last part of the domain to match, since
	// Suffix(..., "tenant.userclouds.com") will miss tenant-aws-us-west-2.userclouds.com,
	// and HasSuffix( ..., "userclouds.com") won't work in dev
	// this is pretty custom for this use case (eg. it wouldn't go high enough to catch
	// bbc.co.uk, and just check .co.uk, but it seems sufficient for our use case right now)
	fqdn := h.tenantsSubDomain
	if strings.Contains(fqdn, ":") {
		fqdn, _, err = net.SplitHostPort(fqdn)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}
	parts := strings.Split(fqdn, ".")
	lastTwo := parts[len(parts)-2:]
	suffix := strings.Join(lastTwo, ".")

	if err := h.storage.SaveTenantURL(ctx, &req.TenantURL); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// TODO: better place to kick off this logic?
	if !strings.HasSuffix(u.Hostname(), suffix) {
		if err := h.workerClient.Send(ctx, worker.CreateNewTenantCNAMEMessage(tenantID, req.TenantURL.TenantURL)); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, updateTenantURLResponse(req))
}

func (h *handler) deleteTenantURL(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, tenantURLID uuid.UUID) {
	ctx := r.Context()

	e, err := h.storage.GetTenantURL(ctx, tenantURLID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if e.System {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "cannot delete system tenant URLs"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	if err := h.storage.DeleteTenantURL(ctx, e.ID); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TODO: we currently pass this in the request since we don't support doubly-nested method handlers
type validateTenantURLRequest struct {
	TenantURLID uuid.UUID `json:"tenant_url_id"`
}

func (h *handler) validateTenantURL(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req validateTenantURLRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tu, err := h.storage.GetTenantURL(ctx, req.TenantURLID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	msg := worker.CreateValidateDNSMessage(tenantID, tu.TenantURL)
	if err := h.workerClient.Send(ctx, msg); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
