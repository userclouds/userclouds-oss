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

// IsAccessPolicySoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsAccessPolicySoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *AccessPolicy
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[AccessPolicy](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsAccessPolicySoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsAccessPolicySoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM access_policies WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsAccessPolicySoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetLatestAccessPolicy looks up the latest version of AccessPolicy by ID
func (s *Storage) GetLatestAccessPolicy(ctx context.Context, id uuid.UUID) (*AccessPolicy, error) {
	var err error
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *AccessPolicy
		cachedObj, conflict, sentinel, err = cache.GetItemFromCacheWithModifiedKey[AccessPolicy](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), true)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetLatestAccessPolicy] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetLatestAccessPolicy] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	const q = "SELECT id, updated, deleted, is_system, name, description, policy_type, tag_ids, version, is_autogenerated, thresholds, component_ids, component_parameters, component_types, metadata, created FROM access_policies WHERE id=$1 AND deleted='0001-01-01 00:00:00' ORDER BY version DESC LIMIT 1;"
	var obj AccessPolicy
	if err := s.db.GetContextWithDirty(ctx, "GetLatestAccessPolicy", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "AccessPolicy %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	if s.cm != nil {
		cache.SaveItemToCache(ctx, *s.cm, obj, sentinel, false, nil)
	}
	return &obj, nil
}

// ListAccessPoliciesPaginated loads a paginated list of AccessPolicies for the specified paginator settings
func (s *Storage) ListAccessPoliciesPaginated(ctx context.Context, p pagination.Paginator) ([]AccessPolicy, *pagination.ResponseFields, error) {
	return s.listInnerAccessPoliciesPaginated(ctx, p, false)
}

// listInnerAccessPoliciesPaginated loads a paginated list of AccessPolicies for the specified paginator settings
func (s *Storage) listInnerAccessPoliciesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]AccessPolicy, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(AccessPolicyCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]AccessPolicy
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[AccessPolicy](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, policy_type, tag_ids, version, is_autogenerated, thresholds, component_ids, component_parameters, component_types, metadata, created FROM (SELECT id, updated, deleted, is_system, name, description, policy_type, tag_ids, version, is_autogenerated, thresholds, component_ids, component_parameters, component_types, metadata, created FROM access_policies WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []AccessPolicy
	if err := s.db.SelectContextWithDirty(ctx, "ListAccessPoliciesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, AccessPolicy{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListAccessPoliciesNonPaginated loads a AccessPolicy up to a limit of 10 pages
func (s *Storage) ListAccessPoliciesNonPaginated(ctx context.Context) ([]AccessPolicy, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]AccessPolicy
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[AccessPolicy](ctx, *s.cm, s.cm.N.GetKeyNameStatic(AccessPolicyCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewAccessPolicyPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]AccessPolicy, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListAccessPoliciesPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListAccessPoliciesNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(AccessPolicyCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, AccessPolicy{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveAccessPolicy saves a AccessPolicy
func (s *Storage) SaveAccessPolicy(ctx context.Context, obj *AccessPolicy) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, AccessPolicyKeyID, nil, func(i *AccessPolicy) error {
		return ucerr.Wrap(s.saveInnerAccessPolicy(ctx, obj))
	}))
}

// SaveAccessPolicy saves a AccessPolicy
func (s *Storage) saveInnerAccessPolicy(ctx context.Context, obj *AccessPolicy) error {
	const q = "INSERT INTO access_policies (id, updated, deleted, is_system, name, description, policy_type, tag_ids, version, is_autogenerated, thresholds, component_ids, component_parameters, component_types, metadata) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (id, version, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, is_system = $3, name = $4, description = $5, policy_type = $6, tag_ids = $7, version = $8, is_autogenerated = $9, thresholds = $10, component_ids = $11, component_parameters = $12, component_types = $13, metadata = $14 WHERE (access_policies.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveAccessPolicy", obj, q, obj.ID, obj.Deleted, obj.IsSystem, obj.Name, obj.Description, obj.PolicyType, obj.TagIDs, obj.Version, obj.IsAutogenerated, obj.Thresholds, obj.ComponentIDs, obj.ComponentParameters, obj.ComponentTypes, obj.Metadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "AccessPolicy %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// PriorVersionSaveAccessPolicy saves an older version of AccessPolicy
func (s *Storage) PriorVersionSaveAccessPolicy(ctx context.Context, obj *AccessPolicy) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerAccessPolicy(ctx, obj))
}

// DeleteAccessPolicyByVersion soft-deletes a AccessPolicy which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s Storage) DeleteAccessPolicyByVersion(ctx context.Context, objID uuid.UUID, version int) error {
	if err := s.preDeleteAccessPolicy(ctx, objID, version); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[AccessPolicy](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := AccessPolicy{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[AccessPolicy](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE access_policies SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAccessPolicyByVersion", q, objID, version)
	return ucerr.Wrap(err)
}

// DeleteAllAccessPolicyVersions soft-deletes all versions of a AccessPolicy
func (s Storage) DeleteAllAccessPolicyVersions(ctx context.Context, objID uuid.UUID) error {
	if err := s.preDeleteAccessPolicy(ctx, objID, -1); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[AccessPolicy](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := AccessPolicy{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[AccessPolicy](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE access_policies SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAllAccessPolicyVersions", q, objID)
	return ucerr.Wrap(err)
}

// FlushCacheForAccessPolicy flushes cache for AccessPolicy. It may flush a larger scope then
func (s *Storage) FlushCacheForAccessPolicy(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "AccessPolicy"))
}
