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

// IsSessionSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsSessionSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Session
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Session](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SessionKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsSessionSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsSessionSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM sessions WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsSessionSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetSession loads a Session by ID
func (s *Storage) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	return cache.ServerGetItem(ctx, s.cm, id, SessionKeyID, IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *Session) error {
			const q = "SELECT id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token, created FROM sessions WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetSession", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "Session %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetSessionSoftDeleted loads a Session by ID iff it's soft-deleted
func (s *Storage) GetSessionSoftDeleted(ctx context.Context, id uuid.UUID) (*Session, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Session
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Session](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SessionKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetSessionSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetSessionSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted Session %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token, created FROM sessions WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Session
	if err := s.db.GetContextWithDirty(ctx, "GetSessionSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Session %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetSessionsForIDs loads multiple Session for a given list of IDs
func (s *Storage) GetSessionsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Session, error) {
	items := make([]Session, 0, len(ids))

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
			keys = append(keys, s.cm.N.GetKeyNameWithID(SessionKeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[Session](ctx, *s.cm, keys, modKeys, locks)
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
		itemsFromDB, err := s.getSessionsHelperForIDs(ctx, dirty, true, missed.Items()...)
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
	uclog.Verbosef(ctx, "GetSessionMap: returning %d Session. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)

	return items, nil
}

// getSessionsHelperForIDs loads multiple Session for a given list of IDs from the DB
func (s *Storage) getSessionsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Session, error) {
	const q = "SELECT id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token, created FROM sessions WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Session
	if err := s.db.SelectContextWithDirty(ctx, "GetSessionsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Sessions  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListSessionsPaginated loads a paginated list of Sessions for the specified paginator settings
func (s *Storage) ListSessionsPaginated(ctx context.Context, p pagination.Paginator) ([]Session, *pagination.ResponseFields, error) {
	return s.listInnerSessionsPaginated(ctx, p, false)
}

// listInnerSessionsPaginated loads a paginated list of Sessions for the specified paginator settings
func (s *Storage) listInnerSessionsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Session, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(SessionCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]Session
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[Session](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token, created FROM (SELECT id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token, created FROM sessions WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Session
	if err := s.db.SelectContextWithDirty(ctx, "ListSessionsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, Session{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// SaveSession saves a Session
func (s *Storage) SaveSession(ctx context.Context, obj *Session) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, SessionKeyID, nil, func(i *Session) error {
		return ucerr.Wrap(s.saveInnerSession(ctx, obj))
	}))
}

// SaveSession saves a Session
func (s *Storage) saveInnerSession(ctx context.Context, obj *Session) error {
	const q = "INSERT INTO sessions (id, updated, deleted, id_token, access_token, refresh_token, state, impersonator_id_token, impersonator_access_token, impersonator_refresh_token) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, id_token = $3, access_token = $4, refresh_token = $5, state = $6, impersonator_id_token = $7, impersonator_access_token = $8, impersonator_refresh_token = $9 WHERE (sessions.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveSession", obj, q, obj.ID, obj.Deleted, obj.IDToken, obj.AccessToken, obj.RefreshToken, obj.State, obj.ImpersonatorIDToken, obj.ImpersonatorAccessToken, obj.ImpersonatorRefreshToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Session %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteSession soft-deletes a Session which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteSession(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerSession(ctx, objID, false))
}

// deleteInnerSession soft-deletes a Session which is currently alive
func (s *Storage) deleteInnerSession(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Session](ctx, *s.cm, s.cm.N.GetKeyNameWithID(SessionKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Session{BaseModel: ucdb.NewBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Session](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE sessions SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteSession", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Session %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Session %v not found", objID)
	}
	return nil
}

// FlushCacheForSession flushes cache for Session. It may flush a larger scope then
func (s *Storage) FlushCacheForSession(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Session"))
}
