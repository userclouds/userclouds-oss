// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// IsUserMFAConfigurationSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsUserMFAConfigurationSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM user_mfa_configuration WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsUserMFAConfigurationSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetUserMFAConfiguration loads a UserMFAConfiguration by ID
func (s *Storage) GetUserMFAConfiguration(ctx context.Context, id uuid.UUID) (*UserMFAConfiguration, error) {
	const q = "SELECT id, updated, deleted, last_evaluated, mfa_channels, created FROM user_mfa_configuration WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj UserMFAConfiguration
	if err := s.db.GetContext(ctx, "GetUserMFAConfiguration", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "UserMFAConfiguration %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetUserMFAConfigurationSoftDeleted loads a UserMFAConfiguration by ID iff it's soft-deleted
func (s *Storage) GetUserMFAConfigurationSoftDeleted(ctx context.Context, id uuid.UUID) (*UserMFAConfiguration, error) {
	const q = "SELECT id, updated, deleted, last_evaluated, mfa_channels, created FROM user_mfa_configuration WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj UserMFAConfiguration
	if err := s.db.GetContext(ctx, "GetUserMFAConfigurationSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted UserMFAConfiguration %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetUserMFAConfigurationsForIDs loads multiple UserMFAConfiguration for a given list of IDs
func (s *Storage) GetUserMFAConfigurationsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]UserMFAConfiguration, error) {
	items := make([]UserMFAConfiguration, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getUserMFAConfigurationsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getUserMFAConfigurationsHelperForIDs loads multiple UserMFAConfiguration for a given list of IDs from the DB
func (s *Storage) getUserMFAConfigurationsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]UserMFAConfiguration, error) {
	const q = "SELECT id, updated, deleted, last_evaluated, mfa_channels, created FROM user_mfa_configuration WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []UserMFAConfiguration
	if err := s.db.SelectContextWithDirty(ctx, "GetUserMFAConfigurationsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested UserMFAConfigurations  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListUserMFAConfigurationsPaginated loads a paginated list of UserMFAConfigurations for the specified paginator settings
func (s *Storage) ListUserMFAConfigurationsPaginated(ctx context.Context, p pagination.Paginator) ([]UserMFAConfiguration, *pagination.ResponseFields, error) {
	return s.listInnerUserMFAConfigurationsPaginated(ctx, p, false)
}

// listInnerUserMFAConfigurationsPaginated loads a paginated list of UserMFAConfigurations for the specified paginator settings
func (s *Storage) listInnerUserMFAConfigurationsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]UserMFAConfiguration, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, last_evaluated, mfa_channels, created FROM (SELECT id, updated, deleted, last_evaluated, mfa_channels, created FROM user_mfa_configuration WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []UserMFAConfiguration
	if err := s.db.SelectContext(ctx, "ListUserMFAConfigurationsPaginated", &objsDB, q, queryFields...); err != nil {
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

	return objs, &respFields, nil
}

// SaveUserMFAConfiguration saves a UserMFAConfiguration
func (s *Storage) SaveUserMFAConfiguration(ctx context.Context, obj *UserMFAConfiguration) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerUserMFAConfiguration(ctx, obj))
}

// SaveUserMFAConfiguration saves a UserMFAConfiguration
func (s *Storage) saveInnerUserMFAConfiguration(ctx context.Context, obj *UserMFAConfiguration) error {
	const q = "INSERT INTO user_mfa_configuration (id, updated, deleted, last_evaluated, mfa_channels) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, last_evaluated = $3, mfa_channels = $4 WHERE (user_mfa_configuration.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveUserMFAConfiguration", obj, q, obj.ID, obj.Deleted, obj.LastEvaluated, obj.MFAChannels); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "UserMFAConfiguration %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteUserMFAConfiguration soft-deletes a UserMFAConfiguration which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteUserMFAConfiguration(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerUserMFAConfiguration(ctx, objID, false))
}

// deleteInnerUserMFAConfiguration soft-deletes a UserMFAConfiguration which is currently alive
func (s *Storage) deleteInnerUserMFAConfiguration(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE user_mfa_configuration SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteUserMFAConfiguration", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting UserMFAConfiguration %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "UserMFAConfiguration %v not found", objID)
	}
	return nil
}
