package worker

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
)

// Message defines the message that gets passed over SQS to the worker
// TODO: think about m2m auth someday as these get more complex?
type Message struct {
	TenantID uuid.UUID `json:"tenant_id"` // the tenant that this message is for
	Task     Task      `json:"task"`      // which worker task to pass this to
	// Skip validation on SourceRegion to allow for backwards compatibility
	SourceRegion region.MachineRegion `json:"source_region" validate:"notempty"` // the region the message was sent from

	FinalizeTenantCNAME                  *FinalizeTenantCNAMEParams            `json:"finalize_tenant_cname" validate:"allownil"`                     // used for TaskFinalizeTenantCNAME
	CheckTenantCNAME                     *CheckTenantCNAMEParams               `json:"check_tenant_cname" validate:"allownil"`                        // used for TaskCheckTenantCNAME
	CreateTenant                         *CreateTenantParams                   `json:"create_tenant" validate:"allownil"`                             // used for TaskCreateTenant
	ImportAuth0Apps                      *ImportAuth0AppsParams                `json:"import_auth0_apps"  validate:"allownil"`                        // used for TaskImportAuth0Apps
	ClearCache                           *ClearCacheParams                     `json:"clear_cache" validate:"allownil"`                               // used for TaskClearCache
	LogCache                             *LogCacheParams                       `json:"log_cache" validate:"allownil"`                                 // used for TaskLogCache
	TenantDNS                            *TenantDNSTaskParams                  `json:"tenant_dns" validate:"allownil"`                                // used for TaskValidateDNS & TaskNewTenantCNAME
	DataImportParams                     *DataImportParams                     `json:"data_import_params" validate:"allownil"`                        // used for TaskDataImport
	PlexTokenDataCleanup                 *DataCleanupParams                    `json:"plex_token_data_cleanup" validate:"allownil"`                   // used for TaskPlexTokenDataCleanup
	UserStoreDataCleanup                 *DataCleanupParams                    `json:"userstore_data_cleanup" validate:"allownil"`                    // used for TaskUserStoreDataCleanup
	TenantURLProvisioningParams          *TenantURLProvisioningParams          `json:"tenant_url_provisioning_params" validate:"allownil"`            // used for TaskProvisionTenantURLs
	IngestSqlshimDatabaseSchemasParams   *IngestSqlshimDatabaseSchemasParams   `json:"ingest_sqlshim_database_schemas" validate:"allownil"`           // used for TaskIngestSqlshimDatabaseSchemas
	ProvisionTenantOpenSearchIndexParams *ProvisionTenantOpenSearchIndexParams `json:"provision_tenant_open_search_index_params" validate:"allownil"` // used for TaskProvisionTenantOpenSearchIndex
	BootstrapTenantOpenSearchIndexParams *BootstrapTenantOpenSearchIndexParams `json:"bootstrap_tenant_open_search_index_params" validate:"allownil"` // used for TaskBootstrapTenantOpenSearchIndex
	UpdateTenantOpenSearchIndexParams    *UpdateTenantOpenSearchIndexParams    `json:"update_tenant_opensearch_index" validate:"allownil"`            // used for TaskUpdateTenantOpenSearchIndex
	NoOpParams                           *NoOpTaskParams                       `json:"noop_task_params" validate:"allownil"`                          // used for TaskNoOp
	SaveTenantInternalParams             *SaveTenantInternalParams             `json:"save_tenant_internal_params" validate:"allownil"`               // used for TaskSaveTenantInternal
}

//go:generate genvalidate Message

// CacheType is the type of cache to clear (plex, authz, userstore, company_config, all)
type CacheType string

// SetSourceRegionIfNotSet sets the SourceRegion field to current if it is not already set
func (msg *Message) SetSourceRegionIfNotSet() {
	if string(msg.SourceRegion) == "" {
		msg.SourceRegion = region.Current()
	}
}

// GetTenantID returns the tenant ID for the message
func (msg *Message) GetTenantID() uuid.UUID {
	if !msg.TenantID.IsNil() {
		return msg.TenantID
	}
	if msg.CreateTenant != nil {
		return msg.CreateTenant.Tenant.ID
	}
	return uuid.Nil
}

func (msg Message) String() string {
	return fmt.Sprintf("%v for tenant %v [from %v]", msg.Task, msg.GetTenantID(), msg.SourceRegion)
}

// CacheType constants
const (
	CacheTypeAll           CacheType = "all"
	CacheTypeAuthZ         CacheType = "authz"
	CacheTypeUserStore     CacheType = "userstore"
	CacheTypePlex          CacheType = "plex"
	CacheTypeCompanyConfig CacheType = "company_config"
)

//go:generate genconstant CacheType

