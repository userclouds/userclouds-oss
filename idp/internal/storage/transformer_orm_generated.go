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

// IsTransformerSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsTransformerSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *Transformer
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[Transformer](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TransformerKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsTransformerSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsTransformerSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM transformers WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsTransformerSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetLatestTransformer looks up the latest version of Transformer by ID
func (s *Storage) GetLatestTransformer(ctx context.Context, id uuid.UUID) (*Transformer, error) {
	var err error
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	if s.cm != nil {
		var cachedObj *Transformer
		cachedObj, conflict, sentinel, err = cache.GetItemFromCacheWithModifiedKey[Transformer](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TransformerKeyID, id), s.cm.N.GetKeyNameWithID(IsModifiedKeyID, id), true)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetLatestTransformer] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetLatestTransformer] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	const q = "SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created FROM transformers WHERE id=$1 AND deleted='0001-01-01 00:00:00' ORDER BY version DESC LIMIT 1;"
	var obj Transformer
	if err := s.db.GetContextWithDirty(ctx, "GetLatestTransformer", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Transformer %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	if s.cm != nil {
		cache.SaveItemToCache(ctx, *s.cm, obj, sentinel, false, nil)
	}
	return &obj, nil
}

// ListTransformersPaginated loads a paginated list of Transformers for the specified paginator settings
func (s *Storage) ListTransformersPaginated(ctx context.Context, p pagination.Paginator) ([]Transformer, *pagination.ResponseFields, error) {
	return s.listInnerTransformersPaginated(ctx, p, false)
}

// listInnerTransformersPaginated loads a paginated list of Transformers for the specified paginator settings
func (s *Storage) listInnerTransformersPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Transformer, *pagination.ResponseFields, error) {
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	conflict := cache.GenerateTombstoneSentinel()
	cachable := p.IsCachable() && p.GetCursor() == ""
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
		lkey = s.cm.N.GetKeyNameStatic(TransformerCollectionKeyID)
		ckey = lkey
		var err error
		var v *[]Transformer
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[Transformer](ctx, *s.cm, ckey, cachable)
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
	q := fmt.Sprintf("SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created FROM (SELECT id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version, created FROM transformers WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Transformer
	if err := s.db.SelectContextWithDirty(ctx, "ListTransformersPaginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
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
		cache.SaveItemsToCollection(ctx, *s.cm, Transformer{}, objsDB, lkey, ckey, sentinel, true)
	}

	return objs, &respFields, nil
}

// ListTransformersNonPaginated loads a Transformer up to a limit of 10 pages
func (s *Storage) ListTransformersNonPaginated(ctx context.Context) ([]Transformer, error) {
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]Transformer
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[Transformer](ctx, *s.cm, s.cm.N.GetKeyNameStatic(TransformerCollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	pager, err := NewTransformerPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]Transformer, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.ListTransformersPaginated(ctx, *pager)
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
			return nil, ucerr.Errorf("ListTransformersNonPaginated exceeded max page count of 10")
		}
	}
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(TransformerCollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, Transformer{}, objs, ckey, ckey, sentinel, true)
	}
	return objs, nil
}

// SaveTransformer saves a Transformer
func (s *Storage) SaveTransformer(ctx context.Context, obj *Transformer) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, TransformerKeyID, nil, func(i *Transformer) error {
		return ucerr.Wrap(s.saveInnerTransformer(ctx, obj))
	}))
}

// SaveTransformer saves a Transformer
func (s *Storage) saveInnerTransformer(ctx context.Context, obj *Transformer) error {
	const q = "INSERT INTO transformers (id, updated, deleted, is_system, name, description, input_data_type_id, output_data_type_id, reuse_existing_token, transform_type, tag_ids, function, parameters, version) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (id, version, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, is_system = $3, name = $4, description = $5, input_data_type_id = $6, output_data_type_id = $7, reuse_existing_token = $8, transform_type = $9, tag_ids = $10, function = $11, parameters = $12, version = $13 WHERE (transformers.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveTransformer", obj, q, obj.ID, obj.Deleted, obj.IsSystem, obj.Name, obj.Description, obj.InputDataTypeID, obj.OutputDataTypeID, obj.ReuseExistingToken, obj.TransformType, obj.TagIDs, obj.Function, obj.Parameters, obj.Version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Transformer %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// PriorVersionSaveTransformer saves an older version of Transformer
func (s *Storage) PriorVersionSaveTransformer(ctx context.Context, obj *Transformer) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerTransformer(ctx, obj))
}

// DeleteTransformerByVersion soft-deletes a Transformer which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s Storage) DeleteTransformerByVersion(ctx context.Context, objID uuid.UUID, version int) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Transformer](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TransformerKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Transformer{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Transformer](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE transformers SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteTransformerByVersion", q, objID, version)
	return ucerr.Wrap(err)
}

// DeleteAllTransformerVersions soft-deletes all versions of a Transformer
func (s Storage) DeleteAllTransformerVersions(ctx context.Context, objID uuid.UUID) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[Transformer](ctx, *s.cm, s.cm.N.GetKeyNameWithID(TransformerKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := Transformer{SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[Transformer](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE transformers SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAllTransformerVersions", q, objID)
	return ucerr.Wrap(err)
}

// FlushCacheForTransformer flushes cache for Transformer. It may flush a larger scope then
func (s *Storage) FlushCacheForTransformer(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "Transformer"))
}
