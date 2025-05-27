// NOTE: automatically generated file -- DO NOT EDIT

package auditlog

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

// IsEntrySoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsEntrySoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM auditlog WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsEntrySoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetEntry loads a Entry by ID
func (s *Storage) GetEntry(ctx context.Context, id uuid.UUID) (*Entry, error) {
	const q = "SELECT id, updated, deleted, type, actor_id, payload, created FROM auditlog WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj Entry
	if err := s.db.GetContext(ctx, "GetEntry", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Entry %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetEntrySoftDeleted loads a Entry by ID iff it's soft-deleted
func (s *Storage) GetEntrySoftDeleted(ctx context.Context, id uuid.UUID) (*Entry, error) {
	const q = "SELECT id, updated, deleted, type, actor_id, payload, created FROM auditlog WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Entry
	if err := s.db.GetContext(ctx, "GetEntrySoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Entry %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetEntrysForIDs loads multiple Entry for a given list of IDs
func (s *Storage) GetEntrysForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Entry, error) {
	items := make([]Entry, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getEntriesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getEntriesHelperForIDs loads multiple Entry for a given list of IDs from the DB
func (s *Storage) getEntriesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Entry, error) {
	const q = "SELECT id, updated, deleted, type, actor_id, payload, created FROM auditlog WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Entry
	if err := s.db.SelectContextWithDirty(ctx, "GetEntriesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Entries  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListEntriesPaginated loads a paginated list of Entries for the specified paginator settings
func (s *Storage) ListEntriesPaginated(ctx context.Context, p pagination.Paginator) ([]Entry, *pagination.ResponseFields, error) {
	return s.listInnerEntriesPaginated(ctx, p, false)
}

// listInnerEntriesPaginated loads a paginated list of Entries for the specified paginator settings
func (s *Storage) listInnerEntriesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Entry, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, type, actor_id, payload, created FROM (SELECT id, updated, deleted, type, actor_id, payload, created FROM auditlog WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Entry
	if err := s.db.SelectContext(ctx, "ListEntriesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveEntry saves a Entry
func (s *Storage) SaveEntry(ctx context.Context, obj *Entry) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerEntry(ctx, obj))
}

// SaveEntry saves a Entry
func (s *Storage) saveInnerEntry(ctx context.Context, obj *Entry) error {
	const q = "INSERT INTO auditlog (id, updated, deleted, type, actor_id, payload) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type = $3, actor_id = $4, payload = $5 WHERE (auditlog.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveEntry", obj, q, obj.ID, obj.Deleted, obj.Type, obj.Actor, obj.Payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Entry %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteEntry soft-deletes a Entry which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteEntry(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerEntry(ctx, objID, false))
}

// deleteInnerEntry soft-deletes a Entry which is currently alive
func (s *Storage) deleteInnerEntry(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE auditlog SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteEntry", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Entry %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Entry %v not found", objID)
	}
	return nil
}
