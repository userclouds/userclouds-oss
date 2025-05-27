// NOTE: automatically generated file -- DO NOT EDIT

package logeventmetadata

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

// IsMetricMetadataSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsMetricMetadataSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM event_metadata WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsMetricMetadataSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetMetricMetadata loads a MetricMetadata by ID
func (s *Storage) GetMetricMetadata(ctx context.Context, id uuid.UUID) (*MetricMetadata, error) {
	const q = "SELECT id, updated, deleted, service, category, string_id, code, name, url, description, attributes, created FROM event_metadata WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj MetricMetadata
	if err := s.db.GetContext(ctx, "GetMetricMetadata", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "MetricMetadata %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetMetricMetadataSoftDeleted loads a MetricMetadata by ID iff it's soft-deleted
func (s *Storage) GetMetricMetadataSoftDeleted(ctx context.Context, id uuid.UUID) (*MetricMetadata, error) {
	const q = "SELECT id, updated, deleted, service, category, string_id, code, name, url, description, attributes, created FROM event_metadata WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj MetricMetadata
	if err := s.db.GetContext(ctx, "GetMetricMetadataSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted MetricMetadata %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetMetricMetadatasForIDs loads multiple MetricMetadata for a given list of IDs
func (s *Storage) GetMetricMetadatasForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]MetricMetadata, error) {
	items := make([]MetricMetadata, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getMetricMetadatasHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getMetricMetadatasHelperForIDs loads multiple MetricMetadata for a given list of IDs from the DB
func (s *Storage) getMetricMetadatasHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]MetricMetadata, error) {
	const q = "SELECT id, updated, deleted, service, category, string_id, code, name, url, description, attributes, created FROM event_metadata WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []MetricMetadata
	if err := s.db.SelectContextWithDirty(ctx, "GetMetricMetadatasForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested MetricMetadatas  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListMetricMetadatasPaginated loads a paginated list of MetricMetadatas for the specified paginator settings
func (s *Storage) ListMetricMetadatasPaginated(ctx context.Context, p pagination.Paginator) ([]MetricMetadata, *pagination.ResponseFields, error) {
	return s.listInnerMetricMetadatasPaginated(ctx, p, false)
}

// listInnerMetricMetadatasPaginated loads a paginated list of MetricMetadatas for the specified paginator settings
func (s *Storage) listInnerMetricMetadatasPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]MetricMetadata, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, service, category, string_id, code, name, url, description, attributes, created FROM (SELECT id, updated, deleted, service, category, string_id, code, name, url, description, attributes, created FROM event_metadata WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []MetricMetadata
	if err := s.db.SelectContext(ctx, "ListMetricMetadatasPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveMetricMetadata saves a MetricMetadata
func (s *Storage) SaveMetricMetadata(ctx context.Context, obj *MetricMetadata) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerMetricMetadata(ctx, obj))
}

// SaveMetricMetadata saves a MetricMetadata
func (s *Storage) saveInnerMetricMetadata(ctx context.Context, obj *MetricMetadata) error {
	const q = "INSERT INTO event_metadata (id, updated, deleted, service, category, string_id, code, name, url, description, attributes) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, service = $3, category = $4, string_id = $5, code = $6, name = $7, url = $8, description = $9, attributes = $10 WHERE (event_metadata.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveMetricMetadata", obj, q, obj.ID, obj.Deleted, obj.Service, obj.Category, obj.StringID, obj.Code, obj.Name, obj.ReferenceURL, obj.Description, obj.Attributes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "MetricMetadata %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteMetricMetadata soft-deletes a MetricMetadata which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteMetricMetadata(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerMetricMetadata(ctx, objID, false))
}

// deleteInnerMetricMetadata soft-deletes a MetricMetadata which is currently alive
func (s *Storage) deleteInnerMetricMetadata(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE event_metadata SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteMetricMetadata", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting MetricMetadata %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "MetricMetadata %v not found", objID)
	}
	return nil
}
