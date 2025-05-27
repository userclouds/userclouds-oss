package storage

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
)

// UserCleanupReason is an enum identifying why a user is being cleaned
type UserCleanupReason int

// UserCleanupReason constants
const (
	UserCleanupReasonDuplicateValue UserCleanupReason = 1
)

//go:generate genconstant UserCleanupReason

// UserCleanupCandidate represents a candidate user record to be cleaned
type UserCleanupCandidate struct {
	ucdb.UserBaseModel
	CleanupReason UserCleanupReason `db:"cleanup_reason" json:"cleanup_reason"`
}

//go:generate genpageable UserCleanupCandidate

//go:generate genvalidate UserCleanupCandidate

//go:generate genorm --storageclassprefix UserCleanupCandidate user_cleanup_candidates tenantdb User

// CleanupUsers will attempt to clean up up to maxCandidates user cleanup candidates
func (s *UserStorage) CleanupUsers(ctx context.Context, cm *ColumnManager, maxCandidates int, dryRun bool) (int, error) {
	candidates, err := s.dequeueUserCleanupCandidates(ctx, maxCandidates)
	if err != nil {
		return -1, ucerr.Wrap(err)
	}

	if len(candidates) == 0 {
		return 0, nil
	}
	uclog.Infof(ctx, "cleaning up %d user cleanup candidates", len(candidates))

	for _, candidate := range candidates {
		if err := s.cleanupUser(ctx, cm, candidate, dryRun); err != nil {
			return -1, ucerr.Wrap(err)
		}

		if dryRun {
			if err := s.requeueUserCleanupCandidate(ctx, candidate); err != nil {
				return -1, ucerr.Wrap(err)
			}
		}
	}
	remainingCount, err := s.getCandidatesCount(ctx)
	if err != nil {
		return -1, ucerr.Wrap(err)
	}
	return remainingCount, nil
}

func (s *UserStorage) cleanupUser(
	ctx context.Context,
	cm *ColumnManager,
	candidate UserCleanupCandidate,
	dryRun bool,
) error {
	if candidate.CleanupReason != UserCleanupReasonDuplicateValue {
		return nil
	}
	uclog.Debugf(ctx, "cleaning up user '%v' for reason '%v'", candidate.UserID, candidate.CleanupReason)
	const q = `
SELECT
id,
created,
updated,
deleted,
_version,
column_id,
user_id,
ordering,
consented_purpose_ids,
retention_timeouts,
varchar_value,
varchar_unique_value,
boolean_value,
int_value,
int_unique_value,
timestamp_value,
uuid_value,
uuid_unique_value,
jsonb_value
FROM
user_column_pre_delete_values
WHERE
deleted = '0001-01-01 00:00:00'
AND user_id = $1
ORDER BY
column_id,
ordering,
updated;`

	var values []UserColumnLiveValue
	if err := s.db.SelectContext(
		ctx,
		"cleanupUser.DuplicateValue",
		&values,
		q,
		candidate.UserID,
	); err != nil {
		return ucerr.Wrap(err)
	}

	var deletedValues []UserColumnLiveValue
	columnID := uuid.Nil
	orderings := set.NewIntSet()
	for _, value := range values {
		if value.ColumnID != columnID {
			columnID = value.ColumnID
			orderings = set.NewIntSet()
		}

		if orderings.Contains(value.Ordering) {
			c := cm.GetColumnByID(columnID)
			if c == nil {
				return ucerr.Errorf(
					"value '%v' column ID '%v' is unrecognized",
					value.ID,
					columnID,
				)
			}
			value.Column = c
			value.IsNew = false

			uclog.Infof(
				ctx,
				"deleting duplicate value '%v' for user '%v'",
				value.ID,
				value.UserID,
			)
			deletedValues = append(deletedValues, value)
		} else {
			orderings.Insert(value.Ordering)
		}
	}

	if !dryRun {
		if err := s.DeleteUserColumnLiveValues(ctx, deletedValues); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (s *UserStorage) getCandidatesCount(ctx context.Context) (int, error) {
	const countQuery = "SELECT COUNT(*) FROM user_cleanup_candidates WHERE deleted='0001-01-01 00:00:00'"
	var count int
	if err := s.db.GetContext(ctx, "dequeueUserCleanupCandidates.getCandidatesCount", &count, countQuery); err != nil {
		return -1, ucerr.Wrap(err)
	}
	return count, nil
}

func (s *UserStorage) dequeueUserCleanupCandidates(ctx context.Context, maxCandidates int) ([]UserCleanupCandidate, error) {
	if maxCandidates < 1 {
		return nil, ucerr.Errorf("maxCandidates must be greater than or equal to one: %d", maxCandidates)
	}

	const selectQuery = "SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE deleted='0001-01-01 00:00:00' ORDER BY created LIMIT $1"

	var candidates []UserCleanupCandidate
	if err := s.db.SelectContext(ctx, "dequeueUserCleanupCandidates.selectQuery", &candidates, selectQuery, maxCandidates); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	var ids uuidarray.UUIDArray
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}

	const deleteQuery = "UPDATE user_cleanup_candidates SET deleted=NOW() WHERE id = ANY($1) AND deleted='0001-01-01 00:00:00'"
	res, err := s.db.ExecContext(ctx, "dequeueUserCleanupCandidates.deleteQuery", deleteQuery, ids)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ra, err := res.RowsAffected()
	if err != nil {
		return nil, ucerr.Errorf("error determining number of deleted user candidates, expected to delete '%v': '%v'", ids, err)
	}
	if ra != int64(len(ids)) {
		return nil, ucerr.Errorf("deleted %d user candidates, expected to delete '%v'", ra, ids)
	}

	return candidates, nil
}

// EnqueueUserCleanupCandidate will add a user cleanup candidate to the queue
func (s *UserStorage) EnqueueUserCleanupCandidate(ctx context.Context, userID uuid.UUID, reason UserCleanupReason) error {
	ucc := UserCleanupCandidate{
		UserBaseModel: ucdb.NewUserBase(userID),
		CleanupReason: reason,
	}

	if err := s.SaveUserCleanupCandidate(ctx, &ucc); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (s *UserStorage) requeueUserCleanupCandidate(ctx context.Context, ucc UserCleanupCandidate) error {
	ucc.Deleted = time.Time{}

	if err := s.SaveUserCleanupCandidate(ctx, &ucc); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
