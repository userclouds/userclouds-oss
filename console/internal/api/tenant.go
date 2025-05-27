package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"

	"github.com/gofrs/uuid"
	xrv "github.com/mattermost/xml-roundtrip-validator"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/internal/auth"
	"userclouds.com/idp"
	"userclouds.com/idp/datamapping"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/samlconfig"
	"userclouds.com/plex/manager"
	"userclouds.com/userevent"
	"userclouds.com/worker"
)

func (h *handler) ensureEmployeeAccessToTenant(r *http.Request, tenant *companyconfig.Tenant) (isAdmin bool, err error) {
	ctx := r.Context()

	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenant.ID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	if _, err := manager.NewFromDB(tenantDB, h.cacheConfig).GetEmployeeApp(ctx, tenant.ID); err != nil {
		return false, ucerr.Wrap(err)
	}
	tokenSource, err := m2m.GetM2MTokenSource(ctx, tenant.ID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx,
		h.storage,
		tenant.ID,
		tenant.TenantURL,
		tokenSource,
		apiclient.ClientCacheConfig(h.cacheConfig))
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	// Check that the logged in user has an authz object on the tenant
	tenantRBACClient := authz.NewRBACClient(authzClient)
	user, err := tenantRBACClient.GetUser(ctx, userID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	companyGroup, err := tenantRBACClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	roles, err := companyGroup.GetUserRoles(ctx, *user)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	foundMember := false
	for _, role := range roles {
		if role == ucauthz.AdminRole {
			return true, nil
		} else if role == ucauthz.MemberRole {
			foundMember = true
		}
	}
	if foundMember {
		return false, nil
	}

	return false, ucerr.Friendlyf(nil, "You do not have access to this tenant")
}

// TODO: eventually may want an M2M token for each specific tenant, NOT the console tenant,
// but for now we are going to have our services honor tenant-signed AND console-signed tokens as long as audience matches.
func (h *handler) newIDPMgmtClient(ctx context.Context, tenant *companyconfig.Tenant) (*idp.ManagementClient, error) {
	_, refreshToken := auth.MustGetAccessTokens(ctx)
	ts, err := auth.NewEmployeeTokenSource(ctx, tenant, refreshToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	idpClient, err := idp.NewManagementClient(tenant.TenantURL, ts, security.PassXForwardedFor())
	return idpClient, ucerr.Wrap(err)
}

func (h *handler) newIDPClient(ctx context.Context, tenantID uuid.UUID) (*idp.Client, error) {
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	_, refreshToken := auth.MustGetAccessTokens(ctx)
	ts, err := auth.NewEmployeeTokenSource(ctx, tenant, refreshToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	idpClient, err := idp.NewClient(tenant.TenantURL, idp.JSONClient(ts, security.PassXForwardedFor()))
	return idpClient, ucerr.Wrap(err)
}

func (h *handler) newDatamappingClient(ctx context.Context, tenantID uuid.UUID) (*datamapping.Client, error) {
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	_, refreshToken := auth.MustGetAccessTokens(ctx)
	ts, err := auth.NewEmployeeTokenSource(ctx, tenant, refreshToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	dmClient, err := datamapping.NewClient(tenant.TenantURL, datamapping.JSONClient(ts, security.PassXForwardedFor()))
	return dmClient, ucerr.Wrap(err)
}

// *companyconfig.Tenant
// func (h *handler) newAuthZClient(ctx context.Context, tenantURL string, tenantID uuid.UUID) (*authz.Client, error) {

func (h *handler) newAuthZClient(ctx context.Context, tenant *companyconfig.Tenant) (*authz.Client, error) {
	_, refreshToken := auth.MustGetAccessTokens(ctx)
	ts, err := auth.NewEmployeeTokenSource(ctx, tenant, refreshToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	authZClient, err := apiclient.NewAuthzClientWithTokenSource(ctx,
		h.storage,
		tenant.ID,
		tenant.TenantURL,
		ts,
		apiclient.ClientCacheConfig(h.cacheConfig))
	return authZClient, ucerr.Wrap(err)
}

func (h *handler) newUserEventClient(ctx context.Context, tenant *companyconfig.Tenant) (*userevent.Client, error) {
	tokenSource, err := m2m.GetM2MTokenSource(ctx, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	userEventClient, err := userevent.NewClient(tenant.TenantURL, tokenSource, security.PassXForwardedFor())
	return userEventClient, ucerr.Wrap(err)
}

func (h *handler) newAuditLogClient(ctx context.Context, tenant *companyconfig.Tenant) (*auditlog.Client, error) {
	tokenSource, err := m2m.GetM2MTokenSource(ctx, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	auditLogClient, err := auditlog.NewClient(tenant.TenantURL, tokenSource)
	return auditLogClient, ucerr.Wrap(err)
}

func (h *handler) newTokenizerClient(ctx context.Context, tenantID uuid.UUID) (*idp.TokenizerClient, error) {
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	_, refreshToken := auth.MustGetAccessTokens(ctx)
	ts, err := auth.NewEmployeeTokenSource(ctx, tenant, refreshToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return idp.NewTokenizerClient(tenant.TenantURL, idp.JSONClient(ts)), nil
}

func (h *handler) deprovisionTenant(ctx context.Context, tenantID uuid.UUID) error {

	pt, err := tenantProvisioning.NewProvisionableTenantFromExisting(ctx, "DeprovisionTenantAPI", tenantID, h.storage, &h.provisionDBInfo.companyDB, &h.provisionDBInfo.logDB, h.cacheConfig)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := pt.Cleanup(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (h *handler) newTenantNestedAuthorizer() uchttp.NestedCollectionAuthorizer {
	return &uchttp.NestedMethodAuthorizer{
		GetAllF: func(r *http.Request, companyID uuid.UUID) error {
			return nil
		},
		GetOneF: func(r *http.Request, companyID, tenantID uuid.UUID) error {
			return nil
		},
		PostF: func(r *http.Request, companyID uuid.UUID) error {
			return ucerr.Wrap(h.ensureCompanyAdmin(r, companyID))
		},
		PutF: func(r *http.Request, companyID uuid.UUID, id uuid.UUID) error {
			return ucerr.Wrap(h.ensureCompanyAdmin(r, companyID))
		},
		DeleteF: func(r *http.Request, companyID uuid.UUID, id uuid.UUID) error {
			return ucerr.Wrap(h.ensureCompanyAdmin(r, companyID))
		},
	}
}

func (h *handler) newTenantAuthorizer() uchttp.CollectionAuthorizer {
	ensureCompanyAdmin := func(r *http.Request, id uuid.UUID) error {
		ctx := r.Context()
		tenant, err := h.storage.GetTenant(ctx, id)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if err := h.ensureCompanyAdmin(r, tenant.CompanyID); err != nil {
			return ucerr.Wrap(err)
		}
		return nil
	}

	ensureCompanyEmployee := func(r *http.Request, id uuid.UUID) error {
		ctx := r.Context()
		tenant, err := h.storage.GetTenant(ctx, id)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if err := h.ensureCompanyEmployee(r, tenant.CompanyID); err != nil {
			return ucerr.Wrap(err)
		}
		return nil
	}

	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			// Fine-grained validation happens in method (depends on query params)
			return nil
		},
		PostF: func(r *http.Request) error {
			// Need to inspect payload to validate in method
			return nil
		},
		PutF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(ensureCompanyAdmin(r, id))
		},
		DeleteF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(ensureCompanyAdmin(r, id))
		},
		NestedF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(ensureCompanyEmployee(r, id))
		},
	}
}

// TenantInfo is the HTTP response format for listTenants
type TenantInfo struct {
	companyconfig.Tenant
	IsConsoleTenant bool `json:"is_console_tenant"`
}

// SelectedTenantInfo is the HTTP response format for getTenant, updateTenant, and createTenant
type SelectedTenantInfo struct {
	companyconfig.Tenant
	IsConsoleTenant bool                `json:"is_console_tenant"`
	IsAdmin         bool                `json:"is_admin"`
	IsMember        bool                `json:"is_member"`
	UserRegions     []region.DataRegion `json:"user_regions"`
}

func (h *handler) listTenants(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) {
	ctx := r.Context()

	tenants, err := h.storage.ListTenantsForCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		} else {
			jsonapi.MarshalError(ctx, w, err)
		}
		return
	}

	tenantsAccessibleToUser := []TenantInfo{}
	for _, tenant := range tenants {
		tenantsAccessibleToUser = append(tenantsAccessibleToUser, TenantInfo{
			Tenant:          tenant,
			IsConsoleTenant: tenant.ID == h.consoleTenant.TenantID,
		})
	}

	jsonapi.Marshal(w, tenantsAccessibleToUser)
}

func (h *handler) getTenant(w http.ResponseWriter, r *http.Request, companyID, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	tenantInternal, err := h.storage.GetTenantInternal(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}
	userRegions := []region.DataRegion{tenantInternal.PrimaryUserRegion}
	for region := range tenantInternal.RemoteUserRegionDBConfigs {
		userRegions = append(userRegions, region)
	}

	isMember := true
	isTenantAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		isMember = false
	}

	jsonapi.Marshal(w, SelectedTenantInfo{
		Tenant:          *tenant,
		IsConsoleTenant: tenantID == h.consoleTenant.TenantID,
		IsAdmin:         isTenantAdmin,
		IsMember:        isMember,
		UserRegions:     userRegions,
	})
}

type getTenantPlexConfigResponse struct {
	TenantConfig tenantplex.TenantConfig `json:"tenant_config"`
}

func (h *handler) getTenantPlexConfig(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		} else {
			jsonapi.MarshalError(ctx, w, err)
		}
		return
	}

	if err := tp.PlexConfig.UpdateUISettings(ctx); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, getTenantPlexConfigResponse{
		TenantConfig: tp.PlexConfig,
	})
}

