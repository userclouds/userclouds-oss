package storage

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logeventmetadata"
)

const (
	maxConnectionsPerDB     int = 10
	maxIdleConnectionsPerDB int = 5
)

// ErrTenantBeingProvisioned is returned if an tenant is in process of being provisioned.
var ErrTenantBeingProvisioned = ucerr.Friendlyf(nil, "Tenant in process of being provisioned")

// TenantEventMetadata contains the cached entries for event metadata
type TenantEventMetadata struct {
	CodeMap     map[uclog.EventCode]uclog.LogEventTypeInfo
	StringIDMap map[string]uclog.LogEventTypeInfo
	RowCount    int // count of rows in the metadata table for custom events
}

// TenantCache manages cache of DB connections and event metadata data
type TenantCache struct {
	dbConnections              map[uuid.UUID]*ucdb.DB
	eventMaps                  map[uuid.UUID]*TenantEventMetadata
	providers                  map[uuid.UUID]*oidc.Provider
	globalTenant               uuid.UUID
	globalCodeMap              map[uclog.EventCode]uclog.LogEventTypeInfo
	globalStringIDMap          map[string]uclog.LogEventTypeInfo
	cacheMutex                 sync.RWMutex
	defaultLogDBCfg            ucdb.Config
	globalEventMetadataStorage *logeventmetadata.Storage
	companyCfg                 *companyconfig.Storage
	v                          ucdb.Validator
}

