package storage

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/uuidarray"
)

// SyncRunType is the type of sync run we're recording
type SyncRunType string

// SyncRunType values
const (
	SyncRunTypeUser SyncRunType = "user"
	SyncRunTypeApp  SyncRunType = "app"
)

//go:generate genconstant SyncRunType

// IDPSyncRun stores a record of each time we sync user data between a tenant's IDPs
type IDPSyncRun struct {
	ucdb.BaseModel

	Type SyncRunType `db:"type" validate:"notempty"`

	ActiveProviderID    uuid.UUID           `db:"active_provider_id"`
	FollowerProviderIDs uuidarray.UUIDArray `db:"follower_provider_ids" validate:"skip"`

	// these are explicitly used for user sync and ignored elsewhere today
	Since time.Time `db:"since"` // lower bound of the sync
	Until time.Time `db:"until"` // the time stamp we used as the upper bound for the sync

	// TODO rename FailedRecords to ErrorRecords for consistency?
	Error          string `db:"error"`           // empty if the sync finished successfully
	TotalRecords   int    `db:"total_records"`   // the number of records that were synced during this run
	FailedRecords  int    `db:"failed_records"`  // the number of records that failed to sync during this run
	WarningRecords int    `db:"warning_records"` // the number of records that synced with warnings
}

//go:generate genpageable IDPSyncRun
//go:generate genvalidate IDPSyncRun

//go:generate genorm IDPSyncRun idp_sync_runs tenantdb

func (IDPSyncRun) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"type": pagination.StringKeyType,
	}
}

// IDPSyncRecord stores a record of each user that was synced during a sync run
type IDPSyncRecord struct {
	ucdb.BaseModel

	SyncRunID uuid.UUID `db:"sync_run_id"`
	ObjectID  string    `db:"object_id"`

	Error   string `db:"error"`
	Warning string `db:"warning"`

	// TODO: deprecated, remove after deploy
	UserID string `db:"user_id"`
}

//go:generate genpageable IDPSyncRecord
//go:generate genvalidate IDPSyncRecord

//go:generate genorm IDPSyncRecord idp_sync_records tenantdb

func (IDPSyncRecord) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"sync_run_id": pagination.UUIDKeyType,
	}
}

// FailedRecords is a slice of strings
type FailedRecords []string

// Value implements sql.Valuer
func (o FailedRecords) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *FailedRecords) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.Errorf("type assertion failed for Config.Scan(), got %T", value)
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}

// IDPDataImportJobStatus is the status of an IDPDataImportJob
type IDPDataImportJobStatus string

const (
	// IDPDataImportJobStatusPending is the status of an IDPDataImportJob when it is waiting for the file to be uploaded to s3 (initial status of the job)
	IDPDataImportJobStatusPending IDPDataImportJobStatus = "Pending"

	// IDPDataImportJobStatusInProgress is the status of an IDPDataImportJob when it is being imported
	IDPDataImportJobStatusInProgress IDPDataImportJobStatus = "Import in progress"

	// IDPDataImportJobStatusCompleted is the status of an IDPDataImportJob when it has been imported
	IDPDataImportJobStatusCompleted IDPDataImportJobStatus = "Completed"

	// IDPDataImportJobStatusFailed is the status of an IDPDataImportJob when it has failed to import
	IDPDataImportJobStatusFailed IDPDataImportJobStatus = "Failed"

	// IDPDataImportJobStatusExpired is the status of an IDPDataImportJob when the upload endpoint has expired
	IDPDataImportJobStatusExpired IDPDataImportJobStatus = "Upload endpoint expired"
)

// IDPDataImportJob stores a record of each time we import user data from a file into userstore
type IDPDataImportJob struct {
	ucdb.BaseModel
	LastRunTime          time.Time              `db:"last_run_time"`
	ImportType           string                 `db:"import_type" validate:"notempty"`
	Status               IDPDataImportJobStatus `db:"status" validate:"notempty"`
	Error                string                 `db:"error"`
	S3Bucket             string                 `db:"s3_bucket" validate:"notempty"`
	ObjectKey            string                 `db:"object_key" validate:"notempty"`
	ExpirationMinutes    int                    `db:"expiration_minutes"`
	FileSize             int64                  `db:"file_size"`
	ProcessedSize        int64                  `db:"processed_size"`
	ProcessedRecordCount int                    `db:"processed_record_count"`
	FailedRecords        FailedRecords          `db:"failed_records"`
	FailedRecordCount    int                    `db:"failed_record_count"`
}

//go:generate genorm IDPDataImportJob idp_data_import_jobs tenantdb

//go:generate genpageable IDPDataImportJob

//go:generate genvalidate IDPDataImportJob