// SaveTenantPlexConfigRequest is the HTTP request format for saving tenant plex config
// public for testing reasons
type SaveTenantPlexConfigRequest struct {
	TenantConfig tenantplex.TenantConfig `json:"tenant_config"`
}

// SaveTenantPlexConfigResponse is the HTTP response format for saving tenant plex config
// public for testing reasons
type SaveTenantPlexConfigResponse struct {
	TenantConfig tenantplex.TenantConfig `json:"tenant_config"`
}

func (h *handler) saveTenantPlexConfig(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req SaveTenantPlexConfigRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// load company for ACL checks
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := h.ensureEmployeeAccessToTenant(r, tenant); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if err := req.TenantConfig.UpdateSaveSettings(ctx, tenantID, tp.PlexConfig); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusServiceUnavailable))
		return
	}

	// grab any new SAML metadata
	// TODO: don't love doing this inline but it's rare for now...should move to worker
	for _, app := range req.TenantConfig.PlexMap.Apps {
		if app.SAMLIDP == nil {
			continue
		}

		for i, tsp := range app.SAMLIDP.TrustedServiceProviders {
			if tsp.EntityID != "" && len(tsp.SPSSODescriptors) == 0 {
				uclog.Debugf(ctx, "fetching SAML metadata for %v", tsp.EntityID)
				sp, err := loadSPMetadata(tsp.EntityID)
				if err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
				app.SAMLIDP.TrustedServiceProviders[i] = *sp
			}
		}
	}

	uclog.Debugf(ctx, "updating tenant Plex config for %v", tenantID)
	tp.PlexConfig = req.TenantConfig
	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	ap, err := tp.PlexConfig.PlexMap.GetActiveProvider()
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if ap.CanSyncUsers() {
		uclog.Debugf(ctx, "triggering tenant Plex config sync for %v", tenantID)
		if err := h.workerClient.Send(ctx, worker.CreateSyncAllUsersMessage(tenantID)); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	// plex's tenantconfig cache will timeout and update soon

	if err := tp.PlexConfig.UpdateUISettings(ctx); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, SaveTenantPlexConfigResponse{tp.PlexConfig})
}

