package userstore

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// retentionCache handles looking up retention and deletion durations for a given tenant, region,
// column, and purpose. The cache is meant to be use for the life of a single mutation, and will
// generate appropriate retention and deletion timeouts based on the configured durations.

type retentionCache struct {
	s                  *storage.Storage
	tenantID           uuid.UUID
	regionID           uuid.UUID
	baseTime           time.Time
	retentionDurations map[string]idp.RetentionDuration
	retentionTimeouts  map[string]time.Time
}

func newRetentionCache(s *storage.Storage, tenantID uuid.UUID, regionID uuid.UUID, baseTime time.Time) retentionCache {
	return retentionCache{
		s:                  s,
		tenantID:           tenantID,
		regionID:           regionID,
		baseTime:           baseTime,
		retentionDurations: map[string]idp.RetentionDuration{},
		retentionTimeouts:  map[string]time.Time{},
	}
}

func (rc *retentionCache) getDeletionTimeout(
	ctx context.Context,
	columnID uuid.UUID,
	purposeID uuid.UUID,
) (time.Time, error) {
	key := rc.getKey(column.DataLifeCycleStateSoftDeleted, columnID, purposeID)
	timeout, found := rc.retentionTimeouts[key]
	if !found {
		timeout = userstore.DataLifeCycleStateSoftDeleted.GetDefaultRetentionTimeout()
		duration, err := rc.getRetentionDuration(ctx, column.DataLifeCycleStateSoftDeleted, columnID, purposeID)
		if err != nil {
			return timeout, ucerr.Wrap(err)
		}

		if duration != storage.RetentionDurationImmediateDeletion {
			timeout = duration.AddToTime(rc.baseTime)
		}
		rc.retentionTimeouts[key] = timeout
	}

	return timeout, nil
}

func (rc retentionCache) getKey(durationType column.DataLifeCycleState, columnID uuid.UUID, purposeID uuid.UUID) string {
	return fmt.Sprintf("%v|%v|%v", durationType, columnID, purposeID)
}

func (rc *retentionCache) getRetentionDuration(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	purposeID uuid.UUID,
) (idp.RetentionDuration, error) {
	if err := rc.initializeRetentionDurations(ctx); err != nil {
		return storage.RetentionDurationImmediateDeletion, ucerr.Wrap(err)
	}

	rd, found := rc.retentionDurations[rc.getKey(durationType, columnID, purposeID)]
	if !found {
		rd, found = rc.retentionDurations[rc.getKey(durationType, uuid.Nil, purposeID)]
		if !found {
			rd, found = rc.retentionDurations[rc.getKey(durationType, uuid.Nil, uuid.Nil)]
			if !found {
				return storage.RetentionDurationImmediateDeletion, ucerr.New("no default retention duration found")
			}
		}
	}

	return rd, nil
}

func (rc *retentionCache) getRetentionTimeout(
	ctx context.Context,
	columnID uuid.UUID,
	purposeID uuid.UUID,
) (time.Time, error) {
	key := rc.getKey(column.DataLifeCycleStateLive, columnID, purposeID)
	timeout, found := rc.retentionTimeouts[key]
	if !found {
		duration, err := rc.getRetentionDuration(ctx, column.DataLifeCycleStateLive, columnID, purposeID)
		if err != nil {
			return userstore.DataLifeCycleStateLive.GetDefaultRetentionTimeout(), ucerr.Wrap(err)
		}

		timeout = duration.AddToTime(rc.baseTime)
		rc.retentionTimeouts[key] = timeout
	}

	return timeout, nil
}

func (rc *retentionCache) initializeRetentionDurations(ctx context.Context) error {
	if len(rc.retentionDurations) > 0 {
		return nil
	}

	pager, err := storage.NewColumnValueRetentionDurationPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}

	rc.retentionDurations[rc.getKey(column.DataLifeCycleStateLive, uuid.Nil, uuid.Nil)] =
		storage.GetDefaultRetentionDuration(column.DataLifeCycleStateLive)
	rc.retentionDurations[rc.getKey(column.DataLifeCycleStateSoftDeleted, uuid.Nil, uuid.Nil)] =
		storage.GetDefaultRetentionDuration(column.DataLifeCycleStateSoftDeleted)

	for {
		rds, respFields, err :=
			rc.s.ListColumnValueRetentionDurationsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, rd := range rds {
			rc.retentionDurations[rc.getKey(rd.DurationType, rd.ColumnID, rd.PurposeID)] =
				idp.RetentionDuration{
					Unit:     rd.DurationUnit.ToClient(),
					Duration: rd.Duration,
				}
		}

		if !pager.AdvanceCursor(*respFields) {
			return nil
		}
	}
}
