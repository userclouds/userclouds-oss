// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// IsMutatorSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsMutatorSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Mutator
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Mutator](ctx, *s.cm, s.cm.N.GetKeyNameWithID(MutatorKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsMutatorSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsMutatorSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM mutators WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsMutatorSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetLatestMutator looks up the latest version of Mutator by ID
func (s *Storage) GetLatestMutator(ctx context.Context, id uuid.UUID) (*Mutator, error) {
	var err error
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *Mutator
		cachedObj, conflict, sentinel, err = cache.GetItemFromCacheWithModifiedKey[Mutator](ctx, *s.cm, s.cm.N.GetKeyNameWithID(MutatorKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), true)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetLatestMutator] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetLatestMutator] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	const q = "SELECT id, updated, deleted, is_system, name, description, version, column_ids, validator_ids, access_policy_id, selector_config, created FROM mutators WHERE id=$1 AND deleted='0001-01-01 00:00:00' ORDER BY version DESC LIMIT 1;"
	var obj Mutator
	if err := s.db.GetContextWithDirty(ctx, "GetLatestMutator", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Mutator %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	if s.cm != nil {
		cache.SaveItemToCache(ctx, *s.cm, obj, sentinel, false, nil)
	}
	return &obj, nil
}

// ListMutatorsPaginated loads a paginated list of Mutators for the specified paginator settings
func (s *Storage) ListMutatorsPaginated(ctx context.Context, p pagination.Paginator) ([]Mutator, *pagination.ResponseFields, error) {
	return s.listInnerMutatorsPaginated(ctx, p, false)
}

// listInnerMutatorsPaginated loads a paginated list of Mutators for the specified paginator settings
func (s *Storage) listInnerMutatorsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Mutator, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(MutatorCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]Mutator
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[Mutator](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, version, column_ids, validator_ids, access_policy_id, selector_config, created FROM (SELECT id, updated, deleted, is_system, name, description, version, column_ids, validator_ids, access_policy_id, selector_config, created FROM mutators WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Mutator
	if err := s.db.SelectContextWithDirty(ctx, "ListMutatorsPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, Mutator{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListMutatorsNonPaginated loads a Mutator up to a limit of 10 pages
func (s *Storage) ListMutatorsNonPaginated(ctx context.Context) ([]Mutator, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]Mutator
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[Mutator](ctx, *s.cm, s.cm.N.GetKeyNameStatic(MutatorCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewMutatorPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]Mutator, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListMutatorsPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListMutatorsNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(MutatorCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, Mutator{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveMutator saves a Mutator
func (s *Storage) SaveMutator(ctx context.Context, obj *Mutator) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, MutatorKeyID, nil, func(i *Mutator) error {
		return ucerr.Wrap(s.saveInnerMutator(ctx, obj))
	}))
}

// SaveMutator saves a Mutator
func (s *Storage) saveInnerMutator(ctx context.Context, obj *Mutator) error {
	const q = "INSERT INTO mutators (id, updated, deleted, is_system, name, description, version, column_ids, validator_ids, access_policy_id, selector_config) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id, version, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, is_system = $3, name = $4, description = $5, version = $6, column_ids = $7, validator_ids = $8, access_policy_id = $9, selector_config = $10 WHERE (mutators.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveMutator", obj, q, obj.ID, obj.Deleted, obj.IsSystem, obj.Name, obj.Description, obj.Version, obj.ColumnIDs, obj.NormalizerIDs, obj.AccessPolicyID, obj.SelectorConfig); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Mutator %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// PriorVersionSaveMutator saves an older version of Mutator
func (s *Storage) PriorVersionSaveMutator(ctx context.Context, obj *Mutator) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerMutator(ctx, obj))
}

// DeleteMutatorByVersion soft-deletes a Mutator which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s Storage) DeleteMutatorByVersion(ctx context.Context, objID uuid.UUID, version int) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Mutator](ctx, *s.cm, s.cm.N.GetKeyNameWithID(MutatorKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Mutator{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Mutator](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE mutators SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteMutatorByVersion", q, objID, version)
	return ucerr.Wrap(err)
}

// DeleteAllMutatorVersions soft-deletes all versions of a Mutator
func (s Storage) DeleteAllMutatorVersions(ctx context.Context, objID uuid.UUID) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Mutator](ctx, *s.cm, s.cm.N.GetKeyNameWithID(MutatorKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Mutator{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Mutator](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE mutators SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAllMutatorVersions", q, objID)
	return ucerr.Wrap(err)
}

// FlushCacheForMutator flushes cache for Mutator. It may flush a larger scope then
func (s *Storage) FlushCacheForMutator(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Mutator"))
}
