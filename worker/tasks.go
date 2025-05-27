package worker

// Task defines the worker task to run
type Task string

// Task names
const (
	TaskNoOp            Task = "noop" // For testing
	TaskSyncAllUsers    Task = "sync_all_users"
	TaskImportAuth0Apps Task = "import_auth0_apps"

	TaskNewTenantCNAME      Task = "new_tenant_cname"
	TaskValidateDNS         Task = "validate_tenant_dns"
	TaskFinalizeTenantCNAME Task = "finalize_tenant_cname"
	TaskCheckTenantCNAME    Task = "check_tenant_cname"

	TaskCreateTenant                   Task = "create_tenant"
	TaskClearCache                     Task = "clear_cache"
	TaskLogCache                       Task = "log_cache"
	TaskDataImport                     Task = "data_import"
	TaskPlexTokenDataCleanup           Task = "plex_token_data_cleanup"
	TaskUserStoreDataCleanup           Task = "userstore_data_cleanup"
	TaskProvisionTenantURLs            Task = "provision_tenant_urls"
	TaskIngestSqlshimDatabaseSchema    Task = "ingest_sqlshim_database_schema"
	TaskProvisionTenantOpenSearchIndex Task = "provision_tenant_opensearch_index"
	TaskBootstrapTenantOpenSearchIndex Task = "bootstrap_tenant_opensearch_index"
	TaskUpdateTenantOpenSearchIndex    Task = "update_tenant_opensearch_index"
	TaskSaveTenantInternal             Task = "save_tenant_internal"
)
