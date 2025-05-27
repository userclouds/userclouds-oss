package tenantmap

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantcache"
	"userclouds.com/internal/tenantdb"
)

const (
	initJitter      = 1000 // random delay up to 1 second
	connectJitter   = 3000 // random delay up to 3 second
	refreshInterval = 1 * time.Minute
)

// ErrInvalidTenantName exists to differentiate this specific "error", which is likely
// externally caused (by direct IP requests rather than legit tenant-URL requests), and
// should be logged as a warning rather than an error
var ErrInvalidTenantName = ucerr.NewWarning("invalid tenant name")

// StateMap contains the per-tenant state for all known tenants at runtime.
type StateMap struct {
	companyConfigStorage *companyconfig.Storage

	// tenantsByHost is a map of hostname to tenant state. Incoming requests are resolved
	// to specific tenantsByHost based on the FQDN in the Host header of the request.
	tenantsByHost map[string]*TenantState

	// tenantsByID is a map of tenant ID to tenant state. This is used to look up tenants
	// primarily in worker right now
	tenantsByID map[uuid.UUID]*TenantState

	perTenantLocks map[uuid.UUID]*sync.Mutex

	tenantsMutex sync.RWMutex
	cacheConfig  *cache.Config

	done             chan bool
	refreshTicker    time.Ticker
	tenantsToRefresh []uuid.UUID // list of tenants to refresh

	tenantInvalidationMutex      sync.Mutex
	tenantInvalidationRegistered map[uuid.UUID]bool
}

// NewStateMap returns a new structure to manage runtime state for all IDP Tenants.
func NewStateMap(companyConfigStorage *companyconfig.Storage, clientCacheConfig *cache.Config) *StateMap {
	return &StateMap{
		companyConfigStorage:         companyConfigStorage,
		tenantsByHost:                map[string]*TenantState{},
		tenantsByID:                  map[uuid.UUID]*TenantState{},
		perTenantLocks:               map[uuid.UUID]*sync.Mutex{},
		cacheConfig:                  clientCacheConfig,
		done:                         make(chan bool),
		tenantInvalidationRegistered: make(map[uuid.UUID]bool),
	}
}

// GetHostFromTenantURL extracts just the Host (FQDN) from a tenant URL.
func GetHostFromTenantURL(tenantURL string) (string, error) {
	parsedURL, err := url.Parse(tenantURL)
	if err != nil {
		return "", ucerr.Errorf("unable to get host from tenant url '%s': %w", tenantURL, err)
	}
	return parsedURL.Host, nil
}

// getPerTenantLock is a helper method to get a lock for the particular tenant (should be called once we know the tenant is valid)
func (tm *StateMap) getPerTenantLock(tenantID uuid.UUID) *sync.Mutex {
	tm.tenantsMutex.RLock()
	lock, ok := tm.perTenantLocks[tenantID]
	tm.tenantsMutex.RUnlock()

	if ok {
		// Return existing lock if found.
		return lock
	}

	// Otherwise create a new per tenant lock and add it to the map. We need full map lock to ensure we don't create duplicate per tenant locks
	tm.tenantsMutex.Lock()
	defer tm.tenantsMutex.Unlock()

	// Check again under the full lock in case another thread has already created the lock.
	lock, ok = tm.perTenantLocks[tenantID]
	if ok {
		// Return existing lock if found.
		return lock
	}

	// Create a new lock and add it to the map.
	tm.perTenantLocks[tenantID] = &sync.Mutex{}
	return tm.perTenantLocks[tenantID]
}

// readTenantByHostFromMap is a helper method to read from the tenant map with a narrowly scoped reader lock
func (tm *StateMap) readTenantByHostFromMap(key string) *TenantState {
	tm.tenantsMutex.RLock()
	defer tm.tenantsMutex.RUnlock()
	tenantState, ok := tm.tenantsByHost[key]
	if ok {
		// Return matching tenant if found.
		return tenantState
	}
	return nil
}

