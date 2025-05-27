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

	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

// IsColumnValueRetentionDurationSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsColumnValueRetentionDurationSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *ColumnValueRetentionDuration
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[ColumnValueRetentionDuration](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnValueRetentionDurationKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsColumnValueRetentionDurationSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsColumnValueRetentionDurationSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM column_value_retention_durations WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsColumnValueRetentionDurationSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetColumnValueRetentionDuration loads a ColumnValueRetentionDuration by ID
func (s *Storage) GetColumnValueRetentionDuration(ctx context.Context, id uuid.UUID) (*ColumnValueRetentionDuration, error) {
	return cache.ServerGetItem(ctx, s.cm, id, ColumnValueRetentionDurationKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *ColumnValueRetentionDuration) error {
			const q = "SELECT id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration, created FROM column_value_retention_durations WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetColumnValueRetentionDuration", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "ColumnValueRetentionDuration %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetColumnValueRetentionDurationSoftDeleted loads a ColumnValueRetentionDuration by ID iff it's soft-deleted
func (s *Storage) GetColumnValueRetentionDurationSoftDeleted(ctx context.Context, id uuid.UUID) (*ColumnValueRetentionDuration, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *ColumnValueRetentionDuration
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[ColumnValueRetentionDuration](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnValueRetentionDurationKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetColumnValueRetentionDurationSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetColumnValueRetentionDurationSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted ColumnValueRetentionDuration %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration, created FROM column_value_retention_durations WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj ColumnValueRetentionDuration
	if err := s.db.GetContextWithDirty(ctx, "GetColumnValueRetentionDurationSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted ColumnValueRetentionDuration %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetColumnValueRetentionDurationsForIDs loads multiple ColumnValueRetentionDuration for a given list of IDs
func (s *Storage) GetColumnValueRetentionDurationsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]ColumnValueRetentionDuration, error) {
	items := make([]ColumnValueRetentionDuration, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if len(ids) == 0 {
		return items, nil
	}

	if len(ids) != missed.Size() {
		// We have duplicate IDs in the list
		ids = missed.Items()
	}

	var cachedItemsCount, dbItemsCount int
	sentinelsMap := make(map[uuid.UUID]cache.Sentinel)
	if s.cm != nil {
		keys := make([]cache.Key, 0, len(ids))
		modKeys := make([]cache.Key, 0, len(ids))
		locks := make([]bool, 0, len(keys))
		for _, id := range ids {
			locks = append(locks, true)
			keys = append(keys, s.cm.N.GetKeyNameWithID(ColumnValueRetentionDurationKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[ColumnValueRetentionDuration](ctx, *s.cm, keys, modKeys, locks)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for i, item := range cachedItems {
			if item != nil {
				items = append(items, *item)
				missed.Evict(item.ID)
				cachedItemsCount++
			} else if sentinels != nil { // sentinels array will be nil if we are not using a cache
				sentinelsMap[ids[i]] = sentinels[i]
			}
		}
		dirty = cdirty
	}
	if missed.Size() > 0 {
		itemsFromDB, err := s.getColumnValueRetentionDurationsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
		dbItemsCount = len(itemsFromDB)
		if s.cm != nil && len(sentinelsMap) > 0 {
			for _, item := range itemsFromDB {
				cache.SaveItemToCache(ctx, *s.cm, item, sentinelsMap[item.ID], false, nil)
			}
		}
	}
	uclog.Verbosef(ctx, "GetColumnValueRetentionDurationMap: returning %d ColumnValueRetentionDuration. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getColumnValueRetentionDurationsHelperForIDs loads multiple ColumnValueRetentionDuration for a given list of IDs from the DB
func (s *Storage) getColumnValueRetentionDurationsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]ColumnValueRetentionDuration, error) {
	const q = "SELECT id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration, created FROM column_value_retention_durations WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []ColumnValueRetentionDuration
	if err := s.db.SelectContextWithDirty(ctx, "GetColumnValueRetentionDurationsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested ColumnValueRetentionDurations  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListColumnValueRetentionDurationsPaginated loads a paginated list of ColumnValueRetentionDurations for the specified paginator settings
func (s *Storage) ListColumnValueRetentionDurationsPaginated(ctx context.Context, p pagination.Paginator) ([]ColumnValueRetentionDuration, *pagination.ResponseFields, error) {
	return s.listInnerColumnValueRetentionDurationsPaginated(ctx, p, false)
}

// listInnerColumnValueRetentionDurationsPaginated loads a paginated list of ColumnValueRetentionDurations for the specified paginator settings
func (s *Storage) listInnerColumnValueRetentionDurationsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]ColumnValueRetentionDuration, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(ColumnValueRetentionDurationCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]ColumnValueRetentionDuration
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[ColumnValueRetentionDuration](ctx, *s.cm, ckey, cachable)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		if cachable && v != nil {
			v, respFields := pagination.ProcessResults(*v, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
			return v, &respFields, nil
		}
	}
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration, created FROM (SELECT id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration, created FROM column_value_retention_durations WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []ColumnValueRetentionDuration
	if err := s.db.SelectContextWithDirty(ctx, "ListColumnValueRetentionDurationsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
	if s.cm != nil && cachable && !respFields.HasNext && !respFields.HasPrev { /* only cache single page collections */
		cache.SaveItemsToCollection(ctx, *s.cm, ColumnValueRetentionDuration{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveColumnValueRetentionDuration saves a ColumnValueRetentionDuration
func (s *Storage) SaveColumnValueRetentionDuration(ctx context.Context, obj *ColumnValueRetentionDuration) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, ColumnValueRetentionDurationKeyID, nil, func(i *ColumnValueRetentionDuration) error {
		return ucerr.Wrap(s.saveInnerColumnValueRetentionDuration(ctx, obj))
	}))
}

// SaveColumnValueRetentionDuration saves a ColumnValueRetentionDuration
func (s *Storage) saveInnerColumnValueRetentionDuration(ctx context.Context, obj *ColumnValueRetentionDuration) error {
	// this query has three basic parts
	// 1) INSERT INTO column_value_retention_durations is used for create only ... any updates will fail with a CONFLICT on (id, deleted)
	// 2) in that case, WHERE will take over to chose the correct row (if any) to update. This includes a check that obj.Version ($3)
	//    matches the _version currently in the database, so that we aren't writing stale data. If this fails, sql.ErrNoRows is returned.
	// 3) if the WHERE matched a row (including version check), the UPDATE will set the new values including $[max] which is newVersion,
	//    which is set to the current version + 1. This is returned in the RETURNING clause so that we can update obj.Version with the new value.
	newVersion := obj.Version + 1
	const q = "INSERT INTO column_value_retention_durations (id, updated, deleted, _version, column_id, purpose_id, duration_type, duration_unit, duration) VALUES ($1, CLOCK_TIMESTAMP(), $2, $9, $4, $5, $6, $7, $8) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, _version = $9, column_id = $4, purpose_id = $5, duration_type = $6, duration_unit = $7, duration = $8 WHERE (column_value_retention_durations._version = $3 AND column_value_retention_durations.id = $1) RETURNING created, updated, _version; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveColumnValueRetentionDuration", obj, q, obj.ID, obj.Deleted, obj.Version, obj.ColumnID, obj.PurposeID, obj.DurationType, obj.DurationUnit, obj.Duration, newVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "ColumnValueRetentionDuration %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteColumnValueRetentionDuration soft-deletes a ColumnValueRetentionDuration which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteColumnValueRetentionDuration(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerColumnValueRetentionDuration(ctx, objID, false))
}

// deleteInnerColumnValueRetentionDuration soft-deletes a ColumnValueRetentionDuration which is currently alive
func (s *Storage) deleteInnerColumnValueRetentionDuration(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[ColumnValueRetentionDuration](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnValueRetentionDurationKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := ColumnValueRetentionDuration{VersionBaseModel: ucdb.NewVersionBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[ColumnValueRetentionDuration](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE column_value_retention_durations SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteColumnValueRetentionDuration", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting ColumnValueRetentionDuration %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "ColumnValueRetentionDuration %v not found", objID)
	}
	return nil
}

// FlushCacheForColumnValueRetentionDuration flushes cache for ColumnValueRetentionDuration. It may flush a larger scope then
func (s *Storage) FlushCacheForColumnValueRetentionDuration(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "ColumnValueRetentionDuration"))
}
