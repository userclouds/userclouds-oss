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

// IsColumnSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsColumnSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Column
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Column](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsColumnSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsColumnSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM columns WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsColumnSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetColumn loads a Column by ID
func (s *Storage) GetColumn(ctx context.Context, id uuid.UUID) (*Column, error) {
	return cache.ServerGetItem(ctx, s.cm, id, ColumnKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *Column) error {
			const q = "SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM columns WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetColumn", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Column %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getColumnByColumns loads a Column using the provided column names and values as a WHERE clause
func (s *Storage) getColumnByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*Column, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *Column
		mkey := s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID)
		// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
		if s.cm.N.GetKeyNameWithString(IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID))) != "" {
			mkey = s.cm.N.GetKeyNameWithString(IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID)))
		}
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Column](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getColumnByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getColumnByColumns] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	args := ""
	for i := range columnNames {
		if i > 0 {
			args += " AND "
		}
		args += fmt.Sprintf("%s=$%d", columnNames[i], i+1)
	}
	q := fmt.Sprintf("SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM columns WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj Column
	if err := s.db.GetContextWithDirty(ctx, "GetColumnForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Column %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetColumn(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for Column with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving Column with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetColumnSoftDeleted loads a Column by ID iff it's soft-deleted
func (s *Storage) GetColumnSoftDeleted(ctx context.Context, id uuid.UUID) (*Column, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Column
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Column](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetColumnSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetColumnSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Column %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM columns WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Column
	if err := s.db.GetContextWithDirty(ctx, "GetColumnSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Column %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetColumnsForIDs loads multiple Column for a given list of IDs
func (s *Storage) GetColumnsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Column, error) {
	items := make([]Column, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(ColumnKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[Column](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getColumnsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetColumnMap: returning %d Column. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getColumnsHelperForIDs loads multiple Column for a given list of IDs from the DB
func (s *Storage) getColumnsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Column, error) {
	const q = "SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM columns WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Column
	if err := s.db.SelectContextWithDirty(ctx, "GetColumnsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Columns  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListColumnsPaginated loads a paginated list of Columns for the specified paginator settings
func (s *Storage) ListColumnsPaginated(ctx context.Context, p pagination.Paginator) ([]Column, *pagination.ResponseFields, error) {
	return s.listInnerColumnsPaginated(ctx, p, false)
}

// listInnerColumnsPaginated loads a paginated list of Columns for the specified paginator settings
func (s *Storage) listInnerColumnsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Column, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable()
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID)
		ckey = s.cm.N.GetKeyName(ColumnCollectionPageKeyID, []string{string(p.GetCursor()), fmt.Sprintf("%v", p.GetLimit())})
		var err error
		var v *[]Column
		partialHit := false
		// We only try to fetch the page of data if we could use it
		if cachable && !forceDBRead {
			v, _, sentinel, partialHit, err = cache.GetItemsArrayFromCache[Column](ctx, *s.cm, ckey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		// If the page is not in the cache or if request is not cachable, we need to check the global collection cache to see if we can use follower reads
		if v == nil || !cachable {
			mkey := lkey
			// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
			if s.cm.N.GetKeyNameWithString(IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID))) != "" {
				mkey = s.cm.N.GetKeyNameWithString(IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(ColumnCollectionKeyID)))
			}
			_, conflict, _, _, err = cache.GetItemsArrayFromCache[Column](ctx, *s.cm, mkey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		if cachable {
			if v != nil {
				if partialHit {
					uclog.Verbosef(ctx, "Partial cache hit for Column launching async refresh")
					go func(ctx context.Context) {
						if _, _, err := s.listInnerColumnsPaginated(ctx, p, true); err != nil { // lint: ucpagination-safe
							uclog.Errorf(ctx, "Error fetching Column async for cache update: %v", err)
						}
					}(context.WithoutCancel(ctx))
				}

				v, respFields := pagination.ProcessResults(*v, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
				return v, &respFields, nil
			}
			sentinel, err = cache.TakeGlobalCollectionLock(ctx, cache.Read, *s.cm, Column{})
			if err != nil {
				uclog.Errorf(ctx, "Error taking global collection lock for Columns: %v", err)
			} else if sentinel != cache.NoLockSentinel {
				defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{lkey}, Column{}, sentinel)
			}
		}
	}
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM (SELECT id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed, created FROM columns WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Column
	if err := s.db.SelectContextWithDirty(ctx, "ListColumnsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
	if s.cm != nil && cachable {
		cache.SaveItemsToCollection(ctx, *s.cm, Column{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveColumn saves a Column
func (s *Storage) SaveColumn(ctx context.Context, obj *Column) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, ColumnKeyID, nil, func(i *Column) error {
		return ucerr.Wrap(s.saveInnerColumn(ctx, obj))
	}))
}

// SaveColumn saves a Column
func (s *Storage) saveInnerColumn(ctx context.Context, obj *Column) error {
	const q = "INSERT INTO columns (id, updated, deleted, name, tbl, sqlshim_database_id, data_type_id, is_array, default_value, index_type, attributes, access_policy_id, default_transformer_id, default_token_access_policy_id, search_indexed) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, tbl = $4, sqlshim_database_id = $5, data_type_id = $6, is_array = $7, default_value = $8, index_type = $9, attributes = $10, access_policy_id = $11, default_transformer_id = $12, default_token_access_policy_id = $13, search_indexed = $14 WHERE (columns.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveColumn", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Table, obj.SQLShimDatabaseID, obj.DataTypeID, obj.IsArray, obj.DefaultValue, obj.IndexType, obj.Attributes, obj.AccessPolicyID, obj.DefaultTransformerID, obj.DefaultTokenAccessPolicyID, obj.SearchIndexed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Column %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteColumn soft-deletes a Column which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteColumn(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerColumn(ctx, objID, false))
}

// deleteInnerColumn soft-deletes a Column which is currently alive
func (s *Storage) deleteInnerColumn(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Column](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ColumnKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Column{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Column](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE columns SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteColumn", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Column %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Column %v not found", objID)
	}
	return nil
}

// FlushCacheForColumn flushes cache for Column. It may flush a larger scope then
func (s *Storage) FlushCacheForColumn(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Column"))
}
