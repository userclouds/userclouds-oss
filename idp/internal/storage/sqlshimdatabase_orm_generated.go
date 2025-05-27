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

// IsSQLShimDatabaseSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsSQLShimDatabaseSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *SQLShimDatabase
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[SQLShimDatabase](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SQLShimDatabaseKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsSQLShimDatabaseSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsSQLShimDatabaseSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM sqlshim_databases WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsSQLShimDatabaseSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetSQLShimDatabase loads a SQLShimDatabase by ID
func (s *Storage) GetSQLShimDatabase(ctx context.Context, id uuid.UUID) (*SQLShimDatabase, error) {
	return cache.ServerGetItem(ctx, s.cm, id, SQLShimDatabaseKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *SQLShimDatabase) error {
			const q = "SELECT id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled, created FROM sqlshim_databases WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetSQLShimDatabase", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "SQLShimDatabase %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetSQLShimDatabaseSoftDeleted loads a SQLShimDatabase by ID iff it's soft-deleted
func (s *Storage) GetSQLShimDatabaseSoftDeleted(ctx context.Context, id uuid.UUID) (*SQLShimDatabase, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *SQLShimDatabase
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[SQLShimDatabase](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SQLShimDatabaseKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetSQLShimDatabaseSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetSQLShimDatabaseSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted SQLShimDatabase %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled, created FROM sqlshim_databases WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj SQLShimDatabase
	if err := s.db.GetContextWithDirty(ctx, "GetSQLShimDatabaseSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted SQLShimDatabase %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetSQLShimDatabasesForIDs loads multiple SQLShimDatabase for a given list of IDs
func (s *Storage) GetSQLShimDatabasesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]SQLShimDatabase, error) {
	items := make([]SQLShimDatabase, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(SQLShimDatabaseKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[SQLShimDatabase](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getSQLShimDatabasesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetSQLShimDatabaseMap: returning %d SQLShimDatabase. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getSQLShimDatabasesHelperForIDs loads multiple SQLShimDatabase for a given list of IDs from the DB
func (s *Storage) getSQLShimDatabasesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]SQLShimDatabase, error) {
	const q = "SELECT id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled, created FROM sqlshim_databases WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []SQLShimDatabase
	if err := s.db.SelectContextWithDirty(ctx, "GetSQLShimDatabasesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested SQLShimDatabases  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListSQLShimDatabasesPaginated loads a paginated list of SQLShimDatabases for the specified paginator settings
func (s *Storage) ListSQLShimDatabasesPaginated(ctx context.Context, p pagination.Paginator) ([]SQLShimDatabase, *pagination.ResponseFields, error) {
	return s.listInnerSQLShimDatabasesPaginated(ctx, p, false)
}

// listInnerSQLShimDatabasesPaginated loads a paginated list of SQLShimDatabases for the specified paginator settings
func (s *Storage) listInnerSQLShimDatabasesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]SQLShimDatabase, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(SQLShimDatabaseCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]SQLShimDatabase
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[SQLShimDatabase](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled, created FROM (SELECT id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled, created FROM sqlshim_databases WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []SQLShimDatabase
	if err := s.db.SelectContextWithDirty(ctx, "ListSQLShimDatabasesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, SQLShimDatabase{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveSQLShimDatabase saves a SQLShimDatabase
func (s *Storage) SaveSQLShimDatabase(ctx context.Context, obj *SQLShimDatabase) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, SQLShimDatabaseKeyID, nil, func(i *SQLShimDatabase) error {
		return ucerr.Wrap(s.saveInnerSQLShimDatabase(ctx, obj))
	}))
}

// SaveSQLShimDatabase saves a SQLShimDatabase
func (s *Storage) saveInnerSQLShimDatabase(ctx context.Context, obj *SQLShimDatabase) error {
	const q = "INSERT INTO sqlshim_databases (id, updated, deleted, name, type, host, port, username, password, schemas, schemas_updated, schemas_update_scheduled) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, type = $4, host = $5, port = $6, username = $7, password = $8, schemas = $9, schemas_updated = $10, schemas_update_scheduled = $11 WHERE (sqlshim_databases.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveSQLShimDatabase", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Type, obj.Host, obj.Port, obj.Username, obj.Password, obj.Schemas, obj.SchemasUpdated, obj.SchemasUpdateScheduled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "SQLShimDatabase %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteSQLShimDatabase soft-deletes a SQLShimDatabase which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteSQLShimDatabase(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerSQLShimDatabase(ctx, objID, false))
}

// deleteInnerSQLShimDatabase soft-deletes a SQLShimDatabase which is currently alive
func (s *Storage) deleteInnerSQLShimDatabase(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[SQLShimDatabase](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SQLShimDatabaseKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := SQLShimDatabase{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[SQLShimDatabase](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE sqlshim_databases SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteSQLShimDatabase", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting SQLShimDatabase %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "SQLShimDatabase %v not found", objID)
	}
	return nil
}

// FlushCacheForSQLShimDatabase flushes cache for SQLShimDatabase. It may flush a larger scope then
func (s *Storage) FlushCacheForSQLShimDatabase(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "SQLShimDatabase"))
}
