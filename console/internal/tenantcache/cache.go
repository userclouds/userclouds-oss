package tenantcache

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantdb"
)

// Cache is a Cache of tenant information (specifically DB connections)
type Cache struct {
	tenantMap      map[uuid.UUID]*ucdb.DB
	tenantMapMutex sync.RWMutex
	ccs            *companyconfig.Storage
}

// NewCache creates a new tenant cache
func NewCache(ccs *companyconfig.Storage) *Cache {
	return &Cache{
		tenantMap: make(map[uuid.UUID]*ucdb.DB),
		ccs:       ccs,
	}
}

// GetTenantDB returns a DB connection for the given tenant
func (c *Cache) GetTenantDB(ctx context.Context, tenantID uuid.UUID) (*ucdb.DB, error) {
	if c == nil {
		return nil, ucerr.New("tenant cache not initialized")
	}

	c.tenantMapMutex.RLock()
	db, ok := c.tenantMap[tenantID]
	c.tenantMapMutex.RUnlock()
	if ok {
		return db, nil
	}

	c.tenantMapMutex.Lock()
	defer c.tenantMapMutex.Unlock()

	// double check that it wasn't added while we were waiting for the lock
	db, ok = c.tenantMap[tenantID]
	if ok {
		return db, nil
	}

	tenantDB, _, _, err := tenantdb.Connect(ctx, c.ccs, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	c.tenantMap[tenantID] = tenantDB
	return tenantDB, nil
}
