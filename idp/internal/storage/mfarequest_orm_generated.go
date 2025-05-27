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

// IsMFARequestSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsMFARequestSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM mfa_requests WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsMFARequestSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetMFARequest loads a MFARequest by ID
func (s *Storage) GetMFARequest(ctx context.Context, id uuid.UUID) (*MFARequest, error) {
	const q = "SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM mfa_requests WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj MFARequest
	if err := s.db.GetContext(ctx, "GetMFARequest", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "MFARequest %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetMFARequestSoftDeleted loads a MFARequest by ID iff it's soft-deleted
func (s *Storage) GetMFARequestSoftDeleted(ctx context.Context, id uuid.UUID) (*MFARequest, error) {
	const q = "SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM mfa_requests WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj MFARequest
	if err := s.db.GetContext(ctx, "GetMFARequestSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted MFARequest %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetMFARequestsForIDs loads multiple MFARequest for a given list of IDs
func (s *Storage) GetMFARequestsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]MFARequest, error) {
	items := make([]MFARequest, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getMFARequestsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getMFARequestsHelperForIDs loads multiple MFARequest for a given list of IDs from the DB
func (s *Storage) getMFARequestsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]MFARequest, error) {
	const q = "SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM mfa_requests WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []MFARequest
	if err := s.db.SelectContextWithDirty(ctx, "GetMFARequestsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested MFARequests  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListMFARequestsPaginated loads a paginated list of MFARequests for the specified paginator settings
func (s *Storage) ListMFARequestsPaginated(ctx context.Context, p pagination.Paginator) ([]MFARequest, *pagination.ResponseFields, error) {
	return s.listInnerMFARequestsPaginated(ctx, p, false)
}

// listInnerMFARequestsPaginated loads a paginated list of MFARequests for the specified paginator settings
func (s *Storage) listInnerMFARequestsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]MFARequest, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM (SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM mfa_requests WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []MFARequest
	if err := s.db.SelectContext(ctx, "ListMFARequestsPaginated", &objsDB, q, queryFields...); err != nil {
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

// ListMFARequestsForUserID loads the list of MFARequests with a matching UserID field
func (s *Storage) ListMFARequestsForUserID(ctx context.Context, userID uuid.UUID) ([]MFARequest, error) {
	const q = "SELECT id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types, created FROM mfa_requests WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []MFARequest
	if err := s.db.SelectContext(ctx, "ListMFARequestsForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}

// SaveMFARequest saves a MFARequest
func (s *Storage) SaveMFARequest(ctx context.Context, obj *MFARequest) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerMFARequest(ctx, obj))
}

// SaveMFARequest saves a MFARequest
func (s *Storage) saveInnerMFARequest(ctx context.Context, obj *MFARequest) error {
	const q = "INSERT INTO mfa_requests (id, updated, deleted, user_id, issued, code, channel_id, supported_channel_types) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, user_id = $3, issued = $4, code = $5, channel_id = $6, supported_channel_types = $7 WHERE (mfa_requests.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveMFARequest", obj, q, obj.ID, obj.Deleted, obj.UserID, obj.Issued, obj.Code, obj.ChannelID, obj.SupportedChannelTypes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "MFARequest %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteMFARequest soft-deletes a MFARequest which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteMFARequest(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerMFARequest(ctx, objID, false))
}

// deleteInnerMFARequest soft-deletes a MFARequest which is currently alive
func (s *Storage) deleteInnerMFARequest(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE mfa_requests SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteMFARequest", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting MFARequest %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "MFARequest %v not found", objID)
	}
	return nil
}
