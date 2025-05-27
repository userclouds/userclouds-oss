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

	"userclouds.com/idp/datamapping"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

// IsDataSourceSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsDataSourceSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *datamapping.DataSource
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[datamapping.DataSource](ctx, *s.cm, s.cm.N.GetKeyNameWithID(datamapping.DataSourceKeyID, id), s.cm.N.GetKeyNameWithID(datamapping.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsDataSourceSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsDataSourceSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM data_sources WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsDataSourceSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetDataSource loads a DataSource by ID
func (s *Storage) GetDataSource(ctx context.Context, id uuid.UUID) (*datamapping.DataSource, error) {
	return cache.ServerGetItem(ctx, s.cm, id, datamapping.DataSourceKeyID, datamapping.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *datamapping.DataSource) error {
			const q = "SELECT id, updated, deleted, name, type, config, metadata, created FROM data_sources WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetDataSource", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "DataSource %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetDataSourceSoftDeleted loads a DataSource by ID iff it's soft-deleted
func (s *Storage) GetDataSourceSoftDeleted(ctx context.Context, id uuid.UUID) (*datamapping.DataSource, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *datamapping.DataSource
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[datamapping.DataSource](ctx, *s.cm, s.cm.N.GetKeyNameWithID(datamapping.DataSourceKeyID, id), s.cm.N.GetKeyNameWithID(datamapping.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetDataSourceSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetDataSourceSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted DataSource %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, type, config, metadata, created FROM data_sources WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj datamapping.DataSource
	if err := s.db.GetContextWithDirty(ctx, "GetDataSourceSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted DataSource %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetDataSourcesForIDs loads multiple DataSource for a given list of IDs
func (s *Storage) GetDataSourcesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]datamapping.DataSource, error) {
	items := make([]datamapping.DataSource, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(datamapping.DataSourceKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(datamapping.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[datamapping.DataSource](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getDataSourcesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetDataSourceMap: returning %d DataSource. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getDataSourcesHelperForIDs loads multiple DataSource for a given list of IDs from the DB
func (s *Storage) getDataSourcesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]datamapping.DataSource, error) {
	const q = "SELECT id, updated, deleted, name, type, config, metadata, created FROM data_sources WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []datamapping.DataSource
	if err := s.db.SelectContextWithDirty(ctx, "GetDataSourcesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested DataSources  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListDataSourcesPaginated loads a paginated list of DataSources for the specified paginator settings
func (s *Storage) ListDataSourcesPaginated(ctx context.Context, p pagination.Paginator) ([]datamapping.DataSource, *pagination.ResponseFields, error) {
	return s.listInnerDataSourcesPaginated(ctx, p, false)
}

// listInnerDataSourcesPaginated loads a paginated list of DataSources for the specified paginator settings
func (s *Storage) listInnerDataSourcesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]datamapping.DataSource, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(datamapping.DataSourceCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]datamapping.DataSource
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[datamapping.DataSource](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, type, config, metadata, created FROM (SELECT id, updated, deleted, name, type, config, metadata, created FROM data_sources WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []datamapping.DataSource
	if err := s.db.SelectContextWithDirty(ctx, "ListDataSourcesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, datamapping.DataSource{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveDataSource saves a DataSource
func (s *Storage) SaveDataSource(ctx context.Context, obj *datamapping.DataSource) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, datamapping.DataSourceKeyID, nil, func(i *datamapping.DataSource) error {
		return ucerr.Wrap(s.saveInnerDataSource(ctx, obj))
	}))
}

// SaveDataSource saves a DataSource
func (s *Storage) saveInnerDataSource(ctx context.Context, obj *datamapping.DataSource) error {
	const q = "INSERT INTO data_sources (id, updated, deleted, name, type, config, metadata) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, type = $4, config = $5, metadata = $6 WHERE (data_sources.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveDataSource", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Type, obj.Config, obj.Metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "DataSource %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteDataSource soft-deletes a DataSource which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteDataSource(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerDataSource(ctx, objID, false))
}

// deleteInnerDataSource soft-deletes a DataSource which is currently alive
func (s *Storage) deleteInnerDataSource(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[datamapping.DataSource](ctx, *s.cm, s.cm.N.GetKeyNameWithID(datamapping.DataSourceKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := datamapping.DataSource{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[datamapping.DataSource](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE data_sources SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteDataSource", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting DataSource %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "DataSource %v not found", objID)
	}
	return nil
}

// FlushCacheForDataSource flushes cache for DataSource. It may flush a larger scope then
func (s *Storage) FlushCacheForDataSource(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "DataSource"))
}
