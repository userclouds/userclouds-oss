// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
)

// IsTenantPlexSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsTenantPlexSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *tenantplex.TenantPlex
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[tenantplex.TenantPlex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(tenantplex.TenantPlexKeyID, id), s.cm.N.GetKeyNameWithID(tenantplex.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[IsTenantPlexSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[IsTenantPlexSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	const q = "/* lint-sql-ok */ SELECT deleted FROM plex_config WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContextWithDirty(ctx, "IsTenantPlexSoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetTenantPlex loads a TenantPlex by ID
func (s *Storage) GetTenantPlex(ctx context.Context, id uuid.UUID) (*tenantplex.TenantPlex, error) {
	return cache.ServerGetItem(ctx, s.cm, id, tenantplex.TenantPlexKeyID, tenantplex.IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *tenantplex.TenantPlex) error {
			const q = "SELECT id, updated, deleted, _version, plex_config, created FROM plex_config WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "GetTenantPlex", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "TenantPlex %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
}

// GetTenantPlexSoftDeleted loads a TenantPlex by ID iff it's soft-deleted
func (s *Storage) GetTenantPlexSoftDeleted(ctx context.Context, id uuid.UUID) (*tenantplex.TenantPlex, error) {
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *tenantplex.TenantPlex
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[tenantplex.TenantPlex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(tenantplex.TenantPlexKeyID, id), s.cm.N.GetKeyNameWithID(tenantplex.IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetTenantPlexSoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetTenantPlexSoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantPlex %v not found", id)
		}
	}
	const q = "SELECT id, updated, deleted, _version, plex_config, created FROM plex_config WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj tenantplex.TenantPlex
	if err := s.db.GetContextWithDirty(ctx, "GetTenantPlexSoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted TenantPlex %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// SaveTenantPlex saves a TenantPlex
func (s *Storage) SaveTenantPlex(ctx context.Context, obj *tenantplex.TenantPlex) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, tenantplex.TenantPlexKeyID, nil, func(i *tenantplex.TenantPlex) error {
		return ucerr.Wrap(s.saveInnerTenantPlex(ctx, obj))
	}))
}

// SaveTenantPlex saves a TenantPlex
func (s *Storage) saveInnerTenantPlex(ctx context.Context, obj *tenantplex.TenantPlex) error {
	// this query has three basic parts
	// 1) INSERT INTO plex_config is used for create only ... any updates will fail with a CONFLICT on (id, deleted)
	// 2) in that case, WHERE will take over to chose the correct row (if any) to update. This includes a check that obj.Version ($3)
	//    matches the _version currently in the database, so that we aren't writing stale data. If this fails, sql.ErrNoRows is returned.
	// 3) if the WHERE matched a row (including version check), the UPDATE will set the new values including $[max] which is newVersion,
	//    which is set to the current version + 1. This is returned in the RETURNING clause so that we can update obj.Version with the new value.
	newVersion := obj.Version + 1
	const q = "INSERT INTO plex_config (id, updated, deleted, _version, plex_config) VALUES ($1, CLOCK_TIMESTAMP(), $2, $5, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, _version = $5, plex_config = $4 WHERE (plex_config._version = $3 AND plex_config.id = $1) RETURNING created, updated, _version; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveTenantPlex", obj, q, obj.ID, obj.Deleted, obj.Version, obj.PlexConfig, newVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "TenantPlex %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteTenantPlex soft-deletes a TenantPlex which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteTenantPlex(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerTenantPlex(ctx, objID, false))
}

// deleteInnerTenantPlex soft-deletes a TenantPlex which is currently alive
func (s *Storage) deleteInnerTenantPlex(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[tenantplex.TenantPlex](ctx, *s.cm, s.cm.N.GetKeyNameWithID(tenantplex.TenantPlexKeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := tenantplex.TenantPlex{VersionBaseModel: ucdb.NewVersionBaseWithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[tenantplex.TenantPlex](ctx, *s.cm, *obj, sentinel)
	}
	const q = "UPDATE plex_config SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteTenantPlex", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting TenantPlex %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "TenantPlex %v not found", objID)
	}
	return nil
}

// FlushCacheForTenantPlex flushes cache for TenantPlex. It may flush a larger scope then
func (s *Storage) FlushCacheForTenantPlex(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "TenantPlex"))
}