// LogCacheParams is the parameters for logging values in the a cache
type LogCacheParams struct {
	CacheType CacheType `json:"cache_type"`
	TenantID  uuid.UUID `json:"tenant_id"`
}

//go:generate genvalidate LogCacheParams

// CreateLogCacheMessage creates a message to clear a cache
func CreateLogCacheMessage(cacheType CacheType, tenantID uuid.UUID) Message {
	return Message{
		Task:     TaskLogCache,
		TenantID: tenantID,
		LogCache: &LogCacheParams{
			CacheType: cacheType,
			TenantID:  tenantID,
		},
	}
}

// ClearCacheParams is the parameters for clearing a cache
type ClearCacheParams struct {
	CacheType CacheType `json:"cache_type"`
	TenantID  uuid.UUID `json:"tenant_id"`
}

//go:generate genvalidate ClearCacheParams

func (ccp *ClearCacheParams) extraValidate() error {
	if isPerTenantCache := ccp.CacheType == CacheTypeAuthZ || ccp.CacheType == CacheTypeUserStore || ccp.CacheType == CacheTypePlex; !ccp.TenantID.IsNil() && !isPerTenantCache {
		return ucerr.Errorf("tenant ID Only supported for '%v' & '%v' cache types not '%v'", CacheTypeUserStore, CacheTypeAuthZ, ccp.CacheType)
	}
	return nil
}

// CreateClearCacheMessage creates a message to clear a cache
func CreateClearCacheMessage(cacheType CacheType, tenantID uuid.UUID) Message {
	return Message{
		Task:     TaskClearCache,
		TenantID: tenantID,
		ClearCache: &ClearCacheParams{
			CacheType: cacheType,
			TenantID:  tenantID,
		},
	}
}

// CreateTenantParams defines the parameters for creating a tenant task
type CreateTenantParams struct {
	Tenant companyconfig.Tenant `json:"tenant"`
	UserID uuid.UUID            `json:"user_id" validate:"notnil"`
}

// ImportAuth0AppsParams defines the parameters for importing Auth0 apps task
type ImportAuth0AppsParams struct {
	TenantURL  string    `json:"tenant_url"`  // the tenant URL that this message is for
	TenantID   uuid.UUID `json:"tenant_id"`   // the tenant that this message is for
	ProviderID uuid.UUID `json:"provider_id"` // the provider ID for TaskImportAuth0Apps
}

//go:generate genvalidate ImportAuth0AppsParams

// CheckTenantCNAMEParams defines the parameters for checking a tenant CNAME task
type CheckTenantCNAMEParams struct {
	TenantID    uuid.UUID `json:"tenant_id" validate:"notnil"`     // the tenant that this message is for
	TenantURLID uuid.UUID `json:"tenant_url_id" validate:"notnil"` // the tenant URL ID that this message is for

}

//go:generate genvalidate CheckTenantCNAMEParams

// FinalizeTenantCNAMEParams defines the parameters for finalizing a tenant CNAME task
type FinalizeTenantCNAMEParams struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"notnil"`   // the tenant that this message is for
	UCOrderID uuid.UUID `json:"uc_order_id" validate:"notnil"` // the order ID to finalize
}

//go:generate genvalidate FinalizeTenantCNAMEParams

// NoOpTaskParams defines the parameters for the no-op task
type NoOpTaskParams struct {
	Duration time.Duration `json:"duration"`
}

//go:generate genvalidate NoOpTaskParams

// CreateNoOpMessage creates a message to do nothing except test that the worker can receive a message and handle it.
func CreateNoOpMessage(duration time.Duration) Message {
	return Message{
		Task: TaskNoOp,
		NoOpParams: &NoOpTaskParams{
			Duration: duration,
		},
	}
}

// NewCreateTenantMessage creates a message to create a tenant
func NewCreateTenantMessage(tenant companyconfig.Tenant, userID uuid.UUID) Message {
	return Message{
		Task:         TaskCreateTenant,
		CreateTenant: &CreateTenantParams{Tenant: tenant, UserID: userID},
	}
}

// CreateCheckTenantCNameMessage creates a message to check a tenant CNAME
func CreateCheckTenantCNameMessage(tenantID uuid.UUID, tenantURLID uuid.UUID) Message {
	return Message{
		Task:     TaskCheckTenantCNAME,
		TenantID: tenantID,
		CheckTenantCNAME: &CheckTenantCNAMEParams{
			TenantID:    tenantID,
			TenantURLID: tenantURLID,
		},
	}
}

// CreateFinalizeTenantCNAMEMessage creates a message to finalize a tenant CNAME
func CreateFinalizeTenantCNAMEMessage(tenantID uuid.UUID, ucOrderID uuid.UUID) Message {
	return Message{
		Task:     TaskFinalizeTenantCNAME,
		TenantID: tenantID,
		FinalizeTenantCNAME: &FinalizeTenantCNAMEParams{
			TenantID:  tenantID,
			UCOrderID: ucOrderID,
		},
	}
}

// TenantDNSTaskParams defines the parameters for the ValidateDNS & NewTenantCNAME tasks
type TenantDNSTaskParams struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"notnil"`
	URL      string    `json:"tenant_url" validate:"notempty"`
}

