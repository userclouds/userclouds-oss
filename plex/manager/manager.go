package manager

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/storage"
)

// Manager implements the "business logic" of companyconfig
type Manager struct {
	storage *storage.Storage

	// NB: we only save this when we do the connect, so that we can clean it up
	tenantDB *ucdb.DB
	tenantID uuid.UUID
}

// NewFromDB returns a new Manager object constructed from a ucdb.DB object
func NewFromDB(db *ucdb.DB, cc *cache.Config) *Manager {
	return &Manager{storage: storage.New(context.Background(), db, cc)}
}

// NewFromCompanyConfig is an (expensive) convenience method to create a Manager
// TODO this is currently used in provisioning and tests but we should get rid of this method so people don't use it elsewhere
func NewFromCompanyConfig(ctx context.Context, ccs *companyconfig.Storage, tenantID uuid.UUID, cacheCfg *cache.Config) (*Manager, error) {
	tendb, _, _, err := tenantdb.Connect(ctx, ccs, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	m := NewFromDB(tendb, cacheCfg)
	m.tenantDB = tendb
	m.tenantID = tenantID
	return m, nil
}

// GetTenantPlex is a temporary method to get the TenantPlex object as we migrate
// from companyconfig -> tenantDB, and from monolithic object -> tables
func (m *Manager) GetTenantPlex(ctx context.Context, tenantID uuid.UUID) (*tenantplex.TenantPlex, error) {
	tp, err := m.storage.GetTenantPlex(ctx, tenantID)
	return tp, ucerr.Wrap(err) // wrap is no-on on nil
}

// SaveTenantPlex is a temporary method to save the TenantPlex object as we migrate
func (m *Manager) SaveTenantPlex(ctx context.Context, tenantPlex *tenantplex.TenantPlex) error {
	return ucerr.Wrap(m.storage.SaveTenantPlex(ctx, tenantPlex))
}

// Close cleans up the tenant DB handle we opened (if needed)
// TODO (sgarrity 9/23): I think with better factoring over time, we can get rid of
// NewFromCompanyConfig and thus get rid of Close()
func (m *Manager) Close(ctx context.Context) {
	if m.tenantDB != nil {
		if err := m.tenantDB.Close(ctx); err != nil {
			uclog.Errorf(ctx, "failed to close tenantDB (%v) connection we opened: %v", m.tenantID, err)
		}
	} else {
		// probably a logic error
		uclog.Warningf(ctx, "Close() called on Manager that was not created with NewFromCompanyConfig")
	}
}
