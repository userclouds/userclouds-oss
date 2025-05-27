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

// IsAccessorSearchIndexSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsAccessorSearchIndexSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *AccessorSearchIndex
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[AccessorSearchIndex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessorSearchIndexKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsAccessorSearchIndexSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsAccessorSearchIndexSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM accessor_search_indices WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsAccessorSearchIndexSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetAccessorSearchIndex loads a AccessorSearchIndex by ID
func (s *Storage) GetAccessorSearchIndex(ctx context.Context, id uuid.UUID) (*AccessorSearchIndex, error) {
	return cache.ServerGetItem(ctx, s.cm, id, AccessorSearchIndexKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *AccessorSearchIndex) error {
			const q = "SELECT id, updated, deleted, user_search_index_id, query_type, created FROM accessor_search_indices WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetAccessorSearchIndex", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "AccessorSearchIndex %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetAccessorSearchIndexSoftDeleted loads a AccessorSearchIndex by ID iff it's soft-deleted
func (s *Storage) GetAccessorSearchIndexSoftDeleted(ctx context.Context, id uuid.UUID) (*AccessorSearchIndex, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *AccessorSearchIndex
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[AccessorSearchIndex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessorSearchIndexKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetAccessorSearchIndexSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetAccessorSearchIndexSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted AccessorSearchIndex %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, user_search_index_id, query_type, created FROM accessor_search_indices WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj AccessorSearchIndex
	if err := s.db.GetContextWithDirty(ctx, "GetAccessorSearchIndexSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted AccessorSearchIndex %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetAccessorSearchIndexsForIDs loads multiple AccessorSearchIndex for a given list of IDs
func (s *Storage) GetAccessorSearchIndexsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]AccessorSearchIndex, error) {
	items := make([]AccessorSearchIndex, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(AccessorSearchIndexKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[AccessorSearchIndex](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getAccessorSearchIndexesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetAccessorSearchIndexMap: returning %d AccessorSearchIndex. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getAccessorSearchIndexesHelperForIDs loads multiple AccessorSearchIndex for a given list of IDs from the DB
func (s *Storage) getAccessorSearchIndexesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]AccessorSearchIndex, error) {
	const q = "SELECT id, updated, deleted, user_search_index_id, query_type, created FROM accessor_search_indices WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []AccessorSearchIndex
	if err := s.db.SelectContextWithDirty(ctx, "GetAccessorSearchIndexesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested AccessorSearchIndexes  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListAccessorSearchIndexesPaginated loads a paginated list of AccessorSearchIndexes for the specified paginator settings
func (s *Storage) ListAccessorSearchIndexesPaginated(ctx context.Context, p pagination.Paginator) ([]AccessorSearchIndex, *pagination.ResponseFields, error) {
	return s.listInnerAccessorSearchIndexesPaginated(ctx, p, false)
}

// listInnerAccessorSearchIndexesPaginated loads a paginated list of AccessorSearchIndexes for the specified paginator settings
func (s *Storage) listInnerAccessorSearchIndexesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]AccessorSearchIndex, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(AccessorSearchIndexCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]AccessorSearchIndex
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[AccessorSearchIndex](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, user_search_index_id, query_type, created FROM (SELECT id, updated, deleted, user_search_index_id, query_type, created FROM accessor_search_indices WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []AccessorSearchIndex
	if err := s.db.SelectContextWithDirty(ctx, "ListAccessorSearchIndexesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, AccessorSearchIndex{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListAccessorSearchIndexesNonPaginated loads a AccessorSearchIndex up to a limit of 10 pages
func (s *Storage) ListAccessorSearchIndexesNonPaginated(ctx context.Context) ([]AccessorSearchIndex, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]AccessorSearchIndex
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[AccessorSearchIndex](ctx, *s.cm, s.cm.N.GetKeyNameStatic(AccessorSearchIndexCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewAccessorSearchIndexPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]AccessorSearchIndex, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListAccessorSearchIndexesPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListAccessorSearchIndexesNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(AccessorSearchIndexCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, AccessorSearchIndex{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveAccessorSearchIndex saves a AccessorSearchIndex
func (s *Storage) SaveAccessorSearchIndex(ctx context.Context, obj *AccessorSearchIndex) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, AccessorSearchIndexKeyID, nil, func(i *AccessorSearchIndex) error {
		return ucerr.Wrap(s.saveInnerAccessorSearchIndex(ctx, obj))
	}))
}

// SaveAccessorSearchIndex saves a AccessorSearchIndex
func (s *Storage) saveInnerAccessorSearchIndex(ctx context.Context, obj *AccessorSearchIndex) error {
	const q = "INSERT INTO accessor_search_indices (id, updated, deleted, user_search_index_id, query_type) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, user_search_index_id = $3, query_type = $4 WHERE (accessor_search_indices.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveAccessorSearchIndex", obj, q, obj.ID, obj.Deleted, obj.UserSearchIndexID, obj.QueryType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "AccessorSearchIndex %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteAccessorSearchIndex soft-deletes a AccessorSearchIndex which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteAccessorSearchIndex(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerAccessorSearchIndex(ctx, objID, false))
}

// deleteInnerAccessorSearchIndex soft-deletes a AccessorSearchIndex which is currently alive
func (s *Storage) deleteInnerAccessorSearchIndex(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[AccessorSearchIndex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessorSearchIndexKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := AccessorSearchIndex{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[AccessorSearchIndex](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE accessor_search_indices SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteAccessorSearchIndex", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting AccessorSearchIndex %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "AccessorSearchIndex %v not found", objID)
	}
	return nil
}

// FlushCacheForAccessorSearchIndex flushes cache for AccessorSearchIndex. It may flush a larger scope then
func (s *Storage) FlushCacheForAccessorSearchIndex(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "AccessorSearchIndex"))
}