// TODO loadSPMetadata should live somewhere else?
func loadSPMetadata(entityID string) (*samlconfig.EntityDescriptor, error) {
	req, err := http.NewRequest(http.MethodGet, entityID, nil)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ucerr.Errorf("unexpected status code: %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var sp *samlconfig.EntityDescriptor
	if err := xrv.Validate(bytes.NewBuffer(data)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := xml.Unmarshal(data, &sp); err != nil {
		if err.Error() == "expected element type <EntityDescriptor> but have <EntitiesDescriptor>" {
			entities := &samlconfig.EntitiesDescriptor{}
			if err := xml.Unmarshal(data, &entities); err != nil {
				return nil, ucerr.Wrap(err)
			}

			for _, e := range entities.EntityDescriptors {
				if len(e.SPSSODescriptors) > 0 {
					return &e, nil
				}
			}

			// there were no SPSSODescriptors in the response
			return nil, ucerr.New("metadata contained no service provider metadata")
		}

		return nil, ucerr.Wrap(err)
	}

	return sp, nil
}

func (h handler) enableSAMLIDP(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	aid := r.URL.Query().Get("app_id")
	appID, err := uuid.FromString(aid)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	// load company for ACL checks
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "only tenant admins can enable SAML IDP"), jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	var samlIDP *tenantplex.SAMLIDP
	for i, app := range tp.PlexConfig.PlexMap.Apps {
		if app.ID != appID {
			continue
		}

		if app.SAMLIDP != nil {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "SAML IDP already enabled for %v", app.Name), jsonapi.Code(http.StatusBadRequest))
			return
		}

		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		var max = big.NewInt(0).Exp(big.NewInt(2), big.NewInt(130), nil)
		sn, err := rand.Int(rand.Reader, max)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		crt := &x509.Certificate{
			SerialNumber: sn,
		}
		certBytes, err := x509.CreateCertificate(rand.Reader, crt, crt, &pk.PublicKey, pk)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		cpb := pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		}
		var certPEM bytes.Buffer
		if err := pem.Encode(&certPEM, &cpb); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		pkpb := pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(pk),
		}
		var keyPEM bytes.Buffer
		if err := pem.Encode(&keyPEM, &pkpb); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		keyPEMSecret, err := secret.NewString(ctx, universe.ServiceName(), samlconfig.KeySecretName(ctx, tenantID, appID), keyPEM.String())
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		// use local var for easy reading
		app.SAMLIDP = &tenantplex.SAMLIDP{
			Certificate: certPEM.String(),
			PrivateKey:  *keyPEMSecret,
			MetadataURL: fmt.Sprintf("%s/saml/metadata/%v", tenant.TenantURL, app.ClientID),
			SSOURL:      fmt.Sprintf("%s/saml/sso/%v", tenant.TenantURL, app.ClientID),
		}

		// inject our own redirect URL for now
		app.AllowedRedirectURIs = append(app.AllowedRedirectURIs, fmt.Sprintf("%s/saml/callback/%v", tenant.TenantURL, app.ClientID))

		// save outside the loop
		tp.PlexConfig.PlexMap.Apps[i] = app
		samlIDP = app.SAMLIDP
	}

	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, samlIDP)
}