//go:generate genvalidate TenantDNSTaskParams

// CreateValidateDNSMessage creates a message to validate a tenant DNS
func CreateValidateDNSMessage(tenantID uuid.UUID, url string) Message {
	return Message{
		Task:     TaskValidateDNS,
		TenantID: tenantID,
		TenantDNS: &TenantDNSTaskParams{
			TenantID: tenantID,
			URL:      url,
		},
	}
}

// CreateNewTenantCNAMEMessage creates a message to create a new tenant CNAME
func CreateNewTenantCNAMEMessage(tenantID uuid.UUID, url string) Message {
	return Message{
		Task:     TaskNewTenantCNAME,
		TenantID: tenantID,
		TenantDNS: &TenantDNSTaskParams{
			TenantID: tenantID,
			URL:      url,
		},
	}
}

// CreateImportAuth0AppsMessage creates a message to import Auth0 apps for a tenant
func CreateImportAuth0AppsMessage(tenantID uuid.UUID, tenantURL string, providerID uuid.UUID) Message {
	return Message{
		Task:     TaskImportAuth0Apps,
		TenantID: tenantID,
		ImportAuth0Apps: &ImportAuth0AppsParams{
			TenantID:   tenantID,
			TenantURL:  tenantURL,
			ProviderID: providerID,
		},
	}
}

// CreateSyncAllUsersMessage creates a message to sync all users for a tenant
func CreateSyncAllUsersMessage(tenantID uuid.UUID) Message {
	return Message{
		Task:     TaskSyncAllUsers,
		TenantID: tenantID,
	}
}

// DataImportParams defines the parameters for the DataImport tasks
type DataImportParams struct {
	JobID       uuid.UUID `json:"job_id" validate:"notnil"`
	ObjectReady bool      `json:"object_ready"`
}

//go:generate genvalidate DataImportParams

// DataCleanupParams defines the parameters for the DataCleanup tasks
type DataCleanupParams struct {
	DryRun        bool `json:"dry_run"`
	MaxCandidates int  `json:"max_candidates" validate:"notzero"`
}

//go:generate genvalidate DataCleanupParams

// TenantURLProvisioningParams defines the parameters for the ProvisionTenantURLs task
type TenantURLProvisioningParams struct {
	AddEKSURLs bool `json:"add_eks_urls"`
	DeleteURLs bool `json:"delete_urls"`
	DryRun     bool `json:"dry_run"` // only for deletion
}

//go:generate genvalidate TenantURLProvisioningParams

// ProvisionTenantOpenSearchIndexParams defines the parameters for the ProvisionTenantOpenSearchIndex task
type ProvisionTenantOpenSearchIndexParams struct {
	IndexID uuid.UUID `json:"index_id" validate:"notnil"`
}

//go:generate genvalidate ProvisionTenantOpenSearchIndexParams

// BootstrapTenantOpenSearchIndexParams defines the parameters for the BootstrapTenantOpenSearchIndex task
type BootstrapTenantOpenSearchIndexParams struct {
	IndexID                         uuid.UUID         `json:"index_id" validate:"notnil"`
	LastRegionalBootstrappedValueID uuid.UUID         `json:"last_regional_bootstrapped_value_id"`
	Region                          region.DataRegion `json:"region"`
	BatchSize                       int               `json:"batch_size"`
}

//go:generate genvalidate BootstrapTenantOpenSearchIndexParams

const maxBootstrapBatchSize = 30000

func (b *BootstrapTenantOpenSearchIndexParams) extraValidate() error {
	// some basic protections here.  We don't want to allow a batch size that is too large
	if b.BatchSize < 0 {
		return ucerr.Errorf("batch size must be greater or equal to 0. got %d", b.BatchSize)
	}
	if b.BatchSize > maxBootstrapBatchSize {
		return ucerr.Errorf("batch size must be less than or equal to %d. got %d", maxBootstrapBatchSize, b.BatchSize)
	}
	return nil
}

// DataImportMessage creates a message to import userstore data for a tenant
func DataImportMessage(tenantID uuid.UUID, importJobID uuid.UUID, objectReady bool) Message {
	return Message{
		Task:             TaskDataImport,
		TenantID:         tenantID,
		DataImportParams: &DataImportParams{JobID: importJobID, ObjectReady: objectReady},
	}
}

