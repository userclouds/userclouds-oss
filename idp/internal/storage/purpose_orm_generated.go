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

// IsPurposeSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsPurposeSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Purpose
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Purpose](ctx, *s.cm, s.cm.N.GetKeyNameWithID(PurposeKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsPurposeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsPurposeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM purposes WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsPurposeSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetPurpose loads a Purpose by ID
func (s *Storage) GetPurpose(ctx context.Context, id uuid.UUID) (*Purpose, error) {
	return cache.ServerGetItem(ctx, s.cm, id, PurposeKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *Purpose) error {
			const q = "SELECT id, updated, deleted, is_system, name, description, created FROM purposes WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetPurpose", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Purpose %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getPurposeByColumns loads a Purpose using the provided column names and values as a WHERE clause
func (s *Storage) getPurposeByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*Purpose, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *Purpose
		mkey := s.cm.N.GetKeyNameStatic(PurposeCollectionKeyID)
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Purpose](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getPurposeByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getPurposeByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, created FROM purposes WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj Purpose
	if err := s.db.GetContextWithDirty(ctx, "GetPurposeForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Purpose %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetPurpose(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for Purpose with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving Purpose with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetPurposeSoftDeleted loads a Purpose by ID iff it's soft-deleted
func (s *Storage) GetPurposeSoftDeleted(ctx context.Context, id uuid.UUID) (*Purpose, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Purpose
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Purpose](ctx, *s.cm, s.cm.N.GetKeyNameWithID(PurposeKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetPurposeSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetPurposeSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Purpose %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, is_system, name, description, created FROM purposes WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Purpose
	if err := s.db.GetContextWithDirty(ctx, "GetPurposeSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Purpose %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetPurposesForIDs loads multiple Purpose for a given list of IDs
func (s *Storage) GetPurposesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Purpose, error) {
	items := make([]Purpose, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(PurposeKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[Purpose](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getPurposesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetPurposeMap: returning %d Purpose. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getPurposesHelperForIDs loads multiple Purpose for a given list of IDs from the DB
func (s *Storage) getPurposesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Purpose, error) {
	const q = "SELECT id, updated, deleted, is_system, name, description, created FROM purposes WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Purpose
	if err := s.db.SelectContextWithDirty(ctx, "GetPurposesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Purposes  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListPurposesPaginated loads a paginated list of Purposes for the specified paginator settings
func (s *Storage) ListPurposesPaginated(ctx context.Context, p pagination.Paginator) ([]Purpose, *pagination.ResponseFields, error) {
	return s.listInnerPurposesPaginated(ctx, p, false)
}

// listInnerPurposesPaginated loads a paginated list of Purposes for the specified paginator settings
func (s *Storage) listInnerPurposesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Purpose, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(PurposeCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]Purpose
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[Purpose](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, created FROM (SELECT id, updated, deleted, is_system, name, description, created FROM purposes WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Purpose
	if err := s.db.SelectContextWithDirty(ctx, "ListPurposesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, Purpose{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListPurposesNonPaginated loads a Purpose up to a limit of 10 pages
func (s *Storage) ListPurposesNonPaginated(ctx context.Context) ([]Purpose, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]Purpose
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[Purpose](ctx, *s.cm, s.cm.N.GetKeyNameStatic(PurposeCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewPurposePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]Purpose, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListPurposesPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListPurposesNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(PurposeCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, Purpose{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SavePurpose saves a Purpose
func (s *Storage) SavePurpose(ctx context.Context, obj *Purpose) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, PurposeKeyID, nil, func(i *Purpose) error {
		return ucerr.Wrap(s.saveInnerPurpose(ctx, obj))
	}))
}

// SavePurpose saves a Purpose
func (s *Storage) saveInnerPurpose(ctx context.Context, obj *Purpose) error {
	const q = "INSERT INTO purposes (id, updated, deleted, is_system, name, description) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, is_system = $3, name = $4, description = $5 WHERE (purposes.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SavePurpose", obj, q, obj.ID, obj.Deleted, obj.IsSystem, obj.Name, obj.Description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Purpose %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeletePurpose soft-deletes a Purpose which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeletePurpose(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerPurpose(ctx, objID, false))
}

// deleteInnerPurpose soft-deletes a Purpose which is currently alive
func (s *Storage) deleteInnerPurpose(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Purpose](ctx, *s.cm, s.cm.N.GetKeyNameWithID(PurposeKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Purpose{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Purpose](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE purposes SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeletePurpose", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Purpose %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Purpose %v not found", objID)
	}
	return nil
}

// FlushCacheForPurpose flushes cache for Purpose. It may flush a larger scope then
func (s *Storage) FlushCacheForPurpose(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Purpose"))
}