// CreateTenantRequest is public just for easy test access today
type CreateTenantRequest struct {
	Tenant companyconfig.Tenant `json:"tenant"`
}

func (h handler) createTenant(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) {
	ctx := r.Context()
	var req CreateTenantRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Tenant.CompanyID.IsNil() {
		req.Tenant.CompanyID = companyID
	} else if req.Tenant.CompanyID != companyID {
		jsonapi.MarshalError(ctx, w, ucerr.New("cannot create tenant for another company"))
		return
	}

	company, err := h.storage.GetCompany(ctx, companyID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// TODO: allow specifying this by API; for now, we always autogenerate it for safety.
	if req.Tenant.TenantURL != "" {
		jsonapi.MarshalError(ctx, w, ucerr.New("must not specify 'tenant_url' when creating tenant"))
		return
	}

	tenantURL, err := tenantProvisioning.GenerateTenantURL(company.Name, req.Tenant.Name, h.tenantsProtocol, h.tenantsSubDomain)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	req.Tenant.TenantURL = tenantURL

	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	msg := worker.NewCreateTenantMessage(req.Tenant, userID)
	if err := h.workerClient.Send(ctx, msg); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// log the creation to audit log
	auditlog.Post(ctx, auditlog.NewEntry(userID.String(), auditlog.TenantCreated,
		auditlog.Payload{"ID": req.Tenant.ID, "Name": req.Tenant.Name}))

	// we return the whole tenant here (even though only ID would suffice)
	// to make the client code easier since POST /tenants returns the same
	// as GET /tenants/:id (that we are polling after this)
	jsonapi.Marshal(w, SelectedTenantInfo{
		Tenant:          req.Tenant,
		IsConsoleTenant: req.Tenant.ID == h.consoleTenant.TenantID,
		IsAdmin:         true,
		IsMember:        true,
	})
}

// UpdateTenantRequest is public for testing
type UpdateTenantRequest struct {
	Tenant companyconfig.Tenant `json:"tenant"`
}

// Validate implements Validatable
// NB: this function is explicitly defined as a no-op because we need this workaround
// to successfully save/update db proxy passwords in `updateTenant`. See #4557 for a
// more detailed explanation.
func (UpdateTenantRequest) Validate() error {
	return nil
}

func (h *handler) updateTenant(w http.ResponseWriter, r *http.Request, companyID uuid.UUID, tenantID uuid.UUID) {
	ctx := r.Context()

	var req UpdateTenantRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Tenant.ID != tenantID {
		jsonapi.MarshalError(ctx, w,
			ucerr.Errorf("tenant id in path (%v) must match tenant id in body (%v)", tenantID, req.Tenant.ID),
			jsonapi.Code(http.StatusBadRequest))
		return
	}

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if tenant.Name == req.Tenant.Name {
		jsonapi.MarshalError(ctx, w,
			ucerr.New("only tenant name can be changed, and request matches existing name"),
			jsonapi.Code(http.StatusBadRequest))
		return
	}

	tenant.Name = req.Tenant.Name

	if err := h.storage.SaveTenant(ctx, tenant); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, SelectedTenantInfo{
		Tenant:          *tenant,
		IsConsoleTenant: tenantID == h.consoleTenant.TenantID,
		IsAdmin:         true,
		IsMember:        true,
	})
}

