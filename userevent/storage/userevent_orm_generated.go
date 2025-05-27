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
	"userclouds.com/userevent"
)

// IsUserEventSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsUserEventSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM user_events WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsUserEventSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetUserEvent loads a UserEvent by ID
func (s *Storage) GetUserEvent(ctx context.Context, id uuid.UUID) (*userevent.UserEvent, error) {
	const q = "SELECT id, updated, deleted, type, user_alias, payload, created FROM user_events WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj userevent.UserEvent
	if err := s.db.GetContext(ctx, "GetUserEvent", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "UserEvent %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetUserEventSoftDeleted loads a UserEvent by ID iff it's soft-deleted
func (s *Storage) GetUserEventSoftDeleted(ctx context.Context, id uuid.UUID) (*userevent.UserEvent, error) {
	const q = "SELECT id, updated, deleted, type, user_alias, payload, created FROM user_events WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj userevent.UserEvent
	if err := s.db.GetContext(ctx, "GetUserEventSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted UserEvent %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetUserEventsForIDs loads multiple UserEvent for a given list of IDs
func (s *Storage) GetUserEventsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]userevent.UserEvent, error) {
	items := make([]userevent.UserEvent, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getUserEventsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getUserEventsHelperForIDs loads multiple UserEvent for a given list of IDs from the DB
func (s *Storage) getUserEventsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]userevent.UserEvent, error) {
	const q = "SELECT id, updated, deleted, type, user_alias, payload, created FROM user_events WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []userevent.UserEvent
	if err := s.db.SelectContextWithDirty(ctx, "GetUserEventsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested UserEvents  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListUserEventsPaginated loads a paginated list of UserEvents for the specified paginator settings
func (s *Storage) ListUserEventsPaginated(ctx context.Context, p pagination.Paginator) ([]userevent.UserEvent, *pagination.ResponseFields, error) {
	return s.listInnerUserEventsPaginated(ctx, p, false)
}

// listInnerUserEventsPaginated loads a paginated list of UserEvents for the specified paginator settings
func (s *Storage) listInnerUserEventsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]userevent.UserEvent, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, type, user_alias, payload, created FROM (SELECT id, updated, deleted, type, user_alias, payload, created FROM user_events WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []userevent.UserEvent
	if err := s.db.SelectContext(ctx, "ListUserEventsPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveUserEvent saves a UserEvent
func (s *Storage) SaveUserEvent(ctx context.Context, obj *userevent.UserEvent) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerUserEvent(ctx, obj))
}

// SaveUserEvent saves a UserEvent
func (s *Storage) saveInnerUserEvent(ctx context.Context, obj *userevent.UserEvent) error {
	const q = "INSERT INTO user_events (id, updated, deleted, type, user_alias, payload) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type = $3, user_alias = $4, payload = $5 WHERE (user_events.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveUserEvent", obj, q, obj.ID, obj.Deleted, obj.Type, obj.UserAlias, obj.Payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "UserEvent %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteUserEvent soft-deletes a UserEvent which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteUserEvent(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerUserEvent(ctx, objID, false))
}

// deleteInnerUserEvent soft-deletes a UserEvent which is currently alive
func (s *Storage) deleteInnerUserEvent(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE user_events SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteUserEvent", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting UserEvent %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "UserEvent %v not found", objID)
	}
	return nil
}
