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

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

// IsDataTypeSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsDataTypeSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *column.DataType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[column.DataType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(column.DataTypeKeyID, id), s.cm.N.GetKeyNameWithID(column.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsDataTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsDataTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM data_types WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsDataTypeSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetDataType loads a DataType by ID
func (s *Storage) GetDataType(ctx context.Context, id uuid.UUID) (*column.DataType, error) {
	return cache.ServerGetItem(ctx, s.cm, id, column.DataTypeKeyID, column.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *column.DataType) error {
			const q = "SELECT id, updated, deleted, name, description, concrete_data_type_id, composite_attributes, created FROM data_types WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetDataType", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "DataType %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetDataTypeSoftDeleted loads a DataType by ID iff it's soft-deleted
func (s *Storage) GetDataTypeSoftDeleted(ctx context.Context, id uuid.UUID) (*column.DataType, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *column.DataType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[column.DataType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(column.DataTypeKeyID, id), s.cm.N.GetKeyNameWithID(column.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetDataTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetDataTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted DataType %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, description, concrete_data_type_id, composite_attributes, created FROM data_types WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj column.DataType
	if err := s.db.GetContextWithDirty(ctx, "GetDataTypeSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted DataType %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetDataTypesForIDs loads multiple DataType for a given list of IDs
func (s *Storage) GetDataTypesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]column.DataType, error) {
	items := make([]column.DataType, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(column.DataTypeKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(column.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[column.DataType](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getDataTypesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetDataTypeMap: returning %d DataType. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getDataTypesHelperForIDs loads multiple DataType for a given list of IDs from the DB
func (s *Storage) getDataTypesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]column.DataType, error) {
	const q = "SELECT id, updated, deleted, name, description, concrete_data_type_id, composite_attributes, created FROM data_types WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []column.DataType
	if err := s.db.SelectContextWithDirty(ctx, "GetDataTypesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested DataTypes  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListDataTypesPaginated loads a paginated list of DataTypes for the specified paginator settings
func (s *Storage) ListDataTypesPaginated(ctx context.Context, p pagination.Paginator) ([]column.DataType, *pagination.ResponseFields, error) {
	return s.listInnerDataTypesPaginated(ctx, p, false)
}

// listInnerDataTypesPaginated loads a paginated list of DataTypes for the specified paginator settings
func (s *Storage) listInnerDataTypesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]column.DataType, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(column.DataTypeCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]column.DataType
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[column.DataType](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, description, concrete_data_type_id, composite_attributes, created FROM (SELECT id, updated, deleted, name, description, concrete_data_type_id, composite_attributes, created FROM data_types WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []column.DataType
	if err := s.db.SelectContextWithDirty(ctx, "ListDataTypesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, column.DataType{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListDataTypesNonPaginated loads a DataType up to a limit of 10 pages
func (s *Storage) ListDataTypesNonPaginated(ctx context.Context) ([]column.DataType, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]column.DataType
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[column.DataType](ctx, *s.cm, s.cm.N.GetKeyNameStatic(column.DataTypeCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := column.NewDataTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]column.DataType, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListDataTypesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		objs = append(objs, objRead...)
		pageCount++
		if s.cm != nil {
			// Save individual items to cache under their own primary keys (optional)
			cache.SaveItemsFromCollectionToCache(ctx, *s.cm, objRead, sentinel)
		}
		if !pager.AdvanceCursor(*respFields) {
			break
		}
		if pageCount >= 10 {
			return nil, ucerr.Errorf("ListDataTypesNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(column.DataTypeCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, column.DataType{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveDataType saves a DataType
func (s *Storage) SaveDataType(ctx context.Context, obj *column.DataType) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, column.DataTypeKeyID, nil, func(i *column.DataType) error {
		return ucerr.Wrap(s.saveInnerDataType(ctx, obj))
	}))
}

// SaveDataType saves a DataType
func (s *Storage) saveInnerDataType(ctx context.Context, obj *column.DataType) error {
	const q = "INSERT INTO data_types (id, updated, deleted, name, description, concrete_data_type_id, composite_attributes) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, description = $4, concrete_data_type_id = $5, composite_attributes = $6 WHERE (data_types.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveDataType", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Description, obj.ConcreteDataTypeID, obj.CompositeAttributes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "DataType %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteDataType soft-deletes a DataType which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteDataType(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerDataType(ctx, objID, false))
}

// deleteInnerDataType soft-deletes a DataType which is currently alive
func (s *Storage) deleteInnerDataType(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[column.DataType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(column.DataTypeKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := column.DataType{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[column.DataType](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE data_types SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteDataType", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting DataType %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "DataType %v not found", objID)
	}
	return nil
}

// FlushCacheForDataType flushes cache for DataType. It may flush a larger scope then
func (s *Storage) FlushCacheForDataType(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "DataType"))
}