// addTenantToMap is a helper method to write to the tenant map with a narrowly scoped writer lock
func (tm *StateMap) addTenantToMap(ctx context.Context, key string, tenant *TenantState) *TenantState {
	// This function should be called while holding the perTenantLock to ensure that we don't have multiple threads creating tenant state for same ID
	tm.tenantsMutex.Lock()

	// Only set the tenant if it hasn't already been set
	t := tenant
	if _, ok := tm.tenantsByHost[key]; !ok {
		tm.tenantsByHost[key] = tenant

		// we can also piggyback on the tenant map to store tenants by ID, but the host
		// check is stricter so we don't need to worry about overwriting existing values
		tm.tenantsByID[tenant.ID] = tenant
	} else {
		t = tm.tenantsByHost[key]
	}
	tm.tenantsMutex.Unlock()

	// If we will throw this tenantState away - close the DB connection
	if t != tenant {
		uclog.Errorf(ctx, "Shouldn't have a duplicate connection  for tenant %v. Did you forget to take per tenant lock?", tenant.ID)
		return t
	}

	// Register the tenant change handler to invalidate the tenant state when the tenant is updated
	if err := tm.registerTenantChangeHandler(ctx, tenant); err != nil {
		uclog.Errorf(ctx, "Failed to register tenant change handler for tenant %v: %v", tenant.ID, err)
	}

	uclog.Infof(ctx, "Adding tenant %v to map for host %s with URL %s ", tenant.ID, key, tenant.TenantURL)

	return t
}

func (tm *StateMap) registerTenantChangeHandler(ctx context.Context, tenant *TenantState) error {
	tm.tenantInvalidationMutex.Lock()
	defer tm.tenantInvalidationMutex.Unlock()
	if tm.tenantInvalidationRegistered[tenant.ID] {
		return nil
	}

	handler := func(ctx context.Context, key cache.Key, flush bool) error {
		tm.ClearTenant(ctx, tenant.ID)
		return nil
	}

	if err := tm.companyConfigStorage.RegisterTenantChangeHandler(ctx, handler, tenant.ID); err != nil {
		return ucerr.Wrap(err)
	}

	tm.tenantInvalidationRegistered[tenant.ID] = true
	return nil
}

// GetTenantStateForHostname returns a TenantServiceState for a given hostname.
// NB: hostname optionally contains port (derived from http.Request.Host), which
// is how this matching works in dev
func (tm *StateMap) GetTenantStateForHostname(ctx context.Context, hostname string) (*TenantState, error) {
	// Always treat host names as lower case.
	hostname = strings.ToLower(hostname)
	if tenantState := tm.readTenantByHostFromMap(hostname); tenantState != nil {
		// Return matching tenant if found.
		return tenantState, nil
	}

	// Lazily create new tenant state if not found.
	// TODO: don't keep looking up known-bad hostnames?
	tenant, company, tenantURL, err := tm.getTenantInformation(ctx, uuid.Nil, hostname)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Take per tenant lock to ensure we don't create a herd of connection requests which load spikes on restart or for new tenant
	perTenantLock := tm.getPerTenantLock(tenant.ID)
	perTenantLock.Lock()
	defer perTenantLock.Unlock()

	// Recheck if the connections have been created while under per tenant lock
	if tenantState := tm.readTenantByHostFromMap(hostname); tenantState != nil {
		// Return matching tenant if found.
		return tenantState, nil
	}

	// Now check if tenant state for this tenant ID exists with different hostname (reuse DB connections in that case)
	tm.tenantsMutex.RLock()
	tenantStateByID := tm.tenantsByID[tenant.ID]
	tm.tenantsMutex.RUnlock()

	// TODO: not sure I love this design but want to write eg. JWTs for the requested URL, not always
	// the primary URL. We could force everything to the primary and just allow you to change it, but that
	// seems overly restrictive right now (and also slower to implement?)
	var tenantState *TenantState
	if tenantStateByID == nil {
		tenantState, err = tm.createTenantStateObject(ctx, tenant, company, tenantURL)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	} else {
		tenantState = tenantStateByID.Clone()
		tenantURL := *tenantStateByID.TenantURL
		tenantURL.Host = hostname
		tenantState.TenantURL = &tenantURL
	}
	tenantState = tm.addTenantToMap(ctx, hostname, tenantState)
	return tenantState, nil
}

