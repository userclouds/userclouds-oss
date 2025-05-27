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

// IsObjectSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsObjectSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Object
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Object](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsObjectSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsObjectSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM objects WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsObjectSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetObject loads a Object by ID
func (s *Storage) GetObject(ctx context.Context, id uuid.UUID) (*authz.Object, error) {
	return cache.ServerGetItem(ctx, s.cm, id, authz.ObjectKeyID, authz.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *authz.Object) error {
			const q = "SELECT id, updated, deleted, alias, type_id, organization_id, created FROM objects WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetObject", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Object %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getObjectByColumns loads a Object using the provided column names and values as a WHERE clause
func (s *Storage) getObjectByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*authz.Object, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *authz.Object
		mkey := s.cm.N.GetKeyNameStatic(authz.ObjectCollectionKeyID)
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Object](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getObjectByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getObjectByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, alias, type_id, organization_id, created FROM objects WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj authz.Object
	if err := s.db.GetContextWithDirty(ctx, "GetObjectForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Object %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetObject(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for Object with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving Object with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetObjectSoftDeleted loads a Object by ID iff it's soft-deleted
func (s *Storage) GetObjectSoftDeleted(ctx context.Context, id uuid.UUID) (*authz.Object, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Object
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Object](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetObjectSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetObjectSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Object %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, alias, type_id, organization_id, created FROM objects WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj authz.Object
	if err := s.db.GetContextWithDirty(ctx, "GetObjectSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Object %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetObjectsForIDs loads multiple Object for a given list of IDs
func (s *Storage) GetObjectsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Object, error) {
	items := make([]authz.Object, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(authz.ObjectKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[authz.Object](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getObjectsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetObjectMap: returning %d Object. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getObjectsHelperForIDs loads multiple Object for a given list of IDs from the DB
func (s *Storage) getObjectsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Object, error) {
	const q = "SELECT id, updated, deleted, alias, type_id, organization_id, created FROM objects WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []authz.Object
	if err := s.db.SelectContextWithDirty(ctx, "GetObjectsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Objects  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListObjectsPaginated loads a paginated list of Objects for the specified paginator settings
func (s *Storage) ListObjectsPaginated(ctx context.Context, p pagination.Paginator) ([]authz.Object, *pagination.ResponseFields, error) {
	return s.listInnerObjectsPaginated(ctx, p, false)
}

// listInnerObjectsPaginated loads a paginated list of Objects for the specified paginator settings
func (s *Storage) listInnerObjectsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]authz.Object, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(authz.ObjectCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]authz.Object
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[authz.Object](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, alias, type_id, organization_id, created FROM (SELECT id, updated, deleted, alias, type_id, organization_id, created FROM objects WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []authz.Object
	if err := s.db.SelectContextWithDirty(ctx, "ListObjectsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, authz.Object{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveObject saves a Object
func (s *Storage) SaveObject(ctx context.Context, obj *authz.Object) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveObject(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.ObjectKeyID, nil, func(i *authz.Object) error {
		return ucerr.Wrap(s.saveInnerObject(ctx, obj))
	}))
}

// SaveObject saves a Object
func (s *Storage) saveInnerObject(ctx context.Context, obj *authz.Object) error {
	const q = "INSERT INTO objects (id, updated, deleted, alias, type_id, organization_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, alias = $3, type_id = $4, organization_id = $5 WHERE (objects.type_id = $4 AND objects.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveObject", obj, q, obj.ID, obj.Deleted, obj.Alias, obj.TypeID, obj.OrganizationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Object %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// InsertObject inserts a Object without resolving conflict with existing rows
func (s *Storage) InsertObject(ctx context.Context, obj *authz.Object) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveObject(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.ObjectKeyID, nil, func(i *authz.Object) error {
		return ucerr.Wrap(s.insertInnerObject(ctx, obj))
	}))
}

// insertInnerObject inserts a Object without resolving conflict with existing rows
func (s *Storage) insertInnerObject(ctx context.Context, obj *authz.Object) error {
	const q = "INSERT INTO objects (id, updated, deleted, alias, type_id, organization_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "InsertObject", obj, q, obj.ID, obj.Deleted, obj.Alias, obj.TypeID, obj.OrganizationID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteObject soft-deletes a Object which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteObject(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerObject(ctx, objID, false))
}

// deleteInnerObject soft-deletes a Object which is currently alive
func (s *Storage) deleteInnerObject(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteObject(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[authz.Object](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjectKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := authz.Object{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[authz.Object](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE objects SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteObject", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Object %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Object %v not found", objID)
	}
	return nil
}

// FlushCacheForObject flushes cache for Object. It may flush a larger scope then
func (s *Storage) FlushCacheForObject(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Object"))
}
