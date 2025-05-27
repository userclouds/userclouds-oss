// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// IsIDPDataImportJobSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsIDPDataImportJobSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM idp_data_import_jobs WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsIDPDataImportJobSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetIDPDataImportJob loads a IDPDataImportJob by ID
func (s *Storage) GetIDPDataImportJob(ctx context.Context, id uuid.UUID) (*IDPDataImportJob, error) {
	const q = "SELECT id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count, created FROM idp_data_import_jobs WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj IDPDataImportJob
	if err := s.db.GetContext(ctx, "GetIDPDataImportJob", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "IDPDataImportJob %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetIDPDataImportJobSoftDeleted loads a IDPDataImportJob by ID iff it's soft-deleted
func (s *Storage) GetIDPDataImportJobSoftDeleted(ctx context.Context, id uuid.UUID) (*IDPDataImportJob, error) {
	const q = "SELECT id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count, created FROM idp_data_import_jobs WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj IDPDataImportJob
	if err := s.db.GetContext(ctx, "GetIDPDataImportJobSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted IDPDataImportJob %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetIDPDataImportJobsForIDs loads multiple IDPDataImportJob for a given list of IDs
func (s *Storage) GetIDPDataImportJobsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]IDPDataImportJob, error) {
	items := make([]IDPDataImportJob, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getIDPDataImportJobsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getIDPDataImportJobsHelperForIDs loads multiple IDPDataImportJob for a given list of IDs from the DB
func (s *Storage) getIDPDataImportJobsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]IDPDataImportJob, error) {
	const q = "SELECT id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count, created FROM idp_data_import_jobs WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []IDPDataImportJob
	if err := s.db.SelectContextWithDirty(ctx, "GetIDPDataImportJobsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested IDPDataImportJobs  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListIDPDataImportJobsPaginated loads a paginated list of IDPDataImportJobs for the specified paginator settings
func (s *Storage) ListIDPDataImportJobsPaginated(ctx context.Context, p pagination.Paginator) ([]IDPDataImportJob, *pagination.ResponseFields, error) {
	return s.listInnerIDPDataImportJobsPaginated(ctx, p, false)
}

// listInnerIDPDataImportJobsPaginated loads a paginated list of IDPDataImportJobs for the specified paginator settings
func (s *Storage) listInnerIDPDataImportJobsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]IDPDataImportJob, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count, created FROM (SELECT id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count, created FROM idp_data_import_jobs WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []IDPDataImportJob
	if err := s.db.SelectContext(ctx, "ListIDPDataImportJobsPaginated", &objsDB, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	objs, respFields := pagination.ProcessResults(objsDB, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// SaveIDPDataImportJob saves a IDPDataImportJob
func (s *Storage) SaveIDPDataImportJob(ctx context.Context, obj *IDPDataImportJob) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerIDPDataImportJob(ctx, obj))
}

// SaveIDPDataImportJob saves a IDPDataImportJob
func (s *Storage) saveInnerIDPDataImportJob(ctx context.Context, obj *IDPDataImportJob) error {
	const q = "INSERT INTO idp_data_import_jobs (id, updated, deleted, last_run_time, import_type, status, error, s3_bucket, object_key, expiration_minutes, file_size, processed_size, processed_record_count, failed_records, failed_record_count) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, last_run_time = $3, import_type = $4, status = $5, error = $6, s3_bucket = $7, object_key = $8, expiration_minutes = $9, file_size = $10, processed_size = $11, processed_record_count = $12, failed_records = $13, failed_record_count = $14 WHERE (idp_data_import_jobs.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveIDPDataImportJob", obj, q, obj.ID, obj.Deleted, obj.LastRunTime, obj.ImportType, obj.Status, obj.Error, obj.S3Bucket, obj.ObjectKey, obj.ExpirationMinutes, obj.FileSize, obj.ProcessedSize, obj.ProcessedRecordCount, obj.FailedRecords, obj.FailedRecordCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "IDPDataImportJob %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteIDPDataImportJob soft-deletes a IDPDataImportJob which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteIDPDataImportJob(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerIDPDataImportJob(ctx, objID, false))
}

// deleteInnerIDPDataImportJob soft-deletes a IDPDataImportJob which is currently alive
func (s *Storage) deleteInnerIDPDataImportJob(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE idp_data_import_jobs SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteIDPDataImportJob", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting IDPDataImportJob %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "IDPDataImportJob %v not found", objID)
	}
	return nil
}
