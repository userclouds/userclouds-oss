// NOTE: automatically generated file -- DO NOT EDIT

package custompages

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

// IsCustomPageSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsCustomPageSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM custom_pages WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsCustomPageSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetCustomPage loads a CustomPage by ID
func (s *Storage) GetCustomPage(ctx context.Context, id uuid.UUID) (*CustomPage, error) {
	const q = "SELECT id, updated, deleted, app_id, page_name, page_source, created FROM custom_pages WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj CustomPage
	if err := s.db.GetContext(ctx, "GetCustomPage", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "CustomPage %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetCustomPageSoftDeleted loads a CustomPage by ID iff it's soft-deleted
func (s *Storage) GetCustomPageSoftDeleted(ctx context.Context, id uuid.UUID) (*CustomPage, error) {
	const q = "SELECT id, updated, deleted, app_id, page_name, page_source, created FROM custom_pages WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj CustomPage
	if err := s.db.GetContext(ctx, "GetCustomPageSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted CustomPage %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetCustomPagesForIDs loads multiple CustomPage for a given list of IDs
func (s *Storage) GetCustomPagesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]CustomPage, error) {
	items := make([]CustomPage, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getCustomPagesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getCustomPagesHelperForIDs loads multiple CustomPage for a given list of IDs from the DB
func (s *Storage) getCustomPagesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]CustomPage, error) {
	const q = "SELECT id, updated, deleted, app_id, page_name, page_source, created FROM custom_pages WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []CustomPage
	if err := s.db.SelectContextWithDirty(ctx, "GetCustomPagesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested CustomPages  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListCustomPagesPaginated loads a paginated list of CustomPages for the specified paginator settings
func (s *Storage) ListCustomPagesPaginated(ctx context.Context, p pagination.Paginator) ([]CustomPage, *pagination.ResponseFields, error) {
	return s.listInnerCustomPagesPaginated(ctx, p, false)
}

// listInnerCustomPagesPaginated loads a paginated list of CustomPages for the specified paginator settings
func (s *Storage) listInnerCustomPagesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]CustomPage, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, app_id, page_name, page_source, created FROM (SELECT id, updated, deleted, app_id, page_name, page_source, created FROM custom_pages WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []CustomPage
	if err := s.db.SelectContext(ctx, "ListCustomPagesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveCustomPage saves a CustomPage
func (s *Storage) SaveCustomPage(ctx context.Context, obj *CustomPage) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerCustomPage(ctx, obj))
}

// SaveCustomPage saves a CustomPage
func (s *Storage) saveInnerCustomPage(ctx context.Context, obj *CustomPage) error {
	const q = "INSERT INTO custom_pages (id, updated, deleted, app_id, page_name, page_source) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, app_id = $3, page_name = $4, page_source = $5 WHERE (custom_pages.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveCustomPage", obj, q, obj.ID, obj.Deleted, obj.AppID, obj.PageName, obj.PageSource); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "CustomPage %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteCustomPage soft-deletes a CustomPage which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteCustomPage(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerCustomPage(ctx, objID, false))
}

// deleteInnerCustomPage soft-deletes a CustomPage which is currently alive
func (s *Storage) deleteInnerCustomPage(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE custom_pages SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteCustomPage", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting CustomPage %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "CustomPage %v not found", objID)
	}
	return nil
}
