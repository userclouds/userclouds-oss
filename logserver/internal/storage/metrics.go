package storage

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/logeventmetadata"
	"userclouds.com/logserver/events"
)

const customEventRange int = 2000000

func getCustomEventCode() uclog.EventCode {
	return (1 << events.CustomEventShift) + uclog.EventCode(rand.Intn(customEventRange))
}

// getEventCode returns next available number for system events and a random number in range for custom events
func getEventCode(ctx context.Context, s *logeventmetadata.Storage, metricType *logeventmetadata.MetricMetadata) (uclog.EventCode, error) {
	if metricType.Attributes.System {
		maxcode, err := s.GetMaxEventCode(ctx)
		if err != nil {
			return 0, ucerr.Wrap(err)
		}
		return maxcode + 1, nil
	}
	return getCustomEventCode(), nil
}

// SaveMetricsMetaDataArray saves a batch of new event types, skipping over ones that already exist
func SaveMetricsMetaDataArray(ctx context.Context, metricTypes []logeventmetadata.MetricMetadata, s *logeventmetadata.Storage) error {
	if len(metricTypes) == 0 {
		return nil
	}
	// Pick random code for the new custom event which is very likely to be available or next available for system
	maxcode, err := getEventCode(ctx, s, &metricTypes[0])
	if err != nil {
		return ucerr.Wrap(err)
	}

	for i := range metricTypes {
		// If the desired code is specified try to respect it
		canChangeCode := false
		if metricTypes[i].Code == 0 {
			metricTypes[i].Code = maxcode + uclog.EventCode(i)
			canChangeCode = true
		}

		success := false

		for !success {
			if err := s.SaveMetricMetadata(ctx, &metricTypes[i]); err != nil {
				// On conflict we need to grab a new event code but we first need to make sure the type is not already defined
				if ucdb.IsUniqueViolation(err) {
					// Check if this metrics is already defined
					if m, errGet := s.GetMetricMetadataByStringID(ctx, metricTypes[i].StringID); errGet != nil {
						if errors.Is(errGet, sql.ErrNoRows) {
							if !canChangeCode {
								return ucerr.Errorf("Found event with different StringID but conflicting code - %v ", metricTypes[i])
							}
							maxcode, err := getEventCode(ctx, s, &metricTypes[i])
							if err != nil {
								return ucerr.Wrap(err)
							}
							metricTypes[i].Code = maxcode
							continue
						} else {
							// We hit some error on reading the row
							return ucerr.Wrap(err)
						}
					} else { // errGet == nil
						// Check if the entry is the same (except for the code)
						m.Code = metricTypes[i].Code
						m.BaseModel = metricTypes[i].BaseModel
						if *m != metricTypes[i] {
							// TODO deal with this scenario by passing through an update flag indicating if we should overwrite existing row
							return ucerr.Errorf("Found event with same StringID but different data old - %v vs %v", *m, metricTypes[i])
						}
						// This event is already in the table so skip it
						success = true
					}
				} else {
					return ucerr.Wrap(err)
				}
			}
			success = true
		}
	}
	return nil
}

// GetMetricsMetaDataArray reads all metric metada for a tenant
func GetMetricsMetaDataArray(ctx context.Context, s *logeventmetadata.Storage) (*[]logeventmetadata.MetricMetadata, error) {
	var allMetrics []logeventmetadata.MetricMetadata

	pager, err := logeventmetadata.NewMetricMetadataPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		pageMetrics, respFields, err := s.ListMetricMetadatasPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		allMetrics = append(allMetrics, pageMetrics...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	return &allMetrics, nil
}

// UpdateMetricsMetaDataArray updates changable fields in a metric metadata
func UpdateMetricsMetaDataArray(ctx context.Context, metricTypes []logeventmetadata.MetricMetadata, s *logeventmetadata.Storage) error {
	for _, m := range metricTypes {
		if err := s.UpdateMetricMetadata(ctx, &m); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// DeleteMetricsMetaDataArray saves a batch of new event types, skipping over ones that already exist
func DeleteMetricsMetaDataArray(ctx context.Context, metricTypes []logeventmetadata.MetricMetadata, s *logeventmetadata.Storage) error {
	for _, m := range metricTypes {
		if err := s.DeleteMetricMetadata(ctx, m.ID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
