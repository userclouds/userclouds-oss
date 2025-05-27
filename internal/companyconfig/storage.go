package companyconfig

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	lookAsideCacheTTL               = 24 * time.Hour
	lookAsideCacheInvalidationDelay = 750 * time.Millisecond
	companyConfigCacheName          = "companyconfigCache"
)

// Storage defines the interface for storing the configuration for
// all Companies (and their Tenants) managed by UserClouds.
type Storage struct {
	db   *ucdb.DB
	cp   cache.Provider
	cm   *cache.Manager
	ttlP *companyConfigCacheTTLProvider
}

// ErrCompanyStillHasTenants is returned if a caller attempts to delete an company with living tenants.
var ErrCompanyStillHasTenants = ucerr.New("cannot delete company because it still has tenants")

var sharedCache cache.Provider
var sharedCacheOnce sync.Once

// NewStorage returns a new DB-backed companyconfig.Storage object to access
// company & tenant metadata.
func NewStorage(ctx context.Context, db *ucdb.DB, cc *cache.Config) (*Storage, error) {
	s := &Storage{
		db: db,
	}

	if err := s.initializeCache(ctx, cc); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return s, nil
}

// NewStorageFromConfig returns a new DB-backed companyconfig.Storage object
// to access company & tenant metadata using DB config.
func NewStorageFromConfig(ctx context.Context, cfg *ucdb.Config, cc *cache.Config) (*Storage, error) {
	db, err := ucdb.New(ctx, cfg, migrate.SchemaValidator(Schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	s := &Storage{
		db: db,
	}

	if err := s.initializeCache(ctx, cc); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return s, nil
}

func (s *Storage) initializeCache(ctx context.Context, cc *cache.Config) error {
	if cc != nil && cc.RedisCacheConfig != nil {
		invalidationDelay := lookAsideCacheInvalidationDelay
		if universe.Current().IsTestOrCI() {
			invalidationDelay = 1
		}

		sharedCacheOnce.Do(func() {
			var err error
			sharedCache, err = cache.InitializeInvalidatingCacheFromConfig(
				ctx,
				cc,
				companyConfigCacheName,
				"",
				cache.Layered(),
				cache.InvalidationDelay(invalidationDelay),
			)
			if err != nil {
				uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
			}
		})

		if sharedCache != nil {
			s.cp = sharedCache
			np := NewCompanyConfigCacheNameProvider()
			s.ttlP = newCompanyConfigCacheTTLProvider(lookAsideCacheTTL, lookAsideCacheTTL, lookAsideCacheTTL, lookAsideCacheTTL, lookAsideCacheTTL, lookAsideCacheTTL)
			cm := cache.NewManager(s.cp, np, s.ttlP)
			s.cm = &cm
		}
	}
	return nil
}

func (s *Storage) preDeleteCompany(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	// NOTE: this check is not transactional but we could do it in a single statement if needed.
	tenants, err := s.ListTenantsForCompany(ctx, id)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if len(tenants) > 0 {
		return ucerr.Wrap(ErrCompanyStillHasTenants)
	}
	return nil
}

// GetTenantInfo returns the TenantInfo for a given TenantID
func (s *Storage) GetTenantInfo(ctx context.Context, tenantID uuid.UUID) (*TenantInfo, error) {
	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &TenantInfo{
		TenantID:  tenant.ID,
		CompanyID: tenant.CompanyID,
		TenantURL: tenant.TenantURL,
	}, nil
}

// SetCompanyType sets the type of a company and saves it to the DB.
func (s *Storage) SetCompanyType(ctx context.Context, company *Company, companyType CompanyType) error {
	if company.Type == companyType {
		uclog.Debugf(ctx, "Company %s [%v] is already '%v'", company.Name, company.ID, company.Type)
		return nil
	}
	uclog.Infof(ctx, "Setting company type for company %s [%v] from '%v' to '%v'", company.Name, company.ID, company.Type, companyType)
	company.Type = companyType
	return ucerr.Wrap(s.SaveCompany(ctx, company))
}

// GetTenantByHost returns summary information about a tenant by its host (hostname + port)
// We lowercase everything in SQL just to be sure.
// We use host instead of full URL because when we look up tenants by request.Host, we don't
// know the scheme.
func (s *Storage) GetTenantByHost(ctx context.Context, host string) (*Tenant, error) {
	if s.cm != nil && host != "" {
		v, _, _, err := cache.GetItemFromCache[Tenant](ctx, *s.cm, s.cm.N.GetKeyNameWithString(TenantHostnameKeyID, host), false)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return v, nil
		}
	}

	// NB: the primary URL is (at this point) always validated
	const query = `SELECT id, created, updated, deleted, name, company_id, tenant_url, use_organizations, state, sync_users FROM tenants WHERE LOWER(tenant_url) LIKE CONCAT('%://', LOWER($1)) AND deleted='0001-01-01 00:00:00';`
	var tenant Tenant
	if err := s.db.GetContext(ctx, "GetTenantByHost", &tenant, query, host); err != nil {
		// if we don't find it in primary, look it up in secondary storage
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Wrap(err)
		}

		// NB: this isn't actually GetTenantURLByURL since we have a host (missing scheme)
		var t TenantURL
		const nq = `SELECT id, created, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until FROM tenants_urls WHERE LOWER(tenant_url) LIKE CONCAT('%://', LOWER($1)) AND validated=TRUE AND deleted='0001-01-01 00:00:00';`
		if err := s.db.GetContext(ctx, "GetTenantByHost", &t, nq, host); err != nil {
			return nil, ucerr.Wrap(err)
		}
		ten, err := s.GetTenant(ctx, t.TenantID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		tenant = *ten
	}

	return &tenant, nil
}

// ListTenantsByName looks up a tenant by name (case insensitive), tenant name may contain pattern matching expression (using % symbols)
// This is command line only functon so it doesn't use cache
func (s *Storage) ListTenantsByName(ctx context.Context, tenantName string) ([]Tenant, error) {
	// NB: tenant name is case insensitive for the purpose of this query
	const query = `SELECT id, created, updated, deleted, name, company_id, tenant_url, use_organizations, state, sync_users FROM tenants WHERE name ILIKE $1 AND deleted='0001-01-01 00:00:00';`
	var tenants []Tenant
	if err := s.db.SelectContext(ctx, "ListTenantsByName", &tenants, query, tenantName); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return tenants, nil
}

// GetTenantURLByURL allows lookup of TenantURL objects by URL
func (s *Storage) GetTenantURLByURL(ctx context.Context, url string) (*TenantURL, error) {
	const query = `SELECT id, created, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until FROM tenants_urls WHERE LOWER(tenant_url)=LOWER($1) AND deleted='0001-01-01 00:00:00';`
	var tu TenantURL
	if err := s.db.GetContext(ctx, "GetTenantURLByURL", &tu, query, url); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &tu, nil
}

func (s *Storage) additionalSaveKeysForTenant(tenant *Tenant) []cache.Key {
	if s.cm != nil {
		return []cache.Key{
			s.cm.N.GetKeyNameWithID(CompanyTenantsKeyID, tenant.CompanyID),
		}
	}
	return []cache.Key{}
}

// ListTenantsForCompany returns summaries of all tenants within a single company.
func (s *Storage) ListTenantsForCompany(ctx context.Context, companyID uuid.UUID) ([]Tenant, error) {
	var obj Company
	var sentinel cache.Sentinel
	var ckey cache.Key
	var err error

	if companyID.IsNil() {
		return nil, ucerr.New("companyID cannot be nil")
	}

	if s.cm != nil {
		ckey = s.cm.N.GetKeyNameWithID(CompanyTenantsKeyID, companyID)

		var tenants *[]Tenant
		if tenants, _, sentinel, _, err = cache.GetItemsArrayFromCache[Tenant](ctx, *s.cm, ckey, true); err != nil {
			return nil, ucerr.Wrap(err)
		}
		// We found collection in the cache, return it
		if tenants != nil {
			return *tenants, nil
		}

		// Clear the lock in case of an error
		defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{ckey}, obj, sentinel)
	}

	const query = `SELECT id, created, updated, deleted, name, company_id, tenant_url, use_organizations, state, sync_users FROM tenants WHERE company_id=$1 AND deleted='0001-01-01 00:00:00';`

	// TODO: validate that Company ID is actually a valid company?
	var tenants []Tenant
	if err := s.db.SelectContext(ctx, "ListTenantsForCompany", &tenants, query, companyID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if s.cm != nil {
		cache.SaveItemsToCollection(ctx, *s.cm, obj, tenants, ckey, ckey, sentinel, false)
	}
	return tenants, nil
}

func (s *Storage) preDeleteTenant(ctx context.Context, tenantID uuid.UUID, wrappedDelete bool) error {
	// on error, we log and continue since we still might be able to delete the "main"
	// Tenant record and that would be better than nothing.
	if err := s.DeleteTenantInternal(ctx, tenantID); err != nil {
		uclog.Errorf(ctx, "error deleting tenant internal: %v", err)
	}

	tus, err := s.ListTenantURLsForTenant(ctx, tenantID)
	if err != nil {
		uclog.Errorf(ctx, "error listing tenant urls: %v", err)
	}

	for _, tu := range tus {
		if err := s.DeleteTenantURL(ctx, tu.ID); err != nil {
			uclog.Errorf(ctx, "error deleting tenant url: %v", err)
		}
	}

	return nil
}

// GetValidInviteKey loads a InviteKey by Key
// NB: we filter out used/expired keys.
func (s Storage) GetValidInviteKey(ctx context.Context, key string) (*InviteKey, error) {
	const q = "SELECT id, created, updated, deleted, type, key, expires, used, company_id, role, tenant_roles, invitee_email, invitee_user_id FROM invite_keys WHERE key=$1 AND expires>NOW() AND used=FALSE AND deleted='0001-01-01 00:00:00';"

	var obj InviteKey
	if err := s.db.GetContext(ctx, "GetValidInviteKey", &obj, q, key); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// ListValidInviteKeysByInvitee loads valid & unused InviteKeys by InviteeUserID
// NB: we filter out used/expired keys.
func (s Storage) ListValidInviteKeysByInvitee(ctx context.Context, inviteeUserID uuid.UUID) ([]InviteKey, error) {
	const q = "SELECT id, created, updated, deleted, type, key, expires, used, company_id, role, tenant_roles, invitee_email, invitee_user_id FROM invite_keys WHERE invitee_user_id=$1 AND expires>NOW() AND used=FALSE AND deleted='0001-01-01 00:00:00';"

	var objs []InviteKey
	if err := s.db.SelectContext(ctx, "ListValidInviteKeysByInvitee", &objs, q, inviteeUserID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return objs, nil
}

func (s *Storage) additionalSaveKeysForTenantURL(tenantURL *TenantURL) []cache.Key {
	if s.cm != nil {
		return []cache.Key{
			s.cm.N.GetKeyNameWithID(TenantsURLsKeyID, tenantURL.TenantID),
		}
	}
	return []cache.Key{}
}

// ListTenantURLsForTenant returns all TenantURLs for a given tenant.
// If we need to paginate this, we have bigger problems for now :)
func (s Storage) ListTenantURLsForTenant(ctx context.Context, id uuid.UUID) ([]TenantURL, error) {
	var obj Tenant
	var sentinel cache.Sentinel
	var ckey cache.Key
	var err error

	if id.IsNil() {
		return nil, ucerr.New("tenantID cannot be nil")
	}

	if s.cm != nil {
		ckey = s.cm.N.GetKeyNameWithID(TenantsURLsKeyID, id)

		var tenantURLs *[]TenantURL
		if tenantURLs, _, sentinel, _, err = cache.GetItemsArrayFromCache[TenantURL](ctx, *s.cm, ckey, true); err != nil {
			return nil, ucerr.Wrap(err)
		}
		// We found collection in the cache, return it
		if tenantURLs != nil {
			return *tenantURLs, nil
		}

		// Clear the lock in case of an error
		defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{ckey}, obj, sentinel)
	}

	const q = `SELECT id, created, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until FROM tenants_urls WHERE tenant_id=$1 AND deleted='0001-01-01 00:00:00';`

	var objs []TenantURL
	if err := s.db.SelectContext(ctx, "ListTenantURLsForTenant", &objs, q, id); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if s.cm != nil {
		cache.SaveItemsToCollection(ctx, *s.cm, obj, objs, ckey, ckey, sentinel, false)
	}
	return objs, nil
}

// GetConnectOnStartupTenants returns all tenants that should be connected to on startup
func (s *Storage) GetConnectOnStartupTenants(ctx context.Context) ([]Tenant, error) {

	// We are emulating this query below with two reads that will both hit the cache so we don't have to add extra cache key for the sub collection
	// Once we have more than 1500 tenant we'll need to add the sub collection cache key
	// const q = `SELECT t.id, t.created, t.updated, t.deleted, t.name, t.company_id, t.tenant_url, t.use_organizations, t.state FROM tenants t JOIN tenants_internal ti ON t.id = ti.id WHERE ti.connect_on_startup=true AND t.deleted='0001-01-01 00:00:00'; /* lint-bypass-known-table-check */`

	// Load all tenants from DB
	tenants := []Tenant{}

	pager, err := NewTenantPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		ts, respFields, err := s.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		tenants = append(tenants, ts...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	// Load all internal tenants from DB
	tenantsInternal := []TenantInternal{}
	pager, err = NewTenantInternalPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for {
		tis, respFields, err := s.ListTenantInternalsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		tenantsInternal = append(tenantsInternal, tis...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	// Filter tenants that should be connected to on startup
	filteredTenants := []Tenant{}
	for _, ti := range tenantsInternal {
		if ti.ConnectOnStartup {
			for _, t := range tenants {
				if t.ID == ti.ID {
					filteredTenants = append(filteredTenants, t)
				}
			}
		}
	}

	return filteredTenants, nil
}

// GetSQLShimProxyForPort returns a SQLShimProxy for a given port
func (s *Storage) GetSQLShimProxyForPort(ctx context.Context, port int) (*SQLShimProxy, error) {
	pager, err := NewSQLShimProxyPaginatorFromOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		proxies, pr, err := s.ListSQLShimProxiesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, proxy := range proxies {
			if proxy.Port == port {
				return &proxy, nil
			}
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	return nil, ucerr.Errorf("no SQLShimProxy found for port %d", port)
}

// RegisterTenantChangeHandler registers a handler to be called when info about a tenant in either `tenants` or `tenants_internal` table changes
func (s *Storage) RegisterTenantChangeHandler(ctx context.Context, handler cache.InvalidationHandler, tenantID uuid.UUID) error {
	if sharedCache == nil {
		return ucerr.Errorf("sharedCache is not initialized")
	}
	if s.cm == nil {
		return ucerr.Errorf("storage class was initialized without cache config")
	}
	key := s.cm.N.GetKeyNameWithString(TenantInternalKeyID, tenantID.String())
	if err := sharedCache.RegisterInvalidationHandler(ctx, handler, key); err != nil {
		return ucerr.Wrap(err)
	}
	key = s.cm.N.GetKeyNameWithString(TenantKeyID, tenantID.String())
	if err := sharedCache.RegisterInvalidationHandler(ctx, handler, key); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
