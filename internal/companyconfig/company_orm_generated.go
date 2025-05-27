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

// IsCompanySoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsCompanySoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Company
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Company](ctx, *s.cm, s.cm.N.GetKeyNameWithID(CompanyKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsCompanySoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsCompanySoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM companies WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsCompanySoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetCompany loads a Company by ID
func (s *Storage) GetCompany(ctx context.Context, id uuid.UUID) (*Company, error) {
	return cache.ServerGetItem(ctx, s.cm, id, CompanyKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *Company) error {
			const q = "SELECT id, updated, deleted, name, type, created FROM companies WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetCompany", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Company %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetCompanySoftDeleted loads a Company by ID iff it's soft-deleted
func (s *Storage) GetCompanySoftDeleted(ctx context.Context, id uuid.UUID) (*Company, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Company
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Company](ctx, *s.cm, s.cm.N.GetKeyNameWithID(CompanyKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetCompanySoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetCompanySoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Company %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, name, type, created FROM companies WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Company
	if err := s.db.GetContextWithDirty(ctx, "GetCompanySoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Company %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetCompanysForIDs loads multiple Company for a given list of IDs
func (s *Storage) GetCompanysForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Company, error) {
	items := make([]Company, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(CompanyKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[Company](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getCompaniesHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetCompanyMap: returning %d Company. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getCompaniesHelperForIDs loads multiple Company for a given list of IDs from the DB
func (s *Storage) getCompaniesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Company, error) {
	const q = "SELECT id, updated, deleted, name, type, created FROM companies WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Company
	if err := s.db.SelectContextWithDirty(ctx, "GetCompaniesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Companies  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListCompaniesPaginated loads a paginated list of Companies for the specified paginator settings
func (s *Storage) ListCompaniesPaginated(ctx context.Context, p pagination.Paginator) ([]Company, *pagination.ResponseFields, error) {
	return s.listInnerCompaniesPaginated(ctx, p, false)
}

// listInnerCompaniesPaginated loads a paginated list of Companies for the specified paginator settings
func (s *Storage) listInnerCompaniesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Company, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(CompanyCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]Company
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[Company](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, name, type, created FROM (SELECT id, updated, deleted, name, type, created FROM companies WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Company
	if err := s.db.SelectContextWithDirty(ctx, "ListCompaniesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, Company{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveCompany saves a Company
func (s *Storage) SaveCompany(ctx context.Context, obj *Company) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, CompanyKeyID, nil, func(i *Company) error {
		return ucerr.Wrap(s.saveInnerCompany(ctx, obj))
	}))
}

// SaveCompany saves a Company
func (s *Storage) saveInnerCompany(ctx context.Context, obj *Company) error {
	const q = "INSERT INTO companies (id, updated, deleted, name, type) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, type = $4 WHERE (companies.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveCompany", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Type); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Company %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteCompany soft-deletes a Company which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteCompany(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerCompany(ctx, objID, false))
}

// deleteInnerCompany soft-deletes a Company which is currently alive
func (s *Storage) deleteInnerCompany(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteCompany(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Company](ctx, *s.cm, s.cm.N.GetKeyNameWithID(CompanyKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Company{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Company](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE companies SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteCompany", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Company %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Company %v not found", objID)
	}
	return nil
}

// FlushCacheForCompany flushes cache for Company. It may flush a larger scope then
func (s *Storage) FlushCacheForCompany(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Company"))
}
