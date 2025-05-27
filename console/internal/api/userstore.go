package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
)

func (h *handler) listTenantUsers(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := []idp.Option{idp.Pagination(pager.GetOptions()...)}

	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		organizationID, err := uuid.FromString(orgID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
		options = append(options, idp.OrganizationID(organizationID))
	}

	idpClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.ListUserBaseProfiles(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

type consoleUser struct {
	idp.UserResponse
	Authns      []idp.UserAuthn      `json:"authns"`
	MFAChannels []idp.UserMFAChannel `json:"mfa_channels"`
}

func (h *handler) getTenantUser(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ts, err := m2m.GetM2MTokenSource(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := idp.NewClient(tenant.TenantURL, idp.JSONClient(ts, security.PassXForwardedFor()))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	user, err := idpClient.GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgmtClient, err := idp.NewManagementClient(tenant.TenantURL, ts, security.PassXForwardedFor())
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	userBaseProfileAndAuthN, err := mgmtClient.GetUserBaseProfileAndAuthN(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	userWithAuthns := consoleUser{
		UserResponse: *user,
		Authns:       userBaseProfileAndAuthN.Authns,
		MFAChannels:  userBaseProfileAndAuthN.MFAChannels,
	}

	jsonapi.Marshal(w, userWithAuthns)
}

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	var req idp.UpdateUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.UpdateUser(ctx, userID, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteUser(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteUser(ctx, userID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) saveProxyForDatabase(ctx context.Context, tenantID uuid.UUID, databaseID uuid.UUID, host string, port int) (*companyconfig.SQLShimProxy, error) {
	pager, err := companyconfig.NewSQLShimProxyPaginatorFromOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		proxies, pr, err := h.storage.ListSQLShimProxiesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, proxy := range proxies {
			if proxy.DatabaseID == databaseID && proxy.TenantID == tenantID {
				proxy.Host = host
				proxy.Port = port

				if err := h.storage.SaveSQLShimProxy(ctx, &proxy); err != nil {
					return nil, ucerr.Wrap(err)
				}

				return &proxy, nil
			}
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	proxy := &companyconfig.SQLShimProxy{
		BaseModel:  ucdb.NewBase(),
		DatabaseID: databaseID,
		TenantID:   tenantID,
		Host:       host,
		Port:       port,
	}

	if err := h.storage.SaveSQLShimProxy(ctx, proxy); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return proxy, nil
}

func (h *handler) updateOnPremProxyPorts(ctx context.Context, tenantID uuid.UUID) error {

	// Gather available ports and databases with ports
	uclog.Infof(ctx, "Updating on-prem SQL shim proxy ports for tenant %s", tenantID)
	availablePorts := set.NewIntSet(h.onPremSQLShimPorts...)
	databasesWithPorts := set.NewUUIDSet()

	pager, err := companyconfig.NewSQLShimProxyPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		proxies, pr, err := h.storage.ListSQLShimProxiesPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, proxy := range proxies {
			availablePorts.Evict(proxy.Port)
			databasesWithPorts.Insert(proxy.DatabaseID)
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	// If there are no available ports, we're done
	if availablePorts.Size() == 0 {
		uclog.Infof(ctx, "No available ports for tenant %s", tenantID)
		return nil
	}

	// Go through the list of databases and assign a proxy port for each one that doesn't already have one
	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	databases, err := idpClient.ListDatabases(ctx, idp.Pagination(pagination.Limit(pagination.MaxLimit)))
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, db := range databases.Data {
		if databasesWithPorts.Contains(db.ID) {
			continue
		}

		proxy := &companyconfig.SQLShimProxy{
			BaseModel:  ucdb.NewBase(),
			DatabaseID: db.ID,
			TenantID:   tenantID,
			Host:       h.onPremSQLShimHost,
			Port:       availablePorts.Items()[0],
		}
		uclog.Infof(ctx, "Assigning port %d to database %s", proxy.Port, db.ID)
		if err := h.storage.SaveSQLShimProxy(ctx, proxy); err != nil {
			return ucerr.Wrap(err)
		}

		availablePorts.Evict(proxy.Port)
		if availablePorts.Size() == 0 {
			uclog.Infof(ctx, "No more available ports for tenant %s", tenantID)
			break
		}
	}

	return nil
}

func (h *handler) updateTenantUserStoreDatabaseProxyPorts(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if universe.Current().IsOnPrem() {
		ctx := r.Context()
		if err := h.updateOnPremProxyPorts(ctx, tenantID); err != nil {
			jsonapi.MarshalError(r.Context(), w, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) testTenantUserStoreDatabase(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "User must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req idp.TestDatabaseRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.TestDatabase(ctx, req.Database)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getProxyForDatabase(ctx context.Context, tenantID uuid.UUID, databaseID uuid.UUID) (*companyconfig.SQLShimProxy, error) {
	pager, err := companyconfig.NewSQLShimProxyPaginatorFromOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		proxies, pr, err := h.storage.ListSQLShimProxiesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, proxy := range proxies {
			if proxy.DatabaseID == databaseID && proxy.TenantID == tenantID {
				return &proxy, nil
			}
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	return nil, ucerr.Friendlyf(nil, "proxy for database %s not found", databaseID)
}

// DatabaseWithProxy is a struct that combines a SQLShimDatabase with a proxy host and port
type DatabaseWithProxy struct {
	userstore.SQLShimDatabase
	ProxyHost string `json:"proxy_host"`
	ProxyPort int    `json:"proxy_port"`
}

// ListDatabaseWithProxyResponse is a struct that combines a list of DatabaseWithProxy objects with pagination fields
type ListDatabaseWithProxyResponse struct {
	Data []DatabaseWithProxy `json:"data"`
	pagination.ResponseFields
}

// TODO: this "list" function is a bit of a misnomer, as it only returns one database, until we support multiple databases per tenant
func (h *handler) listTenantUserStoreDatabases(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.ListDatabases(ctx, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ret := ListDatabaseWithProxyResponse{
		ResponseFields: resp.ResponseFields,
	}
	for _, db := range resp.Data {
		respWithProxy := DatabaseWithProxy{
			SQLShimDatabase: db,
		}
		if proxy, err := h.getProxyForDatabase(ctx, tenantID, db.ID); err == nil {
			respWithProxy.ProxyHost = proxy.Host
			respWithProxy.ProxyPort = proxy.Port
		}
		ret.Data = append(ret.Data, respWithProxy)
	}

	jsonapi.Marshal(w, ret)
}

func (h *handler) getTenantUserStoreDatabase(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	dataTypeID uuid.UUID,
) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.GetDatabase(ctx, dataTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	respWithProxy := DatabaseWithProxy{
		SQLShimDatabase: *resp,
	}

	proxy, err := h.getProxyForDatabase(ctx, tenantID, dataTypeID)
	if err == nil {
		respWithProxy.ProxyHost = proxy.Host
		respWithProxy.ProxyPort = proxy.Port
	}

	jsonapi.Marshal(w, respWithProxy)
}

func (h *handler) saveTenantUserStoreDatabase(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()

	var req DatabaseWithProxy
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create databases"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.CreateDatabase(ctx, userstore.SQLShimDatabase{
		Name:     req.Name,
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.ensureUCAdmin(r); err == nil {
		if _, err := h.saveProxyForDatabase(ctx, tenantID, resp.ID, req.ProxyHost, req.ProxyPort); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, DatabaseWithProxy{
		SQLShimDatabase: *resp,
		ProxyHost:       req.ProxyHost,
		ProxyPort:       req.ProxyPort,
	})
}

func (h *handler) deleteTenantUserStoreDatabase(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	databaseID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete databases"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteDatabase(ctx, databaseID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if proxy, err := h.getProxyForDatabase(ctx, tenantID, databaseID); err == nil {
		if err := h.storage.DeleteSQLShimProxy(ctx, proxy.ID); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateTenantUserStoreDatabase(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	databaseID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update databases"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req DatabaseWithProxy
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.ID != databaseID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "database ID in request body does not match database ID in URL"))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.UpdateDatabase(ctx, databaseID, userstore.SQLShimDatabase{
		ID:       req.ID,
		Name:     req.Name,
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.ensureUCAdmin(r); err == nil {
		if _, err := h.saveProxyForDatabase(ctx, tenantID, databaseID, req.ProxyHost, req.ProxyPort); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, DatabaseWithProxy{
		SQLShimDatabase: *resp,
		ProxyHost:       req.ProxyHost,
		ProxyPort:       req.ProxyPort,
	})
}

// TODO: this "list" function is a bit of a misnomer, as it only returns one object store, until we support multiple object stores per tenant
func (h *handler) listTenantUserStoreObjectStores(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.ListObjectStores(ctx, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getTenantUserStoreObjectStore(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	dataTypeID uuid.UUID,
) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.GetObjectStore(ctx, dataTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

type saveObjectStoreRequest struct {
	ObjectStore          userstore.ShimObjectStore `json:"object_store"`
	ComposedAccessPolicy *policy.AccessPolicy      `json:"composed_access_policy,omitempty" validate:"skip"`
}

func (h *handler) saveTenantUserStoreObjectStore(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()

	var req saveObjectStoreRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create object stores"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.ObjectStore.ID == uuid.Nil {
		req.ObjectStore.ID = uuid.Must(uuid.NewV4())
	}

	newAP, err := createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, objectStoreAccessPolicyNamePrefix, req.ObjectStore.ID, req.ObjectStore.Name)
	if err != nil || newAP == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
		return
	}
	req.ObjectStore.AccessPolicy = userstore.ResourceID{ID: newAP.ID}

	resp, err := idpClient.CreateObjectStore(ctx, req.ObjectStore)
	if err != nil {
		if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
			uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteTenantUserStoreObjectStore(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	objectStoreID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete object stores"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteObjectStore(ctx, objectStoreID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateTenantUserStoreObjectStore(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	objectStoreID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update object stores"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req saveObjectStoreRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.ObjectStore.ID != objectStoreID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "object store ID in request body does not match object store ID in URL"))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the access policy (and auto-generate a new one if the existing one is not autogenerated)
	bypassAPCreation := false
	ap, err := idpClient.GetAccessPolicy(ctx, userstore.ResourceID{ID: req.ObjectStore.AccessPolicy.ID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if ap.IsAutogenerated {
		bypassAPCreation = true
		if _, err := updateExistingAccessPolicyIfNeeded(ctx, idpClient, req.ComposedAccessPolicy, req.ObjectStore.AccessPolicy.ID); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	var newAP *policy.AccessPolicy
	if !bypassAPCreation {
		newAP, err = createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, objectStoreAccessPolicyNamePrefix, req.ObjectStore.ID, req.ObjectStore.Name)
		if err != nil || newAP == nil {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
			return
		}
		req.ObjectStore.AccessPolicy = userstore.ResourceID{ID: newAP.ID}
	}

	resp, err := idpClient.UpdateObjectStore(ctx, objectStoreID, req.ObjectStore)
	if err != nil {
		if newAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}

		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) listTenantUserStoreDataTypes(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.ListDataTypes(ctx, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getTenantUserStoreDataType(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	dataTypeID uuid.UUID,
) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.GetDataType(ctx, dataTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) saveTenantUserStoreDataType(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create data types"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req idp.CreateDataTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.CreateDataType(ctx, req.DataType, idp.IfNotExists())
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteTenantUserStoreDataType(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	dataTypeID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete data types"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteDataType(ctx, dataTypeID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateTenantUserStoreDataType(
	w http.ResponseWriter,
	r *http.Request,
	tenantID uuid.UUID,
	dataTypeID uuid.UUID,
) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update data types"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req idp.UpdateDataTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.DataType.ID != dataTypeID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "data type ID in request body does not match data type ID in URL"))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.UpdateDataType(ctx, dataTypeID, req.DataType)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) listTenantUserStoreColumns(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.ListColumns(ctx, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getTenantUserStoreColumn(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, columnID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.GetColumn(ctx, columnID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

// ConsoleSaveColumnRequest is the format for saving a column in console
type ConsoleSaveColumnRequest struct {
	Column                    userstore.Column     `json:"column"`
	ComposedAccessPolicy      *policy.AccessPolicy `json:"composed_access_policy,omitempty" validate:"skip"`
	ComposedTokenAccessPolicy *policy.AccessPolicy `json:"composed_token_access_policy,omitempty" validate:"skip"`
}

func (h *handler) saveTenantUserStoreColumn(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create columns"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req ConsoleSaveColumnRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	newAP, err := createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, columnAccessPolicyNamePrefix, req.Column.ID, req.Column.Name)
	if err != nil || newAP == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
		return
	}
	req.Column.AccessPolicy = userstore.ResourceID{ID: newAP.ID}

	tokenAP, err := createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedTokenAccessPolicy, columnTokenAccessPolicyNamePrefix, req.Column.ID, req.Column.Name)
	if err != nil {
		if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
			uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if tokenAP != nil {
		req.Column.DefaultTokenAccessPolicy = userstore.ResourceID{ID: tokenAP.ID}
	}

	resp, err := idpClient.CreateColumn(ctx, req.Column, idp.IfNotExists())
	if err != nil {
		if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
			uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
		}

		if tokenAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, tokenAP.ID, tokenAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}

		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteTenantUserStoreColumn(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, columnID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete columns"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteColumn(ctx, columnID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateTenantUserStoreColumn(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, columnID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update columns"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req ConsoleSaveColumnRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Column.ID != columnID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "column ID in request body does not match column ID in URL"))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the access policy (and auto-generate a new one if the existing one is not autogenerated)
	bypassAPCreation := false
	ap, err := idpClient.GetAccessPolicy(ctx, userstore.ResourceID{ID: req.Column.AccessPolicy.ID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if ap.IsAutogenerated {
		bypassAPCreation = true
		if _, err := updateExistingAccessPolicyIfNeeded(ctx, idpClient, req.ComposedAccessPolicy, req.Column.AccessPolicy.ID); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	var newAP *policy.AccessPolicy
	if !bypassAPCreation {
		newAP, err = createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, columnAccessPolicyNamePrefix, req.Column.ID, req.Column.Name)
		if err != nil || newAP == nil {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
			return
		}
		req.Column.AccessPolicy = userstore.ResourceID{ID: newAP.ID}
	}

	bypassAPCreation = false
	if req.Column.DefaultTokenAccessPolicy.Validate() == nil {
		ap, err := idpClient.GetAccessPolicy(ctx, req.Column.DefaultTokenAccessPolicy)
		if err != nil {
			// log an error but continue, since a new AP will be created
			uclog.Errorf(ctx, "error getting existing access policy: %v", err)
		} else if ap.IsAutogenerated {
			bypassAPCreation = true
			if _, err = updateExistingAccessPolicyIfNeeded(ctx, idpClient, req.ComposedTokenAccessPolicy, ap.ID); err != nil {
				if newAP != nil {
					if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
						uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
					}
				}

				jsonapi.MarshalError(ctx, w, err)
				return
			}
		}
	}

	var tokenAP *policy.AccessPolicy
	if !bypassAPCreation {
		tokenAP, err = createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedTokenAccessPolicy, columnTokenAccessPolicyNamePrefix, req.Column.ID, req.Column.Name)
		if err != nil {
			if newAP != nil {
				if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
					uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
				}
			}

			jsonapi.MarshalError(ctx, w, err)
			return
		}
		if tokenAP != nil {
			req.Column.DefaultTokenAccessPolicy = userstore.ResourceID{ID: tokenAP.ID}
		}
	}

	resp, err := idpClient.UpdateColumn(ctx, columnID, req.Column)
	if err != nil {
		if newAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}

		if tokenAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, tokenAP.ID, tokenAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}

		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

// ConsoleAccessorColumn is the format for specifying column/transformer pairs in console
type ConsoleAccessorColumn struct {
	ID                      uuid.UUID `json:"id"`
	Name                    string    `json:"name"`
	Table                   string    `json:"table"`
	DataTypeID              uuid.UUID `json:"data_type_id"`
	DataTypeName            string    `json:"data_type_name"`
	IsArray                 bool      `json:"is_array"`
	TransformerID           uuid.UUID `json:"transformer_id"`
	TransformerName         string    `json:"transformer_name"`
	TokenAccessPolicyID     uuid.UUID `json:"token_access_policy_id"`
	TokenAccessPolicyName   string    `json:"token_access_policy_name"`
	DefaultTransformerName  string    `json:"default_transformer_name"`
	DefaultAccessPolicyID   uuid.UUID `json:"default_access_policy_id"`
	DefaultAccessPolicyName string    `json:"default_access_policy_name"`
}

// AccessorResponse is the response returned by the Console API for an accessor.
type AccessorResponse struct {
	ID                                uuid.UUID                    `json:"id"`
	Name                              string                       `json:"name" validate:"notempty"`
	Description                       string                       `json:"description"`
	Version                           int                          `json:"version"`
	DataLifeCycleState                userstore.DataLifeCycleState `json:"data_life_cycle_state"`
	SelectorConfig                    userstore.UserSelectorConfig `json:"selector_config"`
	Columns                           []ConsoleAccessorColumn      `json:"columns"`
	AccessPolicy                      userstore.ResourceID         `json:"access_policy" validate:"skip"`
	Purposes                          []userstore.ResourceID       `json:"purposes" validate:"skip"`
	IsSystem                          bool                         `json:"is_system"`
	IsAuditLogged                     bool                         `json:"is_audit_logged"`
	AreColumnAccessPoliciesOverridden bool                         `json:"are_column_access_policies_overridden"`
	UseSearchIndex                    bool                         `json:"use_search_index"`
}

// SaveAccessorRequest accepts columns and transformers and access policy_id
type SaveAccessorRequest struct {
	ID                                uuid.UUID                    `json:"id"`
	Name                              string                       `json:"name" validate:"notempty"`
	Description                       string                       `json:"description"`
	Version                           int                          `json:"version"`
	DataLifeCycleState                userstore.DataLifeCycleState `json:"data_life_cycle_state"`
	SelectorConfig                    userstore.UserSelectorConfig `json:"selector_config"`
	Columns                           []ConsoleAccessorColumn      `json:"columns"`
	AccessPolicyID                    *uuid.UUID                   `json:"access_policy_id,omitempty" validate:"skip"`
	Purposes                          []userstore.ResourceID       `json:"purposes" validate:"skip"`
	ComposedAccessPolicy              *policy.AccessPolicy         `json:"composed_access_policy,omitempty" validate:"skip"`
	ComposedTokenAccessPolicy         *policy.AccessPolicy         `json:"composed_token_access_policy,omitempty" validate:"skip"`
	IsAuditLogged                     bool                         `json:"is_audit_logged"`
	AreColumnAccessPoliciesOverridden bool                         `json:"are_column_access_policies_overridden"`
	UseSearchIndex                    bool                         `json:"use_search_index"`
}

func consoleAccessorToUserstoreAccessor(m SaveAccessorRequest) userstore.Accessor {
	cAccessor := userstore.Accessor{
		ID:                                m.ID,
		Name:                              m.Name,
		Description:                       m.Description,
		Version:                           m.Version,
		DataLifeCycleState:                m.DataLifeCycleState,
		SelectorConfig:                    m.SelectorConfig,
		Purposes:                          m.Purposes,
		IsAuditLogged:                     m.IsAuditLogged,
		AreColumnAccessPoliciesOverridden: m.AreColumnAccessPoliciesOverridden,
		UseSearchIndex:                    m.UseSearchIndex,
	}
	if m.AccessPolicyID != nil {
		cAccessor.AccessPolicy = userstore.ResourceID{
			ID: *m.AccessPolicyID,
		}
	}
	for _, c := range m.Columns {
		cAccessor.Columns = append(cAccessor.Columns, userstore.ColumnOutputConfig{
			Column: userstore.ResourceID{
				ID:   c.ID,
				Name: c.Name,
			},
			Transformer: userstore.ResourceID{
				ID:   c.TransformerID,
				Name: c.TransformerName,
			},
			TokenAccessPolicy: userstore.ResourceID{
				ID:   c.TokenAccessPolicyID,
				Name: c.TokenAccessPolicyName,
			},
		})
	}

	return cAccessor
}

func userstoreAccessorToConsoleAccessor(a *userstore.Accessor) AccessorResponse {
	cAccessor := AccessorResponse{
		ID:                                a.ID,
		Name:                              a.Name,
		Description:                       a.Description,
		Version:                           a.Version,
		DataLifeCycleState:                a.DataLifeCycleState,
		SelectorConfig:                    a.SelectorConfig,
		AccessPolicy:                      a.AccessPolicy,
		Purposes:                          a.Purposes,
		IsSystem:                          a.IsSystem,
		IsAuditLogged:                     a.IsAuditLogged,
		AreColumnAccessPoliciesOverridden: a.AreColumnAccessPoliciesOverridden,
		UseSearchIndex:                    a.UseSearchIndex,
	}
	for _, c := range a.Columns {
		cAccessor.Columns = append(cAccessor.Columns, ConsoleAccessorColumn{
			ID:                    c.Column.ID,
			Name:                  c.Column.Name,
			TransformerID:         c.Transformer.ID,
			TransformerName:       c.Transformer.Name,
			TokenAccessPolicyID:   c.TokenAccessPolicy.ID,
			TokenAccessPolicyName: c.TokenAccessPolicy.Name,
		})
	}

	return cAccessor
}

// ListConsoleAccessorResponse returns paginated console accessors.
type ListConsoleAccessorResponse struct {
	Data []AccessorResponse `json:"data"`
	pagination.ResponseFields
}

func (h *handler) listTenantAccessors(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	accessors, err := idpClient.ListAccessors(ctx, false, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	consoleAccessors := make([]AccessorResponse, len(accessors.Data))
	for i := range accessors.Data {
		consoleAccessors[i] = userstoreAccessorToConsoleAccessor(&accessors.Data[i])
		if err := h.addDataToAccessorColumns(ctx, columnMap, consoleAccessors[i].Columns); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, ListConsoleAccessorResponse{Data: consoleAccessors, ResponseFields: accessors.ResponseFields})
}

func (h *handler) addDataToAccessorColumns(ctx context.Context, columnMap map[uuid.UUID]userstore.Column, columns []ConsoleAccessorColumn) error {
	for i, col := range columns {
		colData, ok := columnMap[col.ID]
		if !ok {
			uclog.Warningf(ctx, "Column %s not found in column map", col.ID)
			continue
		}

		columns[i].Table = colData.Table
		columns[i].DataTypeID = colData.DataType.ID
		columns[i].DataTypeName = colData.DataType.Name
		columns[i].IsArray = colData.IsArray
		columns[i].DefaultAccessPolicyName = colData.AccessPolicy.Name
		columns[i].DefaultTransformerName = colData.DefaultTransformer.Name
	}
	return nil
}

func (h *handler) getTenantAccessor(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, accessorID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var accessor *userstore.Accessor
	if versionString := r.URL.Query().Get("version"); versionString != "" {
		version, vErr := strconv.Atoi(versionString)
		if vErr != nil {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid accessor version specified"))
			return
		}

		accessor, err = idpClient.GetAccessorByVersion(ctx, accessorID, version)
	} else {
		accessor, err = idpClient.GetAccessor(ctx, accessorID)
	}
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreAccessorToConsoleAccessor(accessor)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToAccessorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func updateExistingAccessPolicyIfNeeded(ctx context.Context, idpClient *idp.Client, modifiedPolicy *policy.AccessPolicy, existingPolicyID uuid.UUID) (*policy.AccessPolicy, error) {
	var updatedAP *policy.AccessPolicy
	if modifiedPolicy != nil {
		existingAP, err := idpClient.GetAccessPolicy(ctx, userstore.ResourceID{ID: existingPolicyID})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if !existingAP.EqualsIgnoringNilID(*modifiedPolicy) {
			modifiedPolicy.ID = existingPolicyID
			updatedAP, err = idpClient.UpdateAccessPolicy(ctx, *modifiedPolicy)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
		}
	}
	return updatedAP, nil
}

func nameForAccessPolicy(prefix string, parentID uuid.UUID, parentName string) string {
	name := fmt.Sprintf("%s_%s-%s_%s", prefix, parentID, crypto.MustRandomDigits(6), parentName)
	if len(name) > 128 {
		name = name[:128]
	}
	return name
}

const columnAccessPolicyNamePrefix = "AccessPolicyForColumn"
const columnTokenAccessPolicyNamePrefix = "TokenAccessPolicyForColumn"
const accessorAccessPolicyNamePrefix = "AccessPolicyForAccessor"
const mutatorAccessPolicyNamePrefix = "AccessPolicyForMutator"
const objectStoreAccessPolicyNamePrefix = "AccessPolicyForObjectStore"

func createAutogeneratedAccessPolicy(ctx context.Context, idpClient *idp.Client, accessPolicy *policy.AccessPolicy, prefix string, parentID uuid.UUID, parentName string) (*policy.AccessPolicy, error) {
	if accessPolicy != nil && len(accessPolicy.Components) > 0 {
		accessPolicy.ID = uuid.Nil
		accessPolicy.Name = nameForAccessPolicy(prefix, parentID, parentName)
		accessPolicy.IsAutogenerated = true
		accessPolicy.IsSystem = false
		createdAP, err := idpClient.CreateAccessPolicy(ctx, *accessPolicy)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return createdAP, nil
	}
	return nil, nil
}

func (h *handler) createTenantAccessor(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create accessors"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req SaveAccessorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.AccessPolicyID != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "access_policy_id should not be specified in accessor creation"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	if req.ComposedAccessPolicy == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "composed_access_policy is required"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ap, err := createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, accessorAccessPolicyNamePrefix, req.ID, req.Name)
	if err != nil || ap == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
		return
	}
	req.AccessPolicyID = &ap.ID

	accessor, err := idpClient.CreateAccessor(ctx, consoleAccessorToUserstoreAccessor(req))
	if err != nil {
		if errDel := idpClient.DeleteAccessPolicy(ctx, ap.ID, ap.Version); errDel != nil {
			uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreAccessorToConsoleAccessor(accessor)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToAccessorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) updateTenantAccessor(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, accessorID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update accessors"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req SaveAccessorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.ID != accessorID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "invalid `id` specified: %v", accessorID), jsonapi.Code(http.StatusBadRequest))
		return
	}

	if req.AccessPolicyID == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "accessor missing access_policy_id"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the access policy (and auto-generate a new one if the existing one is not autogenerated)
	bypassAPCreation := false
	ap, err := idpClient.GetAccessPolicy(ctx, userstore.ResourceID{ID: *req.AccessPolicyID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if ap.IsAutogenerated {
		bypassAPCreation = true
		if _, err := updateExistingAccessPolicyIfNeeded(ctx, idpClient, req.ComposedAccessPolicy, *req.AccessPolicyID); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	var newAP *policy.AccessPolicy
	if !bypassAPCreation {
		newAP, err = createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, accessorAccessPolicyNamePrefix, req.ID, req.Name)
		if err != nil || newAP == nil {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
			return
		}
		req.AccessPolicyID = &newAP.ID
	}

	accessor, err := idpClient.UpdateAccessor(ctx, accessorID, consoleAccessorToUserstoreAccessor(req))
	if err != nil {
		if newAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreAccessorToConsoleAccessor(accessor)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToAccessorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteTenantAccessor(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, accessorID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete accessors"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteAccessor(ctx, accessorID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) executeTenantAccessor(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can execute accessor from console"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req idp.ExecuteAccessorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := idpClient.ExecuteAccessor(ctx, req.AccessorID, req.Context, req.SelectorValues, options, idp.Debug())
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getColumnMap(ctx context.Context, idpClient *idp.Client) (map[uuid.UUID]userstore.Column, error) {
	columns := map[uuid.UUID]userstore.Column{}

	cursor := pagination.CursorBegin
	for {
		columnsResp, err := idpClient.ListColumns(ctx, idp.Pagination(pagination.StartingAfter(cursor), pagination.Limit(pagination.MaxLimit)))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, column := range columnsResp.Data {
			columns[column.ID] = column
		}

		if !columnsResp.HasNext {
			break
		}
		cursor = columnsResp.Next
	}

	return columns, nil
}

// ConsoleMutatorColumn is the format for specifying column/normalizer pairs in console
type ConsoleMutatorColumn struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Table          string    `json:"table"`
	DataTypeID     uuid.UUID `json:"data_type_id"`
	DataTypeName   string    `json:"data_type_name"`
	IsArray        bool      `json:"is_array"`
	NormalizerID   uuid.UUID `json:"normalizer_id"`
	NormalizerName string    `json:"normalizer_name"`
}

// MutatorResponse holds full data for columns, access policy, and normalizer, not just the IDs
type MutatorResponse struct {
	ID             uuid.UUID                    `json:"id"`
	Name           string                       `json:"name"`
	Description    string                       `json:"description"`
	Version        int                          `json:"version"`
	SelectorConfig userstore.UserSelectorConfig `json:"selector_config"`
	Columns        []ConsoleMutatorColumn       `json:"columns"`
	AccessPolicy   userstore.ResourceID         `json:"access_policy"`
	IsSystem       bool                         `json:"is_system"`
}

// SaveMutatorRequest accepts columns and normalizer and access policy_id
type SaveMutatorRequest struct {
	ID                   uuid.UUID                    `json:"id"`
	Name                 string                       `json:"name"`
	Description          string                       `json:"description"`
	Version              int                          `json:"version"`
	SelectorConfig       userstore.UserSelectorConfig `json:"selector_config"`
	Columns              []ConsoleMutatorColumn       `json:"columns"`
	AccessPolicyID       *uuid.UUID                   `json:"access_policy_id"`
	ComposedAccessPolicy *policy.AccessPolicy         `json:"composed_access_policy,omitempty" validate:"skip"`
}

func consoleMutatorToUserstoreMutator(m SaveMutatorRequest) userstore.Mutator {
	cMutator := userstore.Mutator{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Version:        m.Version,
		SelectorConfig: m.SelectorConfig,
	}
	if m.AccessPolicyID != nil {
		cMutator.AccessPolicy = userstore.ResourceID{
			ID: *m.AccessPolicyID,
		}
	}
	for _, c := range m.Columns {
		cMutator.Columns = append(cMutator.Columns, userstore.ColumnInputConfig{
			Column: userstore.ResourceID{
				ID:   c.ID,
				Name: c.Name,
			},
			Normalizer: userstore.ResourceID{
				ID:   c.NormalizerID,
				Name: c.NormalizerName,
			},
		})
	}

	return cMutator
}

func userstoreMutatorToConsoleMutator(m *userstore.Mutator) MutatorResponse {
	cMutator := MutatorResponse{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		Version:        m.Version,
		SelectorConfig: m.SelectorConfig,
		AccessPolicy:   m.AccessPolicy,
		Columns:        []ConsoleMutatorColumn{},
		IsSystem:       m.IsSystem,
	}
	for _, c := range m.Columns {
		cMutator.Columns = append(cMutator.Columns, ConsoleMutatorColumn{
			ID:             c.Column.ID,
			Name:           c.Column.Name,
			NormalizerID:   c.Normalizer.ID,
			NormalizerName: c.Normalizer.Name,
		})
	}

	return cMutator
}

// ListConsoleMutatorResponse returns paginated console mutators.
type ListConsoleMutatorResponse struct {
	Data []MutatorResponse `json:"data"`
	pagination.ResponseFields
}

func (h *handler) listTenantMutators(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mutators, err := idpClient.ListMutators(ctx, false, idp.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	consoleMutators := make([]MutatorResponse, len(mutators.Data))
	for i := range mutators.Data {
		consoleMutators[i] = userstoreMutatorToConsoleMutator(&mutators.Data[i])

		if err := h.addDataToMutatorColumns(ctx, columnMap, consoleMutators[i].Columns); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, ListConsoleMutatorResponse{Data: consoleMutators, ResponseFields: mutators.ResponseFields})
}

func (h *handler) addDataToMutatorColumns(ctx context.Context, columnMap map[uuid.UUID]userstore.Column, columns []ConsoleMutatorColumn) error {

	for i, col := range columns {
		colData, ok := columnMap[col.ID]
		if !ok {
			uclog.Warningf(ctx, "Column %s not found in column map", col.ID)
			continue
		}

		columns[i].Table = colData.Table
		columns[i].DataTypeID = colData.DataType.ID
		columns[i].DataTypeName = colData.DataType.Name
		columns[i].IsArray = colData.IsArray
	}
	return nil
}

func (h *handler) getTenantMutator(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, mutatorID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var mutator *userstore.Mutator
	if versionString := r.URL.Query().Get("version"); versionString != "" {
		version, vErr := strconv.Atoi(versionString)
		if vErr != nil {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid mutator version specified"))
			return
		}

		mutator, err = idpClient.GetMutatorByVersion(ctx, mutatorID, version)
	} else {
		mutator, err = idpClient.GetMutator(ctx, mutatorID)
	}
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreMutatorToConsoleMutator(mutator)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToMutatorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) createTenantMutator(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can create mutators"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req SaveMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.AccessPolicyID != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "access_policy_id should not be specified in mutator creation"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	if req.ComposedAccessPolicy == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "composed_access_policy is required"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	ap, err := createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, mutatorAccessPolicyNamePrefix, req.ID, req.Name)
	if err != nil || ap == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
		return
	}
	req.AccessPolicyID = &ap.ID

	mutator, err := idpClient.CreateMutator(ctx, consoleMutatorToUserstoreMutator(req))
	if err != nil {
		if errDel := idpClient.DeleteAccessPolicy(ctx, ap.ID, ap.Version); errDel != nil {
			uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreMutatorToConsoleMutator(mutator)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToMutatorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) updateTenantMutator(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, mutatorID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can update mutators"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req SaveMutatorRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.ID != mutatorID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "invalid `id` specified: %v", mutatorID), jsonapi.Code(http.StatusBadRequest))
		return
	}

	if req.AccessPolicyID == nil {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "mutator missing access_policy_id"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the access policy (and auto-generate a new one if the existing one is not autogenerated)
	bypassAPCreation := false
	ap, err := idpClient.GetAccessPolicy(ctx, userstore.ResourceID{ID: *req.AccessPolicyID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if ap.IsAutogenerated {
		bypassAPCreation = true
		if _, err := updateExistingAccessPolicyIfNeeded(ctx, idpClient, req.ComposedAccessPolicy, *req.AccessPolicyID); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	var newAP *policy.AccessPolicy
	if !bypassAPCreation {
		newAP, err = createAutogeneratedAccessPolicy(ctx, idpClient, req.ComposedAccessPolicy, mutatorAccessPolicyNamePrefix, req.ID, req.Name)
		if err != nil || newAP == nil {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "error creating access policy"))
			return
		}
		req.AccessPolicyID = &newAP.ID
	}

	mutator, err := idpClient.UpdateMutator(ctx, mutatorID, consoleMutatorToUserstoreMutator(req))
	if err != nil {
		if newAP != nil {
			if errDel := idpClient.DeleteAccessPolicy(ctx, newAP.ID, newAP.Version); errDel != nil {
				uclog.Errorf(ctx, "error deleting access policy: %v", errDel)
			}
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := userstoreMutatorToConsoleMutator(mutator)

	columnMap, err := h.getColumnMap(ctx, idpClient)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := h.addDataToMutatorColumns(ctx, columnMap, resp.Columns); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) deleteTenantMutator(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, mutatorID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can delete mutators"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeleteMutator(ctx, mutatorID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listTenantPurposes(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := idpClient.ListPurposes(ctx, options)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) getTenantPurpose(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, purposeID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	purpose, err := idpClient.GetPurpose(ctx, purposeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, purpose)
}

func (h *handler) createTenantPurpose(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "User must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req idp.CreatePurposeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	purpose, err := idpClient.CreatePurpose(ctx, req.Purpose)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonapi.Marshal(w, purpose)
}

func (h *handler) updateTenantPurpose(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, purposeID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "User must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req idp.UpdatePurposeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Purpose.ID != purposeID {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "purpose ID in request body does not match purpose ID in URL"))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	purpose, err := idpClient.UpdatePurpose(ctx, req.Purpose)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, purpose)
}

func (h *handler) deleteTenantPurpose(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, purposeID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "User must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := idpClient.DeletePurpose(ctx, purposeID); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) getTenantUserConsentedPurposes(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "Caller must be an admin of the tenant"), jsonapi.Code(http.StatusForbidden))
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := idpClient.GetConsentedPurposesForUser(ctx, userID, nil)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp.Data)
}
