package ucdb

import (
	"context"
	"fmt"
	"strings"

	"userclouds.com/infra/ucerr"
)

// BatchUpdater allows updates to be batched by a specified batch size
type BatchUpdater struct {
	placeholderTypes  []string
	batchIndex        int
	batchSize         int
	batchNumParams    int
	batchParams       [][]any
	batchPlaceholders [][]string
	expectedUpdates   int64
}

// NewBatchUpdater creates a new batch updater with the specified batch size
// and placeholder types
func NewBatchUpdater(batchSize int, placeholderTypes []string) (*BatchUpdater, error) {
	if batchSize <= 0 {
		return nil, ucerr.New("batchSize must be greater than zero")
	}

	if len(placeholderTypes) == 0 {
		return nil, ucerr.New("there must be at least one placeholder")
	}

	bu := &BatchUpdater{
		placeholderTypes: placeholderTypes,
		batchSize:        batchSize,
	}
	bu.reset()

	return bu, nil
}

// ApplyUpdates will actually execute the batched updates, resetting the updater
// after successfully completing
func (bu *BatchUpdater) ApplyUpdates(ctx context.Context, db *DB, queryName string, queryTemplate string) error {
	err := bu.applyUpdates(ctx, db, queryName, queryTemplate)
	bu.reset()
	return ucerr.Wrap(err)
}

func (bu *BatchUpdater) applyUpdates(ctx context.Context, db *DB, queryName string, queryTemplate string) error {
	if bu.expectedUpdates == 0 {
		return nil
	}

	var totalUpdates int64
	for i := range bu.batchPlaceholders {
		q := fmt.Sprintf(queryTemplate, strings.Join(bu.batchPlaceholders[i], ","))

		res, err := db.ExecContext(ctx, queryName, q, bu.batchParams[i]...)
		if err != nil {
			return ucerr.Wrap(err)
		}
		batchUpdates, err := res.RowsAffected()
		if err != nil {
			return ucerr.Wrap(err)
		}
		totalUpdates += batchUpdates
	}

	if totalUpdates != bu.expectedUpdates {
		return ucerr.Errorf("expected %d updates, but made %d", bu.expectedUpdates, totalUpdates)
	}

	return nil
}

func (bu *BatchUpdater) reset() {
	bu.batchIndex = 0
	bu.batchNumParams = 0
	bu.batchParams = [][]any{{}}
	bu.batchPlaceholders = [][]string{{}}
	bu.expectedUpdates = 0
}

// ScheduleUpdate will schedule an update for the given params
func (bu *BatchUpdater) ScheduleUpdate(params ...any) error {
	if err := bu.scheduleUpdate(params); err != nil {
		bu.reset()
		return ucerr.Wrap(err)
	}

	return nil
}

func (bu *BatchUpdater) scheduleUpdate(params []any) error {
	if len(params) != len(bu.placeholderTypes) {
		return ucerr.Errorf("expected %d params but got %d", len(bu.placeholderTypes), len(params))
	}

	if len(bu.batchPlaceholders[bu.batchIndex]) == bu.batchSize {
		bu.batchParams = append(bu.batchParams, []any{})
		bu.batchPlaceholders = append(bu.batchPlaceholders, []string{})
		bu.batchIndex++
		bu.batchNumParams = 0
	}

	placeholders := []string{}
	for i, param := range params {
		bu.batchParams[bu.batchIndex] = append(bu.batchParams[bu.batchIndex], param)
		bu.batchNumParams++
		placeholders = append(placeholders, fmt.Sprintf("$%d::%s", bu.batchNumParams, bu.placeholderTypes[i]))
	}
	bu.batchPlaceholders[bu.batchIndex] = append(bu.batchPlaceholders[bu.batchIndex], fmt.Sprintf("(%s)", strings.Join(placeholders, ",")))
	bu.expectedUpdates++

	return nil
}