// NewTenantStorageCache creates a cache instance
func NewTenantStorageCache(ctx context.Context, dbCfg *ucdb.Config, companyCfg *companyconfig.Storage,
	globalEventMetadataStorage *logeventmetadata.Storage, consoleTenant uuid.UUID, v ucdb.Validator) (*TenantCache, error) {
	var s TenantCache
	s.dbConnections = make(map[uuid.UUID]*ucdb.DB)
	s.eventMaps = make(map[uuid.UUID]*TenantEventMetadata)
	s.providers = make(map[uuid.UUID]*oidc.Provider)
	s.globalCodeMap = make(map[uclog.EventCode]uclog.LogEventTypeInfo)
	s.globalStringIDMap = make(map[string]uclog.LogEventTypeInfo)
	s.cacheMutex = sync.RWMutex{}
	s.defaultLogDBCfg = *dbCfg
	s.companyCfg = companyCfg
	s.globalEventMetadataStorage = globalEventMetadataStorage
	s.globalTenant = consoleTenant
	s.v = v

	// TODO get rid of this
	// Connect to the default log DB and store the connection in the cache under the zero guid
	db, err := ucdb.NewWithLimits(ctx, &s.defaultLogDBCfg, v, maxConnectionsPerDB, maxIdleConnectionsPerDB)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	s.dbConnections[uuid.Nil] = db

	err = s.initGlobalEventMap(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Get the console events map custom events to so they are ready to serve
	_, err = s.initEventMapForTenant(ctx, consoleTenant, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &s, nil
}

// GetStorageForTenant returns an object from the cache wrapping the DB connection
func (tc *TenantCache) GetStorageForTenant(ctx context.Context, tenantID uuid.UUID) (*Storage, error) {

	db, err := tc.connectToDB(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return NewStorage(db), nil
}

// GetEventMetadataStorageForTenant returns an object from the cache wrapping the DB connection
func (tc *TenantCache) GetEventMetadataStorageForTenant(ctx context.Context, tenantID uuid.UUID) (*logeventmetadata.Storage, error) {

	db, err := tc.connectToDB(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return logeventmetadata.NewStorage(db), nil
}

// GetEventMapForTenant returns a map of eventmetadata for the tenant
func (tc *TenantCache) GetEventMapForTenant(ctx context.Context, tenantID uuid.UUID) (*TenantEventMetadata, error) {

	tc.cacheMutex.RLock()
	em, ok := tc.eventMaps[tenantID]
	tc.cacheMutex.RUnlock()

	if !ok {
		var err error
		if em, err = tc.initEventMapForTenant(ctx, tenantID, false); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return em, nil
}

// GetProviderForTenant returns ano oidc provider for a  tenant
func (tc *TenantCache) GetProviderForTenant(ctx context.Context, tenantID uuid.UUID) (*oidc.Provider, error) {

	tc.cacheMutex.RLock()
	prov, ok := tc.providers[tenantID]
	tc.cacheMutex.RUnlock()

	if !ok {
		if err := tc.initializeProvider(ctx, tenantID); err != nil {
			return nil, ucerr.Wrap(err)
		}

		tc.cacheMutex.RLock()
		prov, ok = tc.providers[tenantID]
		tc.cacheMutex.RUnlock()

		if !ok {
			return nil, ucerr.New("Unexpected failure to init a provider for a tenant")
		}
	}
	return prov, nil
}

func (tc *TenantCache) readEventMetadataIntoMaps(ctx context.Context, s *logeventmetadata.Storage,
	codeMap *map[uclog.EventCode]uclog.LogEventTypeInfo, stringIDMap *map[string]uclog.LogEventTypeInfo) (int, error) {

	// Get the event metadata from the DB
	var allMetrics []logeventmetadata.MetricMetadata

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		if err != nil {
			return 0, ucerr.Wrap(err)
		}

		allMetrics = append(allMetrics, pageMetrics...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	rowCount, err := s.GetEventMetadataCount(ctx)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	for _, m := range allMetrics {
		le := uclog.LogEventTypeInfo{
			Name:     m.StringID,
			Code:     m.Code,
			Service:  m.Service,
			URL:      m.ReferenceURL,
			Ignore:   false,
			Category: m.Category,
		}
		(*codeMap)[m.Code] = le
		(*stringIDMap)[m.StringID] = le

	}
	return rowCount, nil
}

func (tc *TenantCache) initGlobalEventMap(ctx context.Context) error {
	if _, err := tc.readEventMetadataIntoMaps(ctx, tc.globalEventMetadataStorage, &tc.globalCodeMap, &tc.globalStringIDMap); err != nil {
		uclog.Warningf(ctx, "Failed to read global event metadata with %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}

func (tc *TenantCache) initEventMapForTenant(ctx context.Context, tenantID uuid.UUID, forceRefresh bool) (*TenantEventMetadata, error) {

	s, err := tc.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		uclog.Warningf(ctx, "Failed to connect to log DB to read event metada for tenant %v with %v", tenantID, err)
		return nil, ucerr.Wrap(err)
	}

	// Create a cache entry
	var tm = TenantEventMetadata{CodeMap: make(map[uclog.EventCode]uclog.LogEventTypeInfo), StringIDMap: make(map[string]uclog.LogEventTypeInfo),
		RowCount: 0}

	tm.RowCount, err = tc.readEventMetadataIntoMaps(ctx, s, &tm.CodeMap, &tm.StringIDMap)
	if err != nil {
		uclog.Warningf(ctx, "Failed to read event metadat tenant %v with %v", tenantID, err)
		return nil, ucerr.Wrap(err)
	}

	// Add global events to the per tenant entry TODO - don't include this in every map
	for c, m := range tc.globalCodeMap {
		tm.CodeMap[c] = m
		tm.StringIDMap[m.Name] = m
	}

	// Write the entry into the cache
	tc.cacheMutex.Lock()
	em, ok := tc.eventMaps[tenantID]
	if !ok || forceRefresh {
		tc.eventMaps[tenantID] = &tm
		em = &tm
	}
	tc.cacheMutex.Unlock()

	return em, nil
}

// RefreshEventMapForTenant returns a map of eventmetadata for the tenant
func (tc *TenantCache) RefreshEventMapForTenant(ctx context.Context, tenantID uuid.UUID) (*TenantEventMetadata, error) {

	tm, err := tc.GetEventMapForTenant(ctx, tenantID)
	if err != nil {
		return tm, ucerr.Wrap(err)
	}

	s, err := tc.GetEventMetadataStorageForTenant(ctx, tenantID)
	if err != nil {
		uclog.Warningf(ctx, "Failed to connect to log DB to read event metada for tenant %v with %v", tenantID, err)
		return tm, ucerr.Wrap(err)
	}

	currCount, err := s.GetEventMetadataCount(ctx)
	if err != nil {
		return tm, ucerr.Wrap(err)
	}

	// If we already have the latest version return (note this doesn't detect in place updates or changes to the global event metadata map)
	if currCount == tm.RowCount {
		return tm, nil
	}

	// TODO - add an accessor to only fetch event metadata rows with higher version than current map
	if tm, err = tc.initEventMapForTenant(ctx, tenantID, true); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return tm, nil
}

func (tc *TenantCache) connectToDB(ctx context.Context, tenantID uuid.UUID) (*ucdb.DB, error) {
	tc.cacheMutex.RLock()
	db, ok := tc.dbConnections[tenantID]
	tc.cacheMutex.RUnlock()

	// short circuit if we already have a connection
	if ok {
		return db, nil
	}

	// Get connection info from the companyconfig
	// TODO - this assumes that storage.companyconfig is thread safe

	tenantCfg, err := tc.companyCfg.GetTenantInternal(ctx, tenantID)
	if err != nil {
		// If there is no entry in the TenantInternal table, we check if tenant is being provisioned
		if errors.Is(err, sql.ErrNoRows) {
			tenant, err := tc.companyCfg.GetTenant(ctx, tenantID)
			if err == nil && tenant.State == companyconfig.TenantStateCreating {
				uclog.Verbosef(ctx, "Tenant %v not found in TenantInternal table, but exists in Tenants table (created at %v)", tenantID, tenant.Created)
				return nil, ErrTenantBeingProvisioned
			}
			uclog.Errorf(ctx, "Tenant %v not found in TenantInternal table, but exists in Tenants table", tenantID)
		}
		uclog.Errorf(ctx, "Failed call to GetTenantInternal - %v", err)
		return nil, ucerr.Wrap(err)
	}

	dbCfg := tenantCfg.LogConfig.LogDB

	db, err = ucdb.NewWithLimits(ctx, &dbCfg, tc.v, maxConnectionsPerDB, maxIdleConnectionsPerDB)
	if err != nil {
		// This means there was a either a provisioning error which resulted in wrong entry in TenantInternal table
		// or there is an intermittent issue connecting to the DB containing logging table. In either case we fail the call
		// and let the client decide if they want to resend data
		uclog.Warningf(ctx, "Failed to connect to per tenant status DB with config %v - %v", dbCfg, err)
		return nil, ucerr.Wrap(err)
	}

	tc.cacheMutex.Lock()
	currDB, ok := tc.dbConnections[tenantID]
	if !ok {
		// if no race, add the connection to the cache
		tc.dbConnections[tenantID] = db
	}
	tc.cacheMutex.Unlock()

	// this is effectively an else, but want to do it outside the lock
	if ok {
		if err := db.Close(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to close a duplicate log DB connection  %v", err)
		}
		return currDB, nil
	}

	return db, nil
}

func (tc *TenantCache) initializeProvider(ctx context.Context, tenantID uuid.UUID) error {
	tc.cacheMutex.RLock()
	_, ok := tc.providers[tenantID]
	tc.cacheMutex.RUnlock()

	// short circuit if we already have a provider
	if ok {
		return nil
	}

	// Get connection info from the companyconfig
	// TODO - this assumes that storage.companyconfig is thread safe
	tenant, err := tc.companyCfg.GetTenant(ctx, tenantID)
	if err != nil {
		uclog.Errorf(ctx, "Failed call to GetTenant - %v", err)
		return ucerr.Wrap(err)
	}
	prov, err := oidc.NewProvider(context.Background(), tenant.TenantURL)
	if err != nil {
		uclog.Errorf(ctx, "Failed call to oidc.NewProvider - %v", err)
		return ucerr.Wrap(err)
	}

	tc.cacheMutex.Lock()
	_, ok = tc.providers[tenantID]
	if !ok {
		tc.providers[tenantID] = prov
	}
	tc.cacheMutex.Unlock()

	return nil
}

// AddMetricsTypesForTenant adds newly defined metrics to the event metatadata cache
func (tc *TenantCache) AddMetricsTypesForTenant(ctx context.Context, tenantID uuid.UUID, m *[]logeventmetadata.MetricMetadata) {
	tc.cacheMutex.Lock()
	tm, ok := tc.eventMaps[tenantID]
	// If the event metadata is not yet cached for this tenant, we will pick up on new event types when it is read from the DB
	if ok {
		// We can't write to the map since it maybe being read by other threads so instead create a copy and then swap it
		c := tm.copyContents()
		for _, v := range *m {
			le := uclog.LogEventTypeInfo{
				Name:     v.StringID,
				Code:     v.Code,
				Service:  v.Service,
				URL:      v.ReferenceURL,
				Ignore:   false,
				Category: v.Category,
			}
			c.CodeMap[v.Code] = le
			c.StringIDMap[v.StringID] = le
		}
		tc.eventMaps[tenantID] = c
	}
	tc.cacheMutex.Unlock()
}

func (tem *TenantEventMetadata) copyContents() *TenantEventMetadata {
	var c TenantEventMetadata

	c.CodeMap = make(map[uclog.EventCode]uclog.LogEventTypeInfo, len(tem.CodeMap))
	maps.Copy(c.CodeMap, tem.CodeMap)
	c.StringIDMap = make(map[string]uclog.LogEventTypeInfo, len(tem.StringIDMap))
	maps.Copy(c.StringIDMap, tem.StringIDMap)
	c.RowCount = tem.RowCount

	return &c
}
