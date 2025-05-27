// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

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

// IsTenantInternalSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsTenantInternalSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *TenantInternal
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[TenantInternal](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantInternalKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsTenantInternalSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsTenantInternalSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM tenants_internal WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsTenantInternalSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetTenantInternal loads a TenantInternal by ID
func (s *Storage) GetTenantInternal(ctx context.Context, id uuid.UUID) (*TenantInternal, error) {
	return cache.ServerGetItem(ctx, s.cm, id, TenantInternalKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *TenantInternal) error {
			const q = "SELECT id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup, created FROM tenants_internal WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetTenantInternal", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "TenantInternal %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetTenantInternalSoftDeleted loads a TenantInternal by ID iff it's soft-deleted
func (s *Storage) GetTenantInternalSoftDeleted(ctx context.Context, id uuid.UUID) (*TenantInternal, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *TenantInternal
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[TenantInternal](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantInternalKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetTenantInternalSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetTenantInternalSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantInternal %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup, created FROM tenants_internal WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj TenantInternal
	if err := s.db.GetContextWithDirty(ctx, "GetTenantInternalSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantInternal %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetTenantInternalsForIDs loads multiple TenantInternal for a given list of IDs
func (s *Storage) GetTenantInternalsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]TenantInternal, error) {
	items := make([]TenantInternal, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(TenantInternalKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[TenantInternal](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getTenantInternalsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetTenantInternalMap: returning %d TenantInternal. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getTenantInternalsHelperForIDs loads multiple TenantInternal for a given list of IDs from the DB
func (s *Storage) getTenantInternalsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]TenantInternal, error) {
	const q = "SELECT id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup, created FROM tenants_internal WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []TenantInternal
	if err := s.db.SelectContextWithDirty(ctx, "GetTenantInternalsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested TenantInternals  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListTenantInternalsPaginated loads a paginated list of TenantInternals for the specified paginator settings
func (s *Storage) ListTenantInternalsPaginated(ctx context.Context, p pagination.Paginator) ([]TenantInternal, *pagination.ResponseFields, error) {
	return s.listInnerTenantInternalsPaginated(ctx, p, false)
}

// listInnerTenantInternalsPaginated loads a paginated list of TenantInternals for the specified paginator settings
func (s *Storage) listInnerTenantInternalsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]TenantInternal, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(TenantInternalCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]TenantInternal
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[TenantInternal](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup, created FROM (SELECT id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup, created FROM tenants_internal WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []TenantInternal
	if err := s.db.SelectContextWithDirty(ctx, "ListTenantInternalsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, TenantInternal{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveTenantInternal saves a TenantInternal
func (s *Storage) SaveTenantInternal(ctx context.Context, obj *TenantInternal) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, TenantInternalKeyID, nil, func(i *TenantInternal) error {
		return ucerr.Wrap(s.saveInnerTenantInternal(ctx, obj))
	}))
}

// SaveTenantInternal saves a TenantInternal
func (s *Storage) saveInnerTenantInternal(ctx context.Context, obj *TenantInternal) error {
	const q = "INSERT INTO tenants_internal (id, updated, deleted, tenant_db_config, log_config, primary_user_region, remote_user_region_db_configs, connect_on_startup) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, tenant_db_config = $3, log_config = $4, primary_user_region = $5, remote_user_region_db_configs = $6, connect_on_startup = $7 WHERE (tenants_internal.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveTenantInternal", obj, q, obj.ID, obj.Deleted, obj.TenantDBConfig, obj.LogConfig, obj.PrimaryUserRegion, obj.RemoteUserRegionDBConfigs, obj.ConnectOnStartup); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "TenantInternal %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteTenantInternal soft-deletes a TenantInternal which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteTenantInternal(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerTenantInternal(ctx, objID, false))
}

// deleteInnerTenantInternal soft-deletes a TenantInternal which is currently alive
func (s *Storage) deleteInnerTenantInternal(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[TenantInternal](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantInternalKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := TenantInternal{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[TenantInternal](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE tenants_internal SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteTenantInternal", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting TenantInternal %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "TenantInternal %v not found", objID)
	}
	return nil
}

// FlushCacheForTenantInternal flushes cache for TenantInternal. It may flush a larger scope then
func (s *Storage) FlushCacheForTenantInternal(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "TenantInternal"))
}
