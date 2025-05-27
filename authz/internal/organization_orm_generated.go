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

// IsOrganizationSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsOrganizationSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Organization
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Organization](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.OrganizationKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsOrganizationSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsOrganizationSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM organizations WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsOrganizationSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetOrganization loads a Organization by ID
func (s *Storage) GetOrganization(ctx context.Context, id uuid.UUID) (*authz.Organization, error) {
	return cache.ServerGetItem(ctx, s.cm, id, authz.OrganizationKeyID, authz.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *authz.Organization) error {
			const q = "SELECT id, updated, deleted, name, region, created FROM organizations WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetOrganization", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Organization %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// getOrganizationByColumns loads a Organization using the provided column names and values as a WHERE clause
func (s *Storage) getOrganizationByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*authz.Organization, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *authz.Organization
		mkey := s.cm.N.GetKeyNameStatic(authz.OrganizationCollectionKeyID)
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Organization](ctx, *s.cm, secondaryKey, mkey, false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[getOrganizationByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[getOrganizationByColumns] error reading from local cache: %v", err)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, region, created FROM organizations WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj authz.Organization
	if err := s.db.GetContextWithDirty(ctx, "GetOrganizationForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Organization %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.GetOrganization(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for Organization with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving Organization with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	return &obj, nil
}

// GetOrganizationSoftDeleted loads a Organization by ID iff it's soft-deleted
func (s *Storage) GetOrganizationSoftDeleted(ctx context.Context, id uuid.UUID) (*authz.Organization, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *authz.Organization
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[authz.Organization](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.OrganizationKeyID, id), s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetOrganizationSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetOrganizationSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Organization %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, region, created FROM organizations WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj authz.Organization
	if err := s.db.GetContextWithDirty(ctx, "GetOrganizationSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Organization %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetOrganizationsForIDs loads multiple Organization for a given list of IDs
func (s *Storage) GetOrganizationsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Organization, error) {
	items := make([]authz.Organization, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(authz.OrganizationKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(authz.IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[authz.Organization](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getOrganizationsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetOrganizationMap: returning %d Organization. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getOrganizationsHelperForIDs loads multiple Organization for a given list of IDs from the DB
func (s *Storage) getOrganizationsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]authz.Organization, error) {
	const q = "SELECT id, updated, deleted, name, region, created FROM organizations WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []authz.Organization
	if err := s.db.SelectContextWithDirty(ctx, "GetOrganizationsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Organizations  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListOrganizationsPaginated loads a paginated list of Organizations for the specified paginator settings
func (s *Storage) ListOrganizationsPaginated(ctx context.Context, p pagination.Paginator) ([]authz.Organization, *pagination.ResponseFields, error) {
	return s.listInnerOrganizationsPaginated(ctx, p, false)
}

// listInnerOrganizationsPaginated loads a paginated list of Organizations for the specified paginator settings
func (s *Storage) listInnerOrganizationsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]authz.Organization, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(authz.OrganizationCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]authz.Organization
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[authz.Organization](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, region, created FROM (SELECT id, updated, deleted, name, region, created FROM organizations WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []authz.Organization
	if err := s.db.SelectContextWithDirty(ctx, "ListOrganizationsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, authz.Organization{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveOrganization saves a Organization
func (s *Storage) SaveOrganization(ctx context.Context, obj *authz.Organization) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveOrganization(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.OrganizationKeyID, nil, func(i *authz.Organization) error {
		return ucerr.Wrap(s.saveInnerOrganization(ctx, obj))
	}))
}

// SaveOrganization saves a Organization
func (s *Storage) saveInnerOrganization(ctx context.Context, obj *authz.Organization) error {
	const q = "INSERT INTO organizations (id, updated, deleted, name, region) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, region = $4 WHERE (organizations.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveOrganization", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Region); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Organization %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// InsertOrganization inserts a Organization without resolving conflict with existing rows
func (s *Storage) InsertOrganization(ctx context.Context, obj *authz.Organization) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveOrganization(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, authz.OrganizationKeyID, nil, func(i *authz.Organization) error {
		return ucerr.Wrap(s.insertInnerOrganization(ctx, obj))
	}))
}

// insertInnerOrganization inserts a Organization without resolving conflict with existing rows
func (s *Storage) insertInnerOrganization(ctx context.Context, obj *authz.Organization) error {
	const q = "INSERT INTO organizations (id, updated, deleted, name, region) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "InsertOrganization", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Region); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteOrganization soft-deletes a Organization which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteOrganization(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerOrganization(ctx, objID, false))
}

// deleteInnerOrganization soft-deletes a Organization which is currently alive
func (s *Storage) deleteInnerOrganization(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteOrganization(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[authz.Organization](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.OrganizationKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := authz.Organization{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[authz.Organization](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE organizations SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteOrganization", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Organization %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Organization %v not found", objID)
	}
	return nil
}

// FlushCacheForOrganization flushes cache for Organization. It may flush a larger scope then
func (s *Storage) FlushCacheForOrganization(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Organization"))
}
