// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// SaveUser saves a User
func (s *UserStorage) SaveUser(ctx context.Context, obj *User) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := s.preSaveUser(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerUser(ctx, obj))
}

// SaveUser saves a User
func (s *UserStorage) saveInnerUser(ctx context.Context, obj *User) error {
	// this query has three basic parts
	// 1) INSERT INTO users is used for create only ... any updates will fail with a CONFLICT on (id, deleted)
	// 2) in that case, WHERE will take over to chose the correct row (if any) to update. This includes a check that obj.Version ($3)
	//    matches the _version currently in the database, so that we aren't writing stale data. If this fails, sql.ErrNoRows is returned.
	// 3) if the WHERE matched a row (including version check), the UPDATE will set the new values including $[max] which is newVersion,
	//    which is set to the current version + 1. This is returned in the RETURNING clause so that we can update obj.Version with the new value.
	newVersion := obj.Version + 1
	const q = "INSERT INTO users (id, updated, deleted, _version, organization_id, region) VALUES ($1, CLOCK_TIMESTAMP(), $2, $6, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, _version = $6, organization_id = $4, region = $5 WHERE (users._version = $3 AND users.id = $1) RETURNING created, updated, _version; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveUser", obj, q, obj.ID, obj.Deleted, obj.Version, obj.OrganizationID, obj.Region, newVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "User %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteUser soft-deletes a User which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *UserStorage) DeleteUser(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerUser(ctx, objID, false))
}

// deleteInnerUser soft-deletes a User which is currently alive
func (s *UserStorage) deleteInnerUser(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	if err := s.preDeleteUser(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	const q = "UPDATE users SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteUser", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting User %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "User %v not found", objID)
	}
	return nil
}
