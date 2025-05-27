// NOTE: automatically generated file -- DO NOT EDIT

package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/authz"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

// IsObjectTypeSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsObjectTypeSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.ObjectType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.ObjectType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectTypeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsObjectTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsObjectTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM object_types WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsObjectTypeSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetObjectType loads a ObjectType by ID
func (s *Storage) GetObjectType(ctx context.Context, id uuid.UUID) (*authz.ObjectType, error) {
	return cache.ServerGetItem(ctx, s.cm, id, authz.ObjectTypeKeyID, authz.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *authz.ObjectType) error {
			const q = "SELECT id, updated, deleted, type_name, created FROM object_types WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetObjectType", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "ObjectType %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getObjectTypeByColumns loads a ObjectType using the provided column names and values as a WHERE clause
func (s *Storage) getObjectTypeByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*authz.ObjectType, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *authz.ObjectType
		mkey := s.cm.N.GetKeyNameStatic(authz.ObjectTypeCollectionKeyID)
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.ObjectType](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getObjectTypeByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getObjectTypeByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, type_name, created FROM object_types WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj authz.ObjectType
	if err := s.db.GetContextWithDirty(ctx, "GetObjectTypeForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "ObjectType %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetObjectType(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for ObjectType with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving ObjectType with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetObjectTypeSoftDeleted loads a ObjectType by ID iff it's soft-deleted
func (s *Storage) GetObjectTypeSoftDeleted(ctx context.Context, id uuid.UUID) (*authz.ObjectType, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.ObjectType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.ObjectType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectTypeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetObjectTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetObjectTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted ObjectType %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, type_name, created FROM object_types WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj authz.ObjectType
	if err := s.db.GetContextWithDirty(ctx, "GetObjectTypeSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted ObjectType %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetObjectTypesForIDs loads multiple ObjectType for a given list of IDs
func (s *Storage) GetObjectTypesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]authz.ObjectType, error) {
	items := make([]authz.ObjectType, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(authz.ObjectTypeKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[authz.ObjectType](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getObjectTypesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetObjectTypeMap: returning %d ObjectType. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getObjectTypesHelperForIDs loads multiple ObjectType for a given list of IDs from the DB
func (s *Storage) getObjectTypesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]authz.ObjectType, error) {
	const q = "SELECT id, updated, deleted, type_name, created FROM object_types WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []authz.ObjectType
	if err := s.db.SelectContextWithDirty(ctx, "GetObjectTypesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested ObjectTypes  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListObjectTypesPaginated loads a paginated list of ObjectTypes for the specified paginator settings
func (s *Storage) ListObjectTypesPaginated(ctx context.Context, p pagination.Paginator) ([]authz.ObjectType, *pagination.ResponseFields, error) {
	return s.listInnerObjectTypesPaginated(ctx, p, false)
}

// listInnerObjectTypesPaginated loads a paginated list of ObjectTypes for the specified paginator settings
func (s *Storage) listInnerObjectTypesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]authz.ObjectType, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(authz.ObjectTypeCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]authz.ObjectType
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[authz.ObjectType](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, type_name, created FROM (SELECT id, updated, deleted, type_name, created FROM object_types WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []authz.ObjectType
	if err := s.db.SelectContextWithDirty(ctx, "ListObjectTypesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, authz.ObjectType{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveObjectType saves a ObjectType
func (s *Storage) SaveObjectType(ctx context.Context, obj *authz.ObjectType) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.ObjectTypeKeyID, nil, func(i *authz.ObjectType) error {
		return ucerr.Wrap(s.saveInnerObjectType(ctx, obj))
	}))
}

// SaveObjectType saves a ObjectType
func (s *Storage) saveInnerObjectType(ctx context.Context, obj *authz.ObjectType) error {
	const q = "INSERT INTO object_types (id, updated, deleted, type_name) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type_name = $3 WHERE (object_types.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveObjectType", obj, q, obj.ID, obj.Deleted, obj.TypeName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "ObjectType %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// InsertObjectType inserts a ObjectType without resolving conflict with existing rows
func (s *Storage) InsertObjectType(ctx context.Context, obj *authz.ObjectType) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.ObjectTypeKeyID, nil, func(i *authz.ObjectType) error {
		return ucerr.Wrap(s.insertInnerObjectType(ctx, obj))
	}))
}

// insertInnerObjectType inserts a ObjectType without resolving conflict with existing rows
func (s *Storage) insertInnerObjectType(ctx context.Context, obj *authz.ObjectType) error {
	const q = "INSERT INTO object_types (id, updated, deleted, type_name) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "InsertObjectType", obj, q, obj.ID, obj.Deleted, obj.TypeName); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteObjectType soft-deletes a ObjectType which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteObjectType(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerObjectType(ctx, objID, false))
}

// deleteInnerObjectType soft-deletes a ObjectType which is currently alive
func (s *Storage) deleteInnerObjectType(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteObjectType(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[authz.ObjectType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectTypeKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := authz.ObjectType{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[authz.ObjectType](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE object_types SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteObjectType", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting ObjectType %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "ObjectType %v not found", objID)
	}
	return nil
}

// FlushCacheForObjectType flushes cache for ObjectType. It may flush a larger scope then
func (s *Storage) FlushCacheForObjectType(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "ObjectType"))
}