func (h *handler) deleteTenant(w http.ResponseWriter, r *http.Request, companyID uuid.UUID, tenantID uuid.UUID) {
	ctx := r.Context()

	// Deprovision the tenant
	if err := h.deprovisionTenant(ctx, tenantID); err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listTenantUserEvents(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// Create a client specifically for this tenant.
	userEventClient, err := h.newUserEventClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	var resp *userevent.ListEventsResponse
	if userAliasStr := r.URL.Query().Get("user_alias"); userAliasStr != "" {
		userAlias, err := uuid.FromString(userAliasStr)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
		resp, err = userEventClient.ListEventsForUserAlias(ctx, userAlias.String(), pager.GetOptions()...)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	} else {
		resp, err = userEventClient.ListEvents(ctx, pager.GetOptions()...)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	jsonapi.Marshal(w, resp.Data)
}

func (h *handler) listObjectTypes(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	var options []authz.Option

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options = append(options, authz.Pagination(pager.GetOptions()...))

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	objectTypes, err := authZClient.ListObjectTypesPaginated(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, objectTypes)
}

func (h *handler) getObjectType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, objectUUID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	objectTypes, err := authZClient.GetObjectType(ctx, objectUUID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, objectTypes)
}

func (h *handler) listEdgeTypes(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	var organizationID uuid.UUID
	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		organizationID, err = uuid.FromString(orgID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
	}

	var options []authz.Option
	if organizationID != uuid.Nil {
		options = append(options, authz.OrganizationID(organizationID))
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options = append(options, authz.Pagination(pager.GetOptions()...))

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeTypes, err := authZClient.ListEdgeTypesPaginated(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeTypes)
}

func (h *handler) getEdgeType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, edgeTypeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeType, err := authZClient.GetEdgeType(ctx, edgeTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeType)
}

// CreateObjectTypeRequest is the request body for creating a new object type.
type CreateObjectTypeRequest struct {
	EdgeType struct {
		ID       uuid.UUID `json:"id"`
		TypeName string    `json:"type_name"`
	} `json:"object_type"`
}

func (h *handler) createObjectType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateObjectTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeTypes, err := authZClient.CreateObjectType(ctx, req.EdgeType.ID, req.EdgeType.TypeName)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeTypes)
}

