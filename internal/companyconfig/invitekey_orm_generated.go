// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// ListInviteKeysPaginated loads a paginated list of InviteKeys for the specified paginator settings
func (s *Storage) ListInviteKeysPaginated(ctx context.Context, p pagination.Paginator) ([]InviteKey, *pagination.ResponseFields, error) {
	return s.listInnerInviteKeysPaginated(ctx, p, false)
}

// listInnerInviteKeysPaginated loads a paginated list of InviteKeys for the specified paginator settings
func (s *Storage) listInnerInviteKeysPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]InviteKey, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, type, key, expires, used, company_id, role, tenant_roles, invitee_email, invitee_user_id, created FROM (SELECT id, updated, deleted, type, key, expires, used, company_id, role, tenant_roles, invitee_email, invitee_user_id, created FROM invite_keys WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []InviteKey
	if err := s.db.SelectContext(ctx, "ListInviteKeysPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveInviteKey saves a InviteKey
func (s *Storage) SaveInviteKey(ctx context.Context, obj *InviteKey) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerInviteKey(ctx, obj))
}

// SaveInviteKey saves a InviteKey
func (s *Storage) saveInnerInviteKey(ctx context.Context, obj *InviteKey) error {
	const q = "INSERT INTO invite_keys (id, updated, deleted, type, key, expires, used, company_id, role, tenant_roles, invitee_email, invitee_user_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type = $3, key = $4, expires = $5, used = $6, company_id = $7, role = $8, tenant_roles = $9, invitee_email = $10, invitee_user_id = $11 WHERE (invite_keys.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveInviteKey", obj, q, obj.ID, obj.Deleted, obj.Type, obj.Key, obj.Expires, obj.Used, obj.CompanyID, obj.Role, obj.TenantRoles, obj.InviteeEmail, obj.InviteeUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "InviteKey %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteInviteKey soft-deletes a InviteKey which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteInviteKey(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerInviteKey(ctx, objID, false))
}

// deleteInnerInviteKey soft-deletes a InviteKey which is currently alive
func (s *Storage) deleteInnerInviteKey(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE invite_keys SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteInviteKey", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting InviteKey %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "InviteKey %v not found", objID)
	}
	return nil
}
