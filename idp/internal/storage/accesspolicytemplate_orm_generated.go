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

// IsAccessPolicyTemplateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsAccessPolicyTemplateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *AccessPolicyTemplate
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[AccessPolicyTemplate](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyTemplateKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsAccessPolicyTemplateSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsAccessPolicyTemplateSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM access_policy_templates WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsAccessPolicyTemplateSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetLatestAccessPolicyTemplate looks up the latest version of AccessPolicyTemplate by ID
func (s *Storage) GetLatestAccessPolicyTemplate(ctx context.Context, id uuid.UUID) (*AccessPolicyTemplate, error) {
	var err error
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *AccessPolicyTemplate
		cachedObj, conflict, sentinel, err = cache.GetItemFromCacheWithModifiedKey[AccessPolicyTemplate](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyTemplateKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), true)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetLatestAccessPolicyTemplate] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetLatestAccessPolicyTemplate] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	const q = "SELECT id, updated, deleted, is_system, name, description, function, version, created FROM access_policy_templates WHERE id=$1 AND deleted='0001-01-01 00:00:00' ORDER BY version DESC LIMIT 1;"
	var obj AccessPolicyTemplate
	if err := s.db.GetContextWithDirty(ctx, "GetLatestAccessPolicyTemplate", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "AccessPolicyTemplate %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	if s.cm != nil {
		cache.SaveItemToCache(ctx, *s.cm, obj, sentinel, false, nil)
	}
	return &obj, nil
}

// ListAccessPolicyTemplatesPaginated loads a paginated list of AccessPolicyTemplates for the specified paginator settings
func (s *Storage) ListAccessPolicyTemplatesPaginated(ctx context.Context, p pagination.Paginator) ([]AccessPolicyTemplate, *pagination.ResponseFields, error) {
	return s.listInnerAccessPolicyTemplatesPaginated(ctx, p, false)
}

// listInnerAccessPolicyTemplatesPaginated loads a paginated list of AccessPolicyTemplates for the specified paginator settings
func (s *Storage) listInnerAccessPolicyTemplatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]AccessPolicyTemplate, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(AccessPolicyTemplateCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]AccessPolicyTemplate
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[AccessPolicyTemplate](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, function, version, created FROM (SELECT id, updated, deleted, is_system, name, description, function, version, created FROM access_policy_templates WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []AccessPolicyTemplate
	if err := s.db.SelectContextWithDirty(ctx, "ListAccessPolicyTemplatesPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, AccessPolicyTemplate{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListAccessPolicyTemplatesNonPaginated loads a AccessPolicyTemplate up to a limit of 10 pages
func (s *Storage) ListAccessPolicyTemplatesNonPaginated(ctx context.Context) ([]AccessPolicyTemplate, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]AccessPolicyTemplate
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[AccessPolicyTemplate](ctx, *s.cm, s.cm.N.GetKeyNameStatic(AccessPolicyTemplateCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewAccessPolicyTemplatePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]AccessPolicyTemplate, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListAccessPolicyTemplatesPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListAccessPolicyTemplatesNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(AccessPolicyTemplateCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, AccessPolicyTemplate{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveAccessPolicyTemplate saves a AccessPolicyTemplate
func (s *Storage) SaveAccessPolicyTemplate(ctx context.Context, obj *AccessPolicyTemplate) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, AccessPolicyTemplateKeyID, nil, func(i *AccessPolicyTemplate) error {
		return ucerr.Wrap(s.saveInnerAccessPolicyTemplate(ctx, obj))
	}))
}

// SaveAccessPolicyTemplate saves a AccessPolicyTemplate
func (s *Storage) saveInnerAccessPolicyTemplate(ctx context.Context, obj *AccessPolicyTemplate) error {
	const q = "INSERT INTO access_policy_templates (id, updated, deleted, is_system, name, description, function, version) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) ON CONFLICT (id, version, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, is_system = $3, name = $4, description = $5, function = $6, version = $7 WHERE (access_policy_templates.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveAccessPolicyTemplate", obj, q, obj.ID, obj.Deleted, obj.IsSystem, obj.Name, obj.Description, obj.Function, obj.Version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "AccessPolicyTemplate %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// PriorVersionSaveAccessPolicyTemplate saves an older version of AccessPolicyTemplate
func (s *Storage) PriorVersionSaveAccessPolicyTemplate(ctx context.Context, obj *AccessPolicyTemplate) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerAccessPolicyTemplate(ctx, obj))
}

// DeleteAccessPolicyTemplateByVersion soft-deletes a AccessPolicyTemplate which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s Storage) DeleteAccessPolicyTemplateByVersion(ctx context.Context, objID uuid.UUID, version int) error {
	if err := s.preDeleteAccessPolicyTemplate(ctx, objID, version); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[AccessPolicyTemplate](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyTemplateKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := AccessPolicyTemplate{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[AccessPolicyTemplate](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE access_policy_templates SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAccessPolicyTemplateByVersion", q, objID, version)
	return ucerr.Wrap(err)
}

// DeleteAllAccessPolicyTemplateVersions soft-deletes all versions of a AccessPolicyTemplate
func (s Storage) DeleteAllAccessPolicyTemplateVersions(ctx context.Context, objID uuid.UUID) error {
	if err := s.preDeleteAccessPolicyTemplate(ctx, objID, -1); err != nil {
		return ucerr.Wrap(err)
	}
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[AccessPolicyTemplate](ctx, *s.cm, s.cm.N.GetKeyNameWithID(AccessPolicyTemplateKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := AccessPolicyTemplate{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[AccessPolicyTemplate](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE access_policy_templates SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAllAccessPolicyTemplateVersions", q, objID)
	return ucerr.Wrap(err)
}

// FlushCacheForAccessPolicyTemplate flushes cache for AccessPolicyTemplate. It may flush a larger scope then
func (s *Storage) FlushCacheForAccessPolicyTemplate(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "AccessPolicyTemplate"))
}
