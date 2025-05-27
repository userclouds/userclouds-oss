package storage

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantdb"
)

// Storage manages sync storage
type Storage struct {
	db *ucdb.DB
}

// New creates a new storage object
func New(db *ucdb.DB) *Storage {
	return &Storage{db: db}
}

// NewFromConfig creates a new storage object from a DBConfig
func NewFromConfig(ctx context.Context, cfg *ucdb.Config) (*Storage, error) {
	db, err := ucdb.New(ctx, cfg, migrate.SchemaValidator(tenantdb.Schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return New(db), nil
}

// GetLatestSuccessfulIDPSyncRun returns the latest IDP sync run
func (s *Storage) GetLatestSuccessfulIDPSyncRun(ctx context.Context, activeProviderID uuid.UUID) (*IDPSyncRun, error) {
	const q = "SELECT id, created, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records FROM idp_sync_runs WHERE active_provider_id=$1 AND error='' AND type='user' AND deleted='0001-01-01 00:00:00' ORDER BY created DESC LIMIT 1;"

	var obj IDPSyncRun
	if err := s.db.GetContext(ctx, "GetLatestSuccessfulIDPSyncRun", &obj, q, activeProviderID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// SetDataImportJobStatus sets the status of a data import job and saves it, errors are logged but otherwise ignored
func (s *Storage) SetDataImportJobStatus(ctx context.Context, job *IDPDataImportJob, newStatus IDPDataImportJobStatus) {
	uclog.Debugf(ctx, "setting data import job %v status %v -> %v", job.ID, job.Status, newStatus)
	job.Status = newStatus
	s.SaveDataImportJobNoError(ctx, job)
}

// SaveDataImportJobNoError saves a data import job, errors are logged but otherwise ignored
func (s *Storage) SaveDataImportJobNoError(ctx context.Context, job *IDPDataImportJob) {
	if err := s.SaveIDPDataImportJob(ctx, job); err != nil {
		uclog.Errorf(ctx, "failed to save data import job %v: %v", job.ID, err)
	}
}