// PlexTokenDataCleanupMessage creates a message to trigger plex token data cleanup for a tenant
func PlexTokenDataCleanupMessage(tenantID uuid.UUID, maxCandidates int, dryRun bool) Message {
	return Message{
		Task:     TaskPlexTokenDataCleanup,
		TenantID: tenantID,
		PlexTokenDataCleanup: &DataCleanupParams{
			DryRun:        dryRun,
			MaxCandidates: maxCandidates,
		},
	}
}

// UserStoreDataCleanupMessage creates a message to trigger userstore data cleanup for a tenant
func UserStoreDataCleanupMessage(tenantID uuid.UUID, maxCandidates int, dryRun bool) Message {
	return Message{
		Task:     TaskUserStoreDataCleanup,
		TenantID: tenantID,
		UserStoreDataCleanup: &DataCleanupParams{
			DryRun:        dryRun,
			MaxCandidates: maxCandidates,
		},
	}
}

// ProvisionTenantURLsMessage creates a message to create a new tenant CNAME
func ProvisionTenantURLsMessage(tenantID uuid.UUID, addEKSURLs, deleteURLs, dryRun bool) Message {
	return Message{
		Task:     TaskProvisionTenantURLs,
		TenantID: tenantID,
		TenantURLProvisioningParams: &TenantURLProvisioningParams{
			AddEKSURLs: addEKSURLs,
			DeleteURLs: deleteURLs,
			DryRun:     dryRun,
		},
	}
}

// IngestSqlshimDatabaseSchemasParams defines the parameters for the IngestSqlshimDatabaseSchemas task
type IngestSqlshimDatabaseSchemasParams struct {
	DatabaseID uuid.UUID `json:"database_id" validate:"notnil"`
}

//go:generate genvalidate IngestSqlshimDatabaseSchemasParams

// IngestSqlshimDatabaseSchemasMessage creates a message to ingest a sqlshim database's schemas
func IngestSqlshimDatabaseSchemasMessage(tenantID uuid.UUID, databaseID uuid.UUID) Message {
	return Message{
		Task:     TaskIngestSqlshimDatabaseSchema,
		TenantID: tenantID,
		IngestSqlshimDatabaseSchemasParams: &IngestSqlshimDatabaseSchemasParams{
			DatabaseID: databaseID,
		},
	}
}

// ProvisionTenantOpenSearchIndexMessage creates a message to provision a tenant's OpenSearch index and CockroachDB change feed
func ProvisionTenantOpenSearchIndexMessage(tenantID uuid.UUID, indexID uuid.UUID) Message {
	return Message{
		Task:     TaskProvisionTenantOpenSearchIndex,
		TenantID: tenantID,
		ProvisionTenantOpenSearchIndexParams: &ProvisionTenantOpenSearchIndexParams{
			IndexID: indexID,
		},
	}
}

// BootstrapTenantOpenSearchIndexMessage creates a message to bootstrap a tenant's OpenSearch index
func BootstrapTenantOpenSearchIndexMessage(tenantID, indexID, lastRegionalBootstrappedValueID uuid.UUID, r region.DataRegion, batchSize int) Message {
	return Message{
		Task:     TaskBootstrapTenantOpenSearchIndex,
		TenantID: tenantID,
		BootstrapTenantOpenSearchIndexParams: &BootstrapTenantOpenSearchIndexParams{
			IndexID:                         indexID,
			LastRegionalBootstrappedValueID: lastRegionalBootstrappedValueID,
			Region:                          r,
			BatchSize:                       batchSize,
		},
	}
}

// UpdateTenantOpenSearchIndexParams defines the parameters for updating tenant OpenSearch index
type UpdateTenantOpenSearchIndexParams struct {
	Data    []byte `json:"data"`
	Attempt int    `json:"attempt"`
}

//go:generate genvalidate UpdateTenantOpenSearchIndexParams

// CreateUpdateTenantOpenSearchIndexMessage creates a message to update a tenant's OpenSearch index
func CreateUpdateTenantOpenSearchIndexMessage(tenantID uuid.UUID, data []byte, attempt int) Message {
	return Message{
		Task:     TaskUpdateTenantOpenSearchIndex,
		TenantID: tenantID,
		UpdateTenantOpenSearchIndexParams: &UpdateTenantOpenSearchIndexParams{
			Data:    data,
			Attempt: attempt,
		},
	}
}

// SaveTenantInternalParams defines the parameters for saving tenant internal data
type SaveTenantInternalParams struct {
	TenantInternal *companyconfig.TenantInternal `json:"tenant_internal"`
}

// SaveTenantInternalMessage creates a message to save tenant internal data
func SaveTenantInternalMessage(ti *companyconfig.TenantInternal) Message {
	return Message{
		Task:     TaskSaveTenantInternal,
		TenantID: ti.ID,
		SaveTenantInternalParams: &SaveTenantInternalParams{
			TenantInternal: ti,
		},
	}
}
