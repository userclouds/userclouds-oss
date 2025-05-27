// NOTE: automatically generated file -- DO NOT EDIT

package acmestorage

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

// IsOrderSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsOrderSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM acme_orders WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsOrderSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetOrder loads a Order by ID
func (s *Storage) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
	const q = "SELECT id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url, created FROM acme_orders WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj Order
	if err := s.db.GetContext(ctx, "GetOrder", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Order %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetOrderSoftDeleted loads a Order by ID iff it's soft-deleted
func (s *Storage) GetOrderSoftDeleted(ctx context.Context, id uuid.UUID) (*Order, error) {
	const q = "SELECT id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url, created FROM acme_orders WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Order
	if err := s.db.GetContext(ctx, "GetOrderSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Order %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetOrdersForIDs loads multiple Order for a given list of IDs
func (s *Storage) GetOrdersForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Order, error) {
	items := make([]Order, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getOrdersHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getOrdersHelperForIDs loads multiple Order for a given list of IDs from the DB
func (s *Storage) getOrdersHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Order, error) {
	const q = "SELECT id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url, created FROM acme_orders WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Order
	if err := s.db.SelectContextWithDirty(ctx, "GetOrdersForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Orders  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListOrdersPaginated loads a paginated list of Orders for the specified paginator settings
func (s *Storage) ListOrdersPaginated(ctx context.Context, p pagination.Paginator) ([]Order, *pagination.ResponseFields, error) {
	return s.listInnerOrdersPaginated(ctx, p, false)
}

// listInnerOrdersPaginated loads a paginated list of Orders for the specified paginator settings
func (s *Storage) listInnerOrdersPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Order, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url, created FROM (SELECT id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url, created FROM acme_orders WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Order
	if err := s.db.SelectContext(ctx, "ListOrdersPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveOrder saves a Order
func (s *Storage) SaveOrder(ctx context.Context, obj *Order) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerOrder(ctx, obj))
}

// SaveOrder saves a Order
func (s *Storage) saveInnerOrder(ctx context.Context, obj *Order) error {
	const q = "INSERT INTO acme_orders (id, updated, deleted, tenant_url_id, url, host, token, status, challenge_url) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, tenant_url_id = $3, url = $4, host = $5, token = $6, status = $7, challenge_url = $8 WHERE (acme_orders.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveOrder", obj, q, obj.ID, obj.Deleted, obj.TenantURLID, obj.URL, obj.Host, obj.Token, obj.Status, obj.ChallengeURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Order %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteOrder soft-deletes a Order which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteOrder(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerOrder(ctx, objID, false))
}

// deleteInnerOrder soft-deletes a Order which is currently alive
func (s *Storage) deleteInnerOrder(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE acme_orders SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteOrder", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Order %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Order %v not found", objID)
	}
	return nil
}
