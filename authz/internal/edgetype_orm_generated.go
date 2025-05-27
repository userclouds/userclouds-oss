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

// IsEdgeTypeSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsEdgeTypeSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.EdgeType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.EdgeType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeTypeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsEdgeTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsEdgeTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM edge_types WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsEdgeTypeSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetEdgeType loads a EdgeType by ID
func (s *Storage) GetEdgeType(ctx context.Context, id uuid.UUID) (*authz.EdgeType, error) {
	return cache.ServerGetItem(ctx, s.cm, id, authz.EdgeTypeKeyID, authz.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *authz.EdgeType) error {
			const q = "SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM edge_types WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetEdgeType", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "EdgeType %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getEdgeTypeByColumns loads a EdgeType using the provided column names and values as a WHERE clause
func (s *Storage) getEdgeTypeByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*authz.EdgeType, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *authz.EdgeType
		mkey := s.cm.N.GetKeyNameStatic(authz.EdgeTypeCollectionKeyID)
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.EdgeType](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getEdgeTypeByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getEdgeTypeByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM edge_types WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj authz.EdgeType
	if err := s.db.GetContextWithDirty(ctx, "GetEdgeTypeForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "EdgeType %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetEdgeType(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for EdgeType with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving EdgeType with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetEdgeTypeSoftDeleted loads a EdgeType by ID iff it's soft-deleted
func (s *Storage) GetEdgeTypeSoftDeleted(ctx context.Context, id uuid.UUID) (*authz.EdgeType, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.EdgeType
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.EdgeType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeTypeKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetEdgeTypeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetEdgeTypeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted EdgeType %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM edge_types WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj authz.EdgeType
	if err := s.db.GetContextWithDirty(ctx, "GetEdgeTypeSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted EdgeType %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetEdgeTypesForIDs loads multiple EdgeType for a given list of IDs
func (s *Storage) GetEdgeTypesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]authz.EdgeType, error) {
	items := make([]authz.EdgeType, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(authz.EdgeTypeKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[authz.EdgeType](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getEdgeTypesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetEdgeTypeMap: returning %d EdgeType. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getEdgeTypesHelperForIDs loads multiple EdgeType for a given list of IDs from the DB
func (s *Storage) getEdgeTypesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]authz.EdgeType, error) {
	const q = "SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM edge_types WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []authz.EdgeType
	if err := s.db.SelectContextWithDirty(ctx, "GetEdgeTypesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested EdgeTypes  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListEdgeTypesPaginated loads a paginated list of EdgeTypes for the specified paginator settings
func (s *Storage) ListEdgeTypesPaginated(ctx context.Context, p pagination.Paginator) ([]authz.EdgeType, *pagination.ResponseFields, error) {
	return s.listInnerEdgeTypesPaginated(ctx, p, false)
}

// listInnerEdgeTypesPaginated loads a paginated list of EdgeTypes for the specified paginator settings
func (s *Storage) listInnerEdgeTypesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]authz.EdgeType, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(authz.EdgeTypeCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]authz.EdgeType
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[authz.EdgeType](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM (SELECT id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id, created FROM edge_types WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []authz.EdgeType
	if err := s.db.SelectContextWithDirty(ctx, "ListEdgeTypesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, authz.EdgeType{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveEdgeType saves a EdgeType
func (s *Storage) SaveEdgeType(ctx context.Context, obj *authz.EdgeType) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveEdgeType(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.EdgeTypeKeyID, nil, func(i *authz.EdgeType) error {
		return ucerr.Wrap(s.saveInnerEdgeType(ctx, obj))
	}))
}

// SaveEdgeType saves a EdgeType
func (s *Storage) saveInnerEdgeType(ctx context.Context, obj *authz.EdgeType) error {
	const q = "INSERT INTO edge_types (id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type_name = $3, source_object_type_id = $4, target_object_type_id = $5, attributes = $6, organization_id = $7 WHERE (edge_types.source_object_type_id = $4 AND edge_types.target_object_type_id = $5 AND edge_types.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveEdgeType", obj, q, obj.ID, obj.Deleted, obj.TypeName, obj.SourceObjectTypeID, obj.TargetObjectTypeID, obj.Attributes, obj.OrganizationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "EdgeType %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// InsertEdgeType inserts a EdgeType without resolving conflict with existing rows
func (s *Storage) InsertEdgeType(ctx context.Context, obj *authz.EdgeType) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveEdgeType(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.EdgeTypeKeyID, nil, func(i *authz.EdgeType) error {
		return ucerr.Wrap(s.insertInnerEdgeType(ctx, obj))
	}))
}

// insertInnerEdgeType inserts a EdgeType without resolving conflict with existing rows
func (s *Storage) insertInnerEdgeType(ctx context.Context, obj *authz.EdgeType) error {
	const q = "INSERT INTO edge_types (id, updated, deleted, type_name, source_object_type_id, target_object_type_id, attributes, organization_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "InsertEdgeType", obj, q, obj.ID, obj.Deleted, obj.TypeName, obj.SourceObjectTypeID, obj.TargetObjectTypeID, obj.Attributes, obj.OrganizationID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteEdgeType soft-deletes a EdgeType which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteEdgeType(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerEdgeType(ctx, objID, false))
}

// deleteInnerEdgeType soft-deletes a EdgeType which is currently alive
func (s *Storage) deleteInnerEdgeType(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteEdgeType(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[authz.EdgeType](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.EdgeTypeKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := authz.EdgeType{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[authz.EdgeType](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE edge_types SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteEdgeType", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting EdgeType %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "EdgeType %v not found", objID)
	}
	return nil
}

// FlushCacheForEdgeType flushes cache for EdgeType. It may flush a larger scope then
func (s *Storage) FlushCacheForEdgeType(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "EdgeType"))
}
