package tenantmap

import (
	"context"
	"net/url"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
)

// TenantState contains common state needed to manage a single tenant for
// IDP, AuthZ, etc. Multi-tenant services should build on this to store/manage
// runtime state.
type TenantState struct {
	ID                  uuid.UUID
	TenantName          string
	CompanyID           uuid.UUID
	CompanyName         string
	TenantURL           *url.URL
	TenantDB            *ucdb.DB
	UserRegionDbMap     map[region.DataRegion]*ucdb.DB
	PrimaryUserRegion   region.DataRegion
	clientCacheProvider cache.Provider
	CacheConfig         *cache.Config // used to create server side caches
	UseOrganizations    bool

	// Delay loaded elements that are not needed for every tenant or in every service
	logDB                *ucdb.DB
	delayLoadLock        *sync.Mutex
	CompanyConfigStorage *companyconfig.Storage
}

// NewTenantState creates a new TenantState object
func NewTenantState(tenant *companyconfig.Tenant, company *companyconfig.Company, tenantURL *url.URL, tenantDB, logDB *ucdb.DB, userRegionDbMap map[region.DataRegion]*ucdb.DB,
	primaryUserRegion region.DataRegion, companyconfig *companyconfig.Storage, useOrganizations bool, clientCacheProvider cache.Provider, cacheConfig *cache.Config) *TenantState {
	// TODO: when we move event collection out of IDP to a new service, it probably makes sense
	// to have a flag to control which DBs get connected to automatically per-tenant. For now we're just
	// connecting to both all the time, but `NewStateMap` could easily take a flag/bitfield to select.

	return &TenantState{
		ID:                   tenant.ID,
		TenantName:           tenant.Name,
		CompanyID:            company.ID,
		CompanyName:          company.Name,
		TenantURL:            tenantURL,
		TenantDB:             tenantDB,
		UserRegionDbMap:      userRegionDbMap,
		PrimaryUserRegion:    primaryUserRegion,
		clientCacheProvider:  clientCacheProvider,
		CacheConfig:          cacheConfig,
		UseOrganizations:     useOrganizations,
		logDB:                logDB,
		delayLoadLock:        &sync.Mutex{},
		CompanyConfigStorage: companyconfig,
	}
}

// Clone creates a shallow copy of the TenantState object with a new delayLoadLock
func (t *TenantState) Clone() *TenantState {
	return &TenantState{
		ID:                   t.ID,
		TenantName:           t.TenantName,
		CompanyID:            t.CompanyID,
		CompanyName:          t.CompanyName,
		TenantURL:            t.TenantURL,
		TenantDB:             t.TenantDB,
		UserRegionDbMap:      t.UserRegionDbMap,
		PrimaryUserRegion:    t.PrimaryUserRegion,
		clientCacheProvider:  t.clientCacheProvider,
		CacheConfig:          t.CacheConfig,
		UseOrganizations:     t.UseOrganizations,
		logDB:                t.logDB,
		delayLoadLock:        &sync.Mutex{},
		CompanyConfigStorage: t.CompanyConfigStorage,
	}
}

// GetLogDB will return a DB connection to log DB, lazy loading it if necessary
func (t *TenantState) GetLogDB(ctx context.Context) (*ucdb.DB, error) {
	if t.logDB == nil {
		t.delayLoadLock.Lock()
		defer t.delayLoadLock.Unlock()
		if t.logDB == nil {
			logDB, err := logdb.Connect(ctx, t.CompanyConfigStorage, t.ID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			t.logDB = logDB
		}
	}
	return t.logDB, nil
}

// GetCacheProvider will return a cache provider to be used by cross service clients
func (t *TenantState) GetCacheProvider(ctx context.Context) (cache.Provider, error) {
	if t.clientCacheProvider == nil {
		return cache.NewInMemoryClientCacheProvider(uuid.Must(uuid.NewV4()).String()), nil
	}
	return t.clientCacheProvider, nil
}

// GetTenantURL will return the tenant URL as a string
func (t *TenantState) GetTenantURL() string {
	if t.TenantURL == nil {
		return ""
	}
	return t.TenantURL.String()
}
