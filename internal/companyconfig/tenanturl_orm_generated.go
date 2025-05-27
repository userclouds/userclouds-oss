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

// IsTenantURLSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsTenantURLSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *TenantURL
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[TenantURL](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantURLKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsTenantURLSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsTenantURLSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM tenants_urls WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsTenantURLSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetTenantURL loads a TenantURL by ID
func (s *Storage) GetTenantURL(ctx context.Context, id uuid.UUID) (*TenantURL, error) {
	return cache.ServerGetItem(ctx, s.cm, id, TenantURLKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *TenantURL) error {
			const q = "SELECT id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until, created FROM tenants_urls WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetTenantURL", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "TenantURL %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetTenantURLSoftDeleted loads a TenantURL by ID iff it's soft-deleted
func (s *Storage) GetTenantURLSoftDeleted(ctx context.Context, id uuid.UUID) (*TenantURL, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *TenantURL
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[TenantURL](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantURLKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetTenantURLSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetTenantURLSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantURL %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until, created FROM tenants_urls WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj TenantURL
	if err := s.db.GetContextWithDirty(ctx, "GetTenantURLSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantURL %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetTenantURLsForIDs loads multiple TenantURL for a given list of IDs
func (s *Storage) GetTenantURLsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]TenantURL, error) {
	items := make([]TenantURL, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(TenantURLKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[TenantURL](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getTenantURLsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetTenantURLMap: returning %d TenantURL. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getTenantURLsHelperForIDs loads multiple TenantURL for a given list of IDs from the DB
func (s *Storage) getTenantURLsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]TenantURL, error) {
	const q = "SELECT id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until, created FROM tenants_urls WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []TenantURL
	if err := s.db.SelectContextWithDirty(ctx, "GetTenantURLsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested TenantURLs  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListTenantURLsPaginated loads a paginated list of TenantURLs for the specified paginator settings
func (s *Storage) ListTenantURLsPaginated(ctx context.Context, p pagination.Paginator) ([]TenantURL, *pagination.ResponseFields, error) {
	return s.listInnerTenantURLsPaginated(ctx, p, false)
}

// listInnerTenantURLsPaginated loads a paginated list of TenantURLs for the specified paginator settings
func (s *Storage) listInnerTenantURLsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]TenantURL, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(TenantURLCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]TenantURL
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[TenantURL](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until, created FROM (SELECT id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until, created FROM tenants_urls WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []TenantURL
	if err := s.db.SelectContextWithDirty(ctx, "ListTenantURLsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, TenantURL{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveTenantURL saves a TenantURL
func (s *Storage) SaveTenantURL(ctx context.Context, obj *TenantURL) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, TenantURLKeyID, s.additionalSaveKeysForTenantURL(obj), func(i *TenantURL) error {
		return ucerr.Wrap(s.saveInnerTenantURL(ctx, obj))
	}))
}

// SaveTenantURL saves a TenantURL
func (s *Storage) saveInnerTenantURL(ctx context.Context, obj *TenantURL) error {
	const q = "INSERT INTO tenants_urls (id, updated, deleted, tenant_id, tenant_url, validated, system, active, dns_verifier, certificate_valid_until) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, tenant_id = $3, tenant_url = $4, validated = $5, system = $6, active = $7, dns_verifier = $8, certificate_valid_until = $9 WHERE (tenants_urls.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveTenantURL", obj, q, obj.ID, obj.Deleted, obj.TenantID, obj.TenantURL, obj.Validated, obj.System, obj.Active, obj.DNSVerifier, obj.CertificateValidUntil); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "TenantURL %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteTenantURL soft-deletes a TenantURL which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteTenantURL(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerTenantURL(ctx, objID, false))
}

// deleteInnerTenantURL soft-deletes a TenantURL which is currently alive
func (s *Storage) deleteInnerTenantURL(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[TenantURL](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TenantURLKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := TenantURL{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[TenantURL](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE tenants_urls SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteTenantURL", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting TenantURL %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "TenantURL %v not found", objID)
	}
	return nil
}

// FlushCacheForTenantURL flushes cache for TenantURL. It may flush a larger scope then
func (s *Storage) FlushCacheForTenantURL(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "TenantURL"))
}
