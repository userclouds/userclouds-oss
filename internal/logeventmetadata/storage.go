package logeventmetadata

import (
	"context"
	"database/sql"
	"errors"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
)

// Storage provides an object for database access
type Storage struct {
	db *ucdb.DB
}

// NewStorage returns a Storage object
func NewStorage(db *ucdb.DB) *Storage {
	return &Storage{db: db}
}

// NewStorageFromConfig returns a new DB-backed companyconfig.Storage object
// to access company & tenant metadata using DB config.
func NewStorageFromConfig(ctx context.Context, cfg *ucdb.Config) (*Storage, error) {
	db, err := ucdb.New(ctx, cfg, migrate.SchemaValidator(companyconfig.Schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return NewStorage(db), nil
}

// GetEventMetadataCount get the count of rows (including deleted) in the event_metadata table
func (s *Storage) GetEventMetadataCount(ctx context.Context) (int, error) {
	var count int
	if err := s.db.GetContext(ctx, "GetEventMetadataCount", &count, "select count(*) from event_metadata /* lint-deleted-ok */"); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, ucerr.Wrap(err)
	}

	return count, nil
}

// GetMaxEventCode get the greatest current event code in the event metadata table for tenant
func (s *Storage) GetMaxEventCode(ctx context.Context) (uclog.EventCode, error) {
	var maxcode uclog.EventCode
	var count int
	if err := s.db.GetContext(ctx, "GetMaxEventCode.Count", &count, "select count(*) from event_metadata /* lint-deleted-ok */"); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, ucerr.Wrap(err)
	}

	if count == 0 {
		return 0, nil
	}

	// For now we don't allow reuse of codes from previously created by deleted custom types
	if err := s.db.GetContext(ctx, "GetMaxEventCode.Max", &maxcode, "select max(code) from event_metadata /* lint-deleted-ok */"); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, ucerr.Wrap(err)
	}
	return maxcode, nil
}

// GetMetricsMetadataByURL returns the set of event types that corespond to a particular objects defined by the URL
func (s *Storage) GetMetricsMetadataByURL(ctx context.Context, referenceURL string) error {
	const q = "SELECT  event_metadata WHERE url=$1 AND deleted='0001-01-01 00:00:00';"
	metricsMetadata := []MetricMetadata{}
	if err := s.db.SelectContext(ctx, "GetMetricsMetadataByURL", &metricsMetadata, q, referenceURL); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}
	return nil
}

// GetMetricMetadataByStringID loads a MetricMetadata by StringID
func (s *Storage) GetMetricMetadataByStringID(ctx context.Context, stringID string) (*MetricMetadata, error) {
	const q = "SELECT id, created, updated, deleted, service, category, string_id, code, name, url, description, attributes FROM event_metadata WHERE string_id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj MetricMetadata
	if err := s.db.GetContext(ctx, "GetMetricMetadataByStringID", &obj, q, stringID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetMetricMetadataByCode loads a MetricMetadata by code
func (s *Storage) GetMetricMetadataByCode(ctx context.Context, code uclog.EventCode) (*MetricMetadata, error) {
	const q = "SELECT id, created, updated, deleted, service, category, string_id, code, name, url, description, attributes FROM event_metadata WHERE code=$1 AND deleted='0001-01-01 00:00:00';"

	var obj MetricMetadata
	if err := s.db.GetContext(ctx, "GetMetricMetadataByCode", &obj, q, code); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// UpdateMetricMetadata updates fields that can be updated on existing object
func (s *Storage) UpdateMetricMetadata(ctx context.Context, obj *MetricMetadata) error {
	const q = "UPDATE event_metadata SET updated=NOW(), service=$2, category=$3, string_id=$4, name=$5, url=$6, description=$7, attributes=$8 WHERE id=$1 AND code = $9 AND deleted='0001-01-01 00:00:00' RETURNING updated;"
	return ucerr.Wrap(s.db.GetContext(ctx, "UpdateMetricMetadata", obj, q, obj.ID, obj.Service, obj.Category, obj.StringID, obj.Name, obj.ReferenceURL, obj.Description, obj.Attributes, obj.Code))
}

// DeleteMetricsMetadataByURL soft-deletes a MetricMetadata which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteMetricsMetadataByURL(ctx context.Context, referenceURL string) error {
	const q = "UPDATE event_metadata SET deleted=NOW() WHERE url=$1 AND deleted='0001-01-01 00:00:00';"
	_, err := s.db.ExecContext(ctx, "DeleteMetricsMetadataByURL", q, referenceURL)
	if !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}
	return nil
}

// NukeNonCustomEvent hard deletes all static event rows
func (s *Storage) NukeNonCustomEvent(ctx context.Context) error {
	const q = "DELETE FROM event_metadata WHERE attributes->>'system' = 'true';"
	var count int
	if err := s.db.GetContext(ctx, "NukeNonCustomEvent", &count, q); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}
	return nil
}