func (h *handler) deleteObjectType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, objectTypeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	err = authZClient.DeleteObjectType(ctx, objectTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateEdgeTypeRequest is the request body for creating a new edge type.
type CreateEdgeTypeRequest struct {
	EdgeType struct {
		ID                 uuid.UUID        `json:"id"`
		SourceObjectTypeID uuid.UUID        `json:"source_object_type_id"`
		TargetObjectTypeID uuid.UUID        `json:"target_object_type_id"`
		TypeName           string           `json:"type_name"`
		Attributes         authz.Attributes `json:"attributes"`
		OrganizationID     uuid.UUID        `json:"organization_id"`
	} `json:"edge_type"`
}

func (h *handler) createEdgeType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateEdgeTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeTypes, err := authZClient.CreateEdgeType(ctx, req.EdgeType.ID, req.EdgeType.SourceObjectTypeID, req.EdgeType.TargetObjectTypeID, req.EdgeType.TypeName, req.EdgeType.Attributes, authz.OrganizationID(req.EdgeType.OrganizationID))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeTypes)
}

func (h *handler) updateEdgeType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, edgeTypeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateEdgeTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeTypes, err := authZClient.UpdateEdgeType(ctx, req.EdgeType.ID, req.EdgeType.SourceObjectTypeID, req.EdgeType.TargetObjectTypeID, req.EdgeType.TypeName, req.EdgeType.Attributes)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeTypes)
}