// GetTenantStateForID returns a TenantServiceState for a given tenant ID.
func (tm *StateMap) GetTenantStateForID(ctx context.Context, tenantID uuid.UUID) (*TenantState, error) {
	tm.tenantsMutex.RLock()
	tenantState, ok := tm.tenantsByID[tenantID]
	tm.tenantsMutex.RUnlock()
	if ok {
		// Return matching tenant if found.
		return tenantState, nil
	}

	// Lazily create new tenant state if not found.
	tenant, company, tenantURL, err := tm.getTenantInformation(ctx, tenantID, "")
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Take per tenant lock to ensure we don't create a herd of connection requests which load spikes on restart or for new tenant
	perTenantLock := tm.getPerTenantLock(tenant.ID)
	perTenantLock.Lock()
	defer perTenantLock.Unlock()

	// Recheck if the connections have been created while under per tenant lock
	tm.tenantsMutex.RLock()
	tenantState, ok = tm.tenantsByID[tenantID]
	tm.tenantsMutex.RUnlock()
	if ok {
		// Return matching tenant if found.
		return tenantState, nil
	}

	tenantState, err = tm.createTenantStateObject(ctx, tenant, company, tenantURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	tenantState = tm.addTenantToMap(ctx, tenantURL.Host, tenantState)
	return tenantState, nil
}

// InitializeConnections opens connections to tenants that should be connected on startup
func (tm *StateMap) InitializeConnections(ctx context.Context) ([]uuid.UUID, error) {
	// Introduce jitter to avoid thundering herd on getting tenant info from company config DB
	// normally all or all but one of the processes will hit the cache and the jitter is only for the
	// case when cache key is not yet set
	time.Sleep(time.Duration(rand.Intn(initJitter)) * time.Millisecond)

	tenants, err := tm.companyConfigStorage.GetConnectOnStartupTenants(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenantStates := make([]*TenantState, 0, len(tenants))
	for _, tenant := range tenants {
		// Space out the connection to tenant DBs avoid impacting the DB
		time.Sleep(time.Duration(rand.Intn(connectJitter)) * time.Millisecond)
		ts, err := tm.GetTenantStateForID(ctx, tenant.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		tenantStates = append(tenantStates, ts)
		tm.tenantsToRefresh = append(tm.tenantsToRefresh, tenant.ID)
	}

	// Initialize a background thread to keep the DB connections alive. This is important for tenants which use certain services
	// like AuthZ rarely so connection can be closed by the DB due to inactivity.
	tm.refreshTicker = *time.NewTicker(refreshInterval)
	go func() {
		for {
			select {
			case <-tm.done:
				return
			case <-tm.refreshTicker.C:
				for _, tenantID := range tm.tenantsToRefresh {
					if tenantState, err := tm.GetTenantStateForID(ctx, tenantID); err != nil {
						uclog.Errorf(ctx, "failed to refresh tenant %v: %v", tenantID, err)
					} else {
						if err := tenantState.TenantDB.Ping(); err != nil {
							uclog.Errorf(ctx, "failed to ping tenant %v: %v", tenantID, err)
						}
					}
				}
			}
		}
	}()

	ids := make([]uuid.UUID, 0, len(tenantStates))
	for _, ts := range tenantStates {
		ids = append(ids, ts.ID)
	}
	return ids, nil
}

// ClearTenant clears the cache for a given tenant
func (tm *StateMap) ClearTenant(ctx context.Context, tenantID uuid.UUID) {
	// Process the invalidation async to avoid blocking the invalidation thread with slow DB connect operation
	go func() {
		ctx := context.WithoutCancel(ctx)
		uclog.Debugf(ctx, "processing TenantState invalidation for tenant %v", tenantID)

		// We can safely assume that tenant id is valid as it is coming from the cache key invalidation
		// Take per tenant lock to ensure we don't have another thread creating a tenant state based on stale information while we process the invalidation
		perTenantLock := tm.getPerTenantLock(tenantID)
		perTenantLock.Lock()
		defer perTenantLock.Unlock()

		// Check if tenant ID is in the tenantsByID map. Since we always insert tenant state into the tenantsByID map (and under perTenantLock), we can safely assume that if it is not in there
		// it is not in the tenantsByHost map either
		tm.tenantsMutex.RLock()
		existingTenantState, tenantExists := tm.tenantsByID[tenantID]
		tm.tenantsMutex.RUnlock()

		// If tenant ID is not in the map we can safely return as there is no tenant state to clear
		if !tenantExists || existingTenantState == nil { // lint: ignore
			return
		}

		// If tenant ID is in the map we can continue to safely hold perTenantLock while we swap/clear the tenant state. If there are multiple invalidation fired
		// they will get serialized behind the perTenantLock lock but all the incoming requests will not get blocked
		// because they will read the value from tenantsByHost/tenantsByID map

		// Create new tenant state on basis of updated information
		newTenantState, err := tm.createTenantState(ctx, tenantID, "")
		if err != nil {
			uclog.Errorf(ctx, "failed to create tenant state for existing tenant during invalidation %v: %v", tenantID, err)
		}

		// Close the DB connections in the existing tenant map entry after a delay
		go closeDBAfterDelay(ctx, existingTenantState.logDB)
		go closeDBAfterDelay(ctx, existingTenantState.TenantDB)
		for _, db := range existingTenantState.UserRegionDbMap {
			go closeDBAfterDelay(ctx, db)
		}

		// Take tenantsMutex lock to ensure that we don't have multiple threads modifying the map at the same time and swap/clear all the entries corresponding to the tenant ID
		// with new tenant state. We are doing a swap to ensure that production requests don't hit the initial DB connection operation
		tm.tenantsMutex.Lock()
		defer tm.tenantsMutex.Unlock()

		if newTenantState != nil {
			tm.tenantsByID[tenantID] = newTenantState
		} else {
			delete(tm.tenantsByID, tenantID)
		}

		for k, v := range tm.tenantsByHost {
			if v.ID == tenantID {
				uclog.Debugf(ctx, "found TenantState (%s) to swap/clear for tenant %v", k, tenantID)

				// If successfully created new tenant state - swap the old one for it
				if newTenantState != nil {
					// Create a copy of the tenant state and update the tenant URL to the new hostname
					perHostTenantState := newTenantState.Clone()
					tenantURL := *newTenantState.TenantURL
					tenantURL.Host = k
					perHostTenantState.TenantURL = &tenantURL

					tm.tenantsByHost[k] = perHostTenantState
				} else {
					delete(tm.tenantsByHost, k)
				}
				break
			}
		}

	}()
}

func (tm *StateMap) createTenantState(ctx context.Context, tenantID uuid.UUID, hostname string) (*TenantState, error) {
	tenant, company, tenantURL, err := tm.getTenantInformation(ctx, tenantID, hostname)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if hostname != "" {
		tenantURL.Host = hostname
	}

	tenantState, err := tm.createTenantStateObject(ctx, tenant, company, tenantURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return tenantState, nil
}

func (tm *StateMap) createTenantStateObject(ctx context.Context, tenant *companyconfig.Tenant, company *companyconfig.Company, tenantURL *url.URL) (*TenantState, error) {
	tenantDB, userRegionDbMap, primaryDataRegion, err := tenantdb.Connect(ctx, tm.companyConfigStorage, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	clientCacheProvider, err := tenantcache.Connect(ctx, tm.cacheConfig, tenant.ID)
	if err != nil {
		// log the error but don't fail
		uclog.Errorf(ctx, "failed to connect to client cache: %v", err)
		clientCacheProvider = nil
		err = nil
	}

	newTenant := NewTenantState(tenant, company, tenantURL, tenantDB, nil, userRegionDbMap, primaryDataRegion, tm.companyConfigStorage, tenant.UseOrganizations, clientCacheProvider, tm.cacheConfig)

	return newTenant, nil
}

func (tm *StateMap) getTenantInformation(ctx context.Context, tenantID uuid.UUID, hostname string) (*companyconfig.Tenant, *companyconfig.Company, *url.URL, error) {
	var tenant *companyconfig.Tenant
	var err error

	if tenantID.IsNil() && hostname == "" {
		return nil, nil, nil, ucerr.Errorf("tenantID and hostname cannot both be empty")
	}

	// Fetch tenant either by ID or hostname
	if !tenantID.IsNil() {
		tenant, err = tm.companyConfigStorage.GetTenant(ctx, tenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil, nil, ucerr.Wrap(ucerr.Combine(err, ucerr.Errorf("tenant %v not found: %w", tenantID, ErrInvalidTenantName)))
			}
			return nil, nil, nil, ucerr.Wrap(err)
		}
	} else {
		tenant, err = tm.companyConfigStorage.GetTenantByHost(ctx, hostname)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil, nil, ucerr.Errorf("tenant for hostname '%s' not found: %w", hostname, ErrInvalidTenantName)
			}
			return nil, nil, nil, ucerr.Wrap(err)
		}
	}

	company, err := tm.companyConfigStorage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}
	tenantURL, err := url.Parse(tenant.TenantURL)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	return tenant, company, tenantURL, nil
}

const delayBeforeClose = 5 * time.Second

func closeDBAfterDelay(ctx context.Context, db *ucdb.DB) {
	go func(ctx context.Context, db *ucdb.DB) {
		if db == nil {
			return
		}
		time.Sleep(delayBeforeClose)
		if err := db.Close(ctx); err != nil {
			uclog.Errorf(ctx, "failed to close DB: %v", err)
		}
	}(ctx, db)
}
