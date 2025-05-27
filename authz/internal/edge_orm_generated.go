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

// IsEdgeSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsEdgeSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Edge
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Edge](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsEdgeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsEdgeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM edges WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsEdgeSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetEdge loads a Edge by ID
func (s *Storage) GetEdge(ctx context.Context, id uuid.UUID) (*authz.Edge, error) {
	return cache.ServerGetItem(ctx, s.cm, id, authz.EdgeKeyID, authz.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *authz.Edge) error {
			const q = "SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetEdge", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Edge %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getEdgeByColumns loads a Edge using the provided column names and values as a WHERE clause
func (s *Storage) getEdgeByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*authz.Edge, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *authz.Edge
		mkey := s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)
		// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
		if s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID))) != "" {
			mkey = s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)))
		}
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Edge](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getEdgeByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getEdgeByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj authz.Edge
	if err := s.db.GetContextWithDirty(ctx, "GetEdgeForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Edge %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetEdge(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for Edge with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving Edge with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetEdgeSoftDeleted loads a Edge by ID iff it's soft-deleted
func (s *Storage) GetEdgeSoftDeleted(ctx context.Context, id uuid.UUID) (*authz.Edge, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Edge
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Edge](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetEdgeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetEdgeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Edge %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj authz.Edge
	if err := s.db.GetContextWithDirty(ctx, "GetEdgeSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Edge %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetEdgesForIDs loads multiple Edge for a given list of IDs
func (s *Storage) GetEdgesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Edge, error) {
	items := make([]authz.Edge, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(authz.EdgeKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[authz.Edge](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getEdgesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetEdgeMap: returning %d Edge. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getEdgesHelperForIDs loads multiple Edge for a given list of IDs from the DB
func (s *Storage) getEdgesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Edge, error) {
	const q = "SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []authz.Edge
	if err := s.db.SelectContextWithDirty(ctx, "GetEdgesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Edges  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListEdgesPaginated loads a paginated list of Edges for the specified paginator settings
func (s *Storage) ListEdgesPaginated(ctx context.Context, p pagination.Paginator) ([]authz.Edge, *pagination.ResponseFields, error) {
	return s.listInnerEdgesPaginated(ctx, p, false)
}

// listInnerEdgesPaginated loads a paginated list of Edges for the specified paginator settings
func (s *Storage) listInnerEdgesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]authz.Edge, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable()
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)
		ckey = s.cm.N.GetKeyName(authz.EdgeCollectionPageKeyID, []string{string(p.GetCursor()), fmt.Sprintf("%v", p.GetLimit())})
		var err error
		var v *[]authz.Edge
		partialHit := false
		// We only try to fetch the page of data if we could use it
		if cachable && !forceDBRead {
			v, _, sentinel, partialHit, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, ckey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		// If the page is not in the cache or if request is not cachable, we need to check the global collection cache to see if we can use follower reads
		if v == nil || !cachable {
			mkey := lkey
			// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
			if s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID))) != "" {
				mkey = s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)))
			}
			_, conflict, _, _, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, mkey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		if cachable {
			if v != nil {
				if partialHit {
					uclog.Verbosef(ctx, "Partial cache hit for authz.Edge launching async refresh")
					go func(ctx context.Context) {
						if _, _, err := s.listInnerEdgesPaginated(ctx, p, true); err != nil { // lint: ucpagination-safe
							uclog.Errorf(ctx, "Error fetching authz.Edge async for cache update: %v", err)
						}
					}(context.WithoutCancel(ctx))
				}

				v, respFields := pagination.ProcessResults(*v, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
				return v, &respFields, nil
			}
			sentinel, err = cache.TakeGlobalCollectionLock(ctx, cache.Read, *s.cm, authz.Edge{})
			if err != nil {
				uclog.Errorf(ctx, "Error taking global collection lock for Edges: %v", err)
			} else if sentinel != cache.NoLockSentinel {
				defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{lkey}, authz.Edge{}, sentinel)
			}
		}
	}
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM (SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []authz.Edge
	if err := s.db.SelectContextWithDirty(ctx, "ListEdgesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, authz.Edge{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveEdge saves a Edge
func (s *Storage) SaveEdge(ctx context.Context, obj *authz.Edge) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveEdge(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.EdgeKeyID, s.additionalSaveKeysForEdge(obj), func(i *authz.Edge) error {
		return ucerr.Wrap(s.saveInnerEdge(ctx, obj))
	}))
}

// SaveEdge saves a Edge
func (s *Storage) saveInnerEdge(ctx context.Context, obj *authz.Edge) error {
	const q = "INSERT INTO edges (id, updated, deleted, edge_type_id, source_object_id, target_object_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, edge_type_id = $3, source_object_id = $4, target_object_id = $5 WHERE (edges.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveEdge", obj, q, obj.ID, obj.Deleted, obj.EdgeTypeID, obj.SourceObjectID, obj.TargetObjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Edge %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// InsertEdge inserts a Edge without resolving conflict with existing rows
func (s *Storage) InsertEdge(ctx context.Context, obj *authz.Edge) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveEdge(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.EdgeKeyID, s.additionalSaveKeysForEdge(obj), func(i *authz.Edge) error {
		return ucerr.Wrap(s.insertInnerEdge(ctx, obj))
	}))
}

// insertInnerEdge inserts a Edge without resolving conflict with existing rows
func (s *Storage) insertInnerEdge(ctx context.Context, obj *authz.Edge) error {
	const q = "INSERT INTO edges (id, updated, deleted, edge_type_id, source_object_id, target_object_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "InsertEdge", obj, q, obj.ID, obj.Deleted, obj.EdgeTypeID, obj.SourceObjectID, obj.TargetObjectID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteEdge soft-deletes a Edge which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteEdge(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerEdge(ctx, objID, false))
}

// deleteInnerEdge soft-deletes a Edge which is currently alive
func (s *Storage) deleteInnerEdge(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[authz.Edge](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := authz.Edge{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[authz.Edge](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE edges SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteEdge", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Edge %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Edge %v not found", objID)
	}
	return nil
}

// FlushCacheForEdge flushes cache for Edge. It may flush a larger scope then
func (s *Storage) FlushCacheForEdge(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Edge"))
}