func (h *handler) deleteEdgeType(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, edgeTypeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	err = authZClient.DeleteEdgeType(ctx, edgeTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listObjects(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)

	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var organizationID uuid.UUID
	if orgID := r.URL.Query().Get("organization_id"); orgID != "" {
		organizationID, err = uuid.FromString(orgID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
	}

	options := []authz.Option{}
	if organizationID != uuid.Nil {
		options = append(options, authz.OrganizationID(organizationID))
	}

	if r.URL.Query().Has("type_id") {
		resp, err := authZClient.ListObjectsFromQuery(ctx, r.URL.Query(), options...)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, resp)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options = append(options, authz.Pagination(pager.GetOptions()...))
	resp, err := authZClient.ListObjects(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	jsonapi.Marshal(w, resp)

}

func (h *handler) getObject(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, objectID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := authZClient.GetObject(ctx, objectID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

// CreateObjectRequest is the request body for creating an object.
type CreateObjectRequest struct {
	Object struct {
		ID             uuid.UUID `json:"id"`
		TypeID         uuid.UUID `json:"type_id"`
		Alias          string    `json:"alias"`
		OrganizationID uuid.UUID `json:"organization_id"`
	} `json:"object"`
}

func (h *handler) createObject(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateObjectRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	obj, err := authZClient.CreateObject(ctx, req.Object.ID, req.Object.TypeID, req.Object.Alias, authz.OrganizationID(req.Object.OrganizationID))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, obj)
}

func (h *handler) deleteObject(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, objectID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	err = authZClient.DeleteObject(ctx, objectID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listEdges(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
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

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if r.URL.Query().Has("object_id") {
		objectID, err := uuid.FromString(r.URL.Query().Get("object_id"))
		if err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.Errorf("invalid `object_id` specified: %v", err), jsonapi.Code(http.StatusBadRequest))
			return
		}

		edges, err := authZClient.ListEdgesOnObject(ctx, objectID, authz.Pagination(pager.GetOptions()...))

		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, edges)
	} else {
		edges, err := authZClient.ListEdges(ctx, authz.Pagination(pager.GetOptions()...))

		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, edges)
	}
}

func (h *handler) getEdge(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, edgeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := authZClient.GetEdge(ctx, edgeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

// CreateEdgeRequest is the request body for creating an edge.
type CreateEdgeRequest struct {
	Edge struct {
		ID             uuid.UUID `json:"id"`
		SourceObjectID uuid.UUID `json:"source_object_id"`
		TargetObjectID uuid.UUID `json:"target_object_id"`
		EdgeTypeID     uuid.UUID `json:"edge_type_id"`
	} `json:"edge"`
}

func (h *handler) createEdge(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateEdgeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	edgeTypes, err := authZClient.CreateEdge(ctx, req.Edge.ID, req.Edge.SourceObjectID, req.Edge.TargetObjectID, req.Edge.EdgeTypeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, edgeTypes)
}

func (h *handler) deleteEdge(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, edgeID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	err = authZClient.DeleteEdge(ctx, edgeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) checkAttribute(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	sourceObjectID, err := uuid.FromString(r.URL.Query().Get("source_object_id"))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}
	targetObjectID, err := uuid.FromString(r.URL.Query().Get("target_object_id"))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}
	attributeName := r.URL.Query().Get("attribute")
	if len(attributeName) == 0 {
		jsonapi.MarshalError(ctx, w, ucerr.New("missing 'attribute' query parameter"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	resp, err := authZClient.CheckAttribute(ctx, sourceObjectID, targetObjectID, attributeName)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) listOrganizations(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := []authz.Option{}
	options = append(options, authz.Pagination(pager.GetOptions()...))

	organizations, err := authZClient.ListOrganizationsPaginated(ctx, options...)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, organizations)
}

func (h *handler) getOrganization(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, orgID uuid.UUID) {
	ctx := r.Context()
	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	organization, err := authZClient.GetOrganization(ctx, orgID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, organization)
}

// CreateOrganizationRequest is used for creating an organization
type CreateOrganizationRequest struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Region string    `json:"region"`
}

func (h *handler) createOrganization(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req CreateOrganizationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var reg region.DataRegion
	if !universe.Current().IsProdOrStaging() {
		reg = region.DataRegion("")
	} else {
		reg = region.DataRegion(req.Region)
	}

	organization, err := authZClient.CreateOrganization(ctx, req.ID, req.Name, reg)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, organization)
}

func (h *handler) updateOrganization(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, organizationID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req authz.UpdateOrganizationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	organization, err := authZClient.UpdateOrganization(ctx, organizationID, req.Name, req.Region)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, organization)
}

// NB: this only returns the "secondary" tenant URL table, not the main tenant record / primary URL
func (h *handler) listTenantURLs(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	// if this ever really needs paginated, we're in trouble :)
	urls, err := h.storage.ListTenantURLsForTenant(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// we only return non-system URLs for now
	// TODO: we should probably show the system ones somewhere too
	var nonSystemURLs []companyconfig.TenantURL
	for _, url := range urls {
		if !url.System {
			nonSystemURLs = append(nonSystemURLs, url)
		}
	}

	jsonapi.Marshal(w, nonSystemURLs)
}
