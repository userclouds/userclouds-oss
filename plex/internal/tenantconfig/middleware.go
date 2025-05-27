package tenantconfig

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/manager"
)

type contextKey int

const (
	ctxTenantConfig contextKey = 1
)

var tenantConfigCacheDuration = time.Second * 15

// TESTONLYSetTenantConfigCacheDuration exists only to allow faster testing
// TODO: is there a better way to do this?
func TESTONLYSetTenantConfigCacheDuration(d time.Duration) {
	tenantConfigCacheDuration = d
}

// TESTONLYSetTenantConfig exists to allow testing without middleware
// Note that using context.Background here makes testing slightly more fragile
// but also makes this safer since it's impossible to use in request path :)
func TESTONLYSetTenantConfig(tc *tenantplex.TenantConfig) context.Context {
	return setTenantConfig(context.Background(), tc)
}

// Cache is an in-memory cache of TenantConfigs loaded from storage.
// This isn't required for correctness but avoids every handler having to re-read from CompanyConfig
// on every invocation. Because actively invalidating it from the console gets harder as
// plex scales out, we're currently just timing out reads from the cache.
type Cache struct {
	companyConfigStorage *companyconfig.Storage
	// configs is a map of tenant ID to tenant config state.
	configs      map[uuid.UUID]cacheObject
	configsMutex sync.RWMutex
}

type cacheObject struct {
	TenantConfig *tenantplex.TenantConfig
	Expires      time.Time
}

// NewCache creates a new tenant config cache
func NewCache(companyConfigStorage *companyconfig.Storage) *Cache {
	return &Cache{
		companyConfigStorage: companyConfigStorage,
		configs:              map[uuid.UUID]cacheObject{},
	}
}

// InvalidateTenantConfig invalidates a cache entry
func (c *Cache) InvalidateTenantConfig(tenantID uuid.UUID) error {
	c.configsMutex.Lock()
	delete(c.configs, tenantID)
	c.configsMutex.Unlock()
	return nil
}

// GetTenantConfig returns a tenant plex config for a tenant ID (either from the cache or the DB),
// and caches the result.
func (c *Cache) GetTenantConfig(ctx context.Context, tenantID uuid.UUID) (*tenantplex.TenantConfig, error) {
	c.configsMutex.RLock()
	tc, ok := c.configs[tenantID]
	if ok {
		// check if we need to invalidate this
		if time.Now().UTC().Before(tc.Expires) {
			c.configsMutex.RUnlock()
			return tc.TenantConfig, nil
		}
	}
	c.configsMutex.RUnlock()

	// NB: this whole middleware does depend on multitenant.Middleware
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	c.configsMutex.Lock()
	c.configs[tenantID] = cacheObject{
		TenantConfig: &tp.PlexConfig,
		Expires:      time.Now().UTC().Add(tenantConfigCacheDuration),
	}
	c.configsMutex.Unlock()

	return &tp.PlexConfig, nil
}

// Middleware is used to resolve the tenant's PlexConfig and relies on multitenant.Middleware to resolve the tenant initially.
func Middleware(cache *Cache) middleware.Middleware {
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tenantState := multitenant.MustGetTenantState(ctx)
			tc, err := cache.GetTenantConfig(ctx, tenantState.ID)
			if err != nil {
				uchttp.Error(ctx, w, ucerr.Errorf("unable to get tenant plex config for tenant ID '%s': %v", tenantState.ID, err), http.StatusInternalServerError)
				return
			}
			ctx = setTenantConfig(ctx, tc)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}))
}

// MustGet extracts the tenant config from the context, or panics
func MustGet(ctx context.Context) tenantplex.TenantConfig {
	tc, ok := ctx.Value(ctxTenantConfig).(*tenantplex.TenantConfig)
	if !ok {
		panic("couldn't get tenant config from context, missing middleware?")
	}
	return *tc
}

func setTenantConfig(ctx context.Context, tc *tenantplex.TenantConfig) context.Context {
	return context.WithValue(ctx, ctxTenantConfig, tc)
}

// MustGetPlexMap extracts the plexmap from the context, or panics
func MustGetPlexMap(ctx context.Context) tenantplex.PlexMap {
	tc, ok := ctx.Value(ctxTenantConfig).(*tenantplex.TenantConfig)
	if !ok {
		panic("couldn't get plexmap from context, missing middleware?")
	}
	return tc.PlexMap
}

// MustGetStorage extracts the Plex storage from the context, or panics
func MustGetStorage(ctx context.Context) *storage.Storage {
	tenant := multitenant.MustGetTenantState(ctx)
	return storage.New(ctx, tenant.TenantDB, tenant.CacheConfig)
}

// MustGetTenantURLString extracts the TenantURL from the context, or panics
func MustGetTenantURLString(ctx context.Context) string {
	tenant := multitenant.MustGetTenantState(ctx)
	return tenant.TenantURL.String()
}

// MustGetTenantURL extracts the TenantURL from the context, or panics
func MustGetTenantURL(ctx context.Context) *url.URL {
	tenant := multitenant.MustGetTenantState(ctx)

	// make a copy of this so we aren't handing out mutable pointers to the context var
	u := *tenant.TenantURL
	return &u
}

// MustGetAuditLogStorage get a audit log storage, or panics
func MustGetAuditLogStorage(ctx context.Context) *auditlog.Storage {
	tenant := multitenant.MustGetTenantState(ctx)
	return auditlog.NewStorage(tenant.TenantDB)
}
