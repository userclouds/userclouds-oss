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

	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// IsBaseUserSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *UserStorage) IsBaseUserSoftDeleted(ctx context.Context, id uuid.UUID, usePrimaryDbOnly bool) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM users WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "IsBaseUserSoftDeleted", &deleted, q, usePrimaryDbOnly || !useReplica, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetBaseUser loads a BaseUser by ID
func (s *UserStorage) GetBaseUser(ctx context.Context, id uuid.UUID, accessPrimaryDBOnly bool) (*BaseUser, error) {
	const q = "SELECT id, updated, deleted, _version, organization_id, region, created FROM users WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj BaseUser
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "GetBaseUser", &obj, q, accessPrimaryDBOnly || !useReplica, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "BaseUser %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetBaseUserSoftDeleted loads a BaseUser by ID iff it's soft-deleted
func (s *UserStorage) GetBaseUserSoftDeleted(ctx context.Context, id uuid.UUID, accessPrimaryDBOnly bool) (*BaseUser, error) {
	const q = "SELECT id, updated, deleted, _version, organization_id, region, created FROM users WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj BaseUser
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "GetBaseUserSoftDeleted", &obj, q, accessPrimaryDBOnly || !useReplica, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted BaseUser %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetBaseUsersForIDs loads multiple BaseUser for a given list of IDs
func (s *UserStorage) GetBaseUsersForIDs(ctx context.Context, errorOnMissing bool, accessPrimaryDBOnly bool, ids ...uuid.UUID) ([]BaseUser, error) {
	items := make([]BaseUser, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	dirty := accessPrimaryDBOnly || !useReplica
	if missed.Size() > 0 {
		itemsFromDB, err := s.getBaseUsersHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getBaseUsersHelperForIDs loads multiple BaseUser for a given list of IDs from the DB
func (s *UserStorage) getBaseUsersHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]BaseUser, error) {
	const q = "SELECT id, updated, deleted, _version, organization_id, region, created FROM users WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []BaseUser
	if err := s.db.SelectContextWithDirty(ctx, "GetBaseUsersForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested BaseUsers  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListBaseUsersPaginated loads a paginated list of BaseUsers for the specified paginator settings
func (s *UserStorage) ListBaseUsersPaginated(ctx context.Context, p pagination.Paginator, accessPrimaryDBOnly bool) ([]BaseUser, *pagination.ResponseFields, error) {
	return s.listInnerBaseUsersPaginated(ctx, p, false, accessPrimaryDBOnly)
}

// listInnerBaseUsersPaginated loads a paginated list of BaseUsers for the specified paginator settings
func (s *UserStorage) listInnerBaseUsersPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool, accessPrimaryDBOnly bool) ([]BaseUser, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, _version, organization_id, region, created FROM (SELECT id, updated, deleted, _version, organization_id, region, created FROM users WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []BaseUser
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.SelectContextWithDirty(ctx, "ListBaseUsersPaginated", &objsDB, q, accessPrimaryDBOnly || !useReplica, queryFields...); err != nil {
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
