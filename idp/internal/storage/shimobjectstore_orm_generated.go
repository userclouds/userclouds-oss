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

// IsShimObjectStoreSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsShimObjectStoreSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *ShimObjectStore
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[ShimObjectStore](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ShimObjectStoreKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsShimObjectStoreSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsShimObjectStoreSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM shim_object_stores WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsShimObjectStoreSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetShimObjectStore loads a ShimObjectStore by ID
func (s *Storage) GetShimObjectStore(ctx context.Context, id uuid.UUID) (*ShimObjectStore, error) {
	return cache.ServerGetItem(ctx, s.cm, id, ShimObjectStoreKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *ShimObjectStore) error {
			const q = "SELECT id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id, created FROM shim_object_stores WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetShimObjectStore", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "ShimObjectStore %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetShimObjectStoreSoftDeleted loads a ShimObjectStore by ID iff it's soft-deleted
func (s *Storage) GetShimObjectStoreSoftDeleted(ctx context.Context, id uuid.UUID) (*ShimObjectStore, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *ShimObjectStore
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[ShimObjectStore](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ShimObjectStoreKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetShimObjectStoreSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetShimObjectStoreSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted ShimObjectStore %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id, created FROM shim_object_stores WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj ShimObjectStore
	if err := s.db.GetContextWithDirty(ctx, "GetShimObjectStoreSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted ShimObjectStore %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetShimObjectStoresForIDs loads multiple ShimObjectStore for a given list of IDs
func (s *Storage) GetShimObjectStoresForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]ShimObjectStore, error) {
	items := make([]ShimObjectStore, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(ShimObjectStoreKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[ShimObjectStore](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getShimObjectStoresHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetShimObjectStoreMap: returning %d ShimObjectStore. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getShimObjectStoresHelperForIDs loads multiple ShimObjectStore for a given list of IDs from the DB
func (s *Storage) getShimObjectStoresHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]ShimObjectStore, error) {
	const q = "SELECT id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id, created FROM shim_object_stores WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []ShimObjectStore
	if err := s.db.SelectContextWithDirty(ctx, "GetShimObjectStoresForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested ShimObjectStores  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListShimObjectStoresPaginated loads a paginated list of ShimObjectStores for the specified paginator settings
func (s *Storage) ListShimObjectStoresPaginated(ctx context.Context, p pagination.Paginator) ([]ShimObjectStore, *pagination.ResponseFields, error) {
	return s.listInnerShimObjectStoresPaginated(ctx, p, false)
}

// listInnerShimObjectStoresPaginated loads a paginated list of ShimObjectStores for the specified paginator settings
func (s *Storage) listInnerShimObjectStoresPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]ShimObjectStore, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(ShimObjectStoreCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]ShimObjectStore
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[ShimObjectStore](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id, created FROM (SELECT id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id, created FROM shim_object_stores WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []ShimObjectStore
	if err := s.db.SelectContextWithDirty(ctx, "ListShimObjectStoresPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, ShimObjectStore{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveShimObjectStore saves a ShimObjectStore
func (s *Storage) SaveShimObjectStore(ctx context.Context, obj *ShimObjectStore) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, ShimObjectStoreKeyID, nil, func(i *ShimObjectStore) error {
		return ucerr.Wrap(s.saveInnerShimObjectStore(ctx, obj))
	}))
}

// SaveShimObjectStore saves a ShimObjectStore
func (s *Storage) saveInnerShimObjectStore(ctx context.Context, obj *ShimObjectStore) error {
	const q = "INSERT INTO shim_object_stores (id, updated, deleted, name, type, region, access_key_id, secret_access_key, role_arn, access_policy_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, type = $4, region = $5, access_key_id = $6, secret_access_key = $7, role_arn = $8, access_policy_id = $9 WHERE (shim_object_stores.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveShimObjectStore", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Type, obj.Region, obj.AccessKeyID, obj.SecretAccessKey, obj.RoleARN, obj.AccessPolicyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "ShimObjectStore %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteShimObjectStore soft-deletes a ShimObjectStore which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteShimObjectStore(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerShimObjectStore(ctx, objID, false))
}

// deleteInnerShimObjectStore soft-deletes a ShimObjectStore which is currently alive
func (s *Storage) deleteInnerShimObjectStore(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[ShimObjectStore](ctx, *s.cm, s.cm.N.GetKeyNameWithID(ShimObjectStoreKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := ShimObjectStore{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[ShimObjectStore](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE shim_object_stores SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteShimObjectStore", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting ShimObjectStore %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "ShimObjectStore %v not found", objID)
	}
	return nil
}

// FlushCacheForShimObjectStore flushes cache for ShimObjectStore. It may flush a larger scope then
func (s *Storage) FlushCacheForShimObjectStore(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "ShimObjectStore"))
}
