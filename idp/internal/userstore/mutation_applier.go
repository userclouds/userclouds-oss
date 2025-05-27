package userstore

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

type addedValue struct {
	value    storage.ColumnConsentedValue
	purposes set.Set[uuid.UUID]
}

type appliedMutation struct {
	addedValues             []addedValue
	currentPurposesByID     map[uuid.UUID]set.Set[uuid.UUID]
	currentValues           map[uuid.UUID]storage.ColumnConsentedValue
	removedValues           map[uuid.UUID]storage.ColumnConsentedValue
	updatedOrderingValueIDs set.Set[uuid.UUID]
	updatedPurposesByID     map[uuid.UUID]set.Set[uuid.UUID]
}

func newAppliedMutation() appliedMutation {
	return appliedMutation{
		currentPurposesByID:     map[uuid.UUID]set.Set[uuid.UUID]{uuid.Nil: set.NewUUIDSet()},
		currentValues:           map[uuid.UUID]storage.ColumnConsentedValue{},
		removedValues:           map[uuid.UUID]storage.ColumnConsentedValue{},
		updatedOrderingValueIDs: set.NewUUIDSet(),
		updatedPurposesByID:     map[uuid.UUID]set.Set[uuid.UUID]{},
	}
}

func applyMutation(ctx context.Context, user *storage.User, cm columnMutation) (*appliedMutation, int, error) {
	if cm.column.Attributes.Constraints.PartialUpdates {
		ma := newPartialMutationApplier(cm.column, cm.dataType)

		if err := ma.setMutation(cm); err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}

		am, code, err := ma.applyMutation(user)
		if err != nil {
			return nil, code, ucerr.Wrap(err)
		}
		return am, http.StatusOK, nil
	}

	ma := newFullMutationApplier(cm.column, cm.dataType)

	if err := ma.setMutation(cm); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	am, code, err := ma.applyMutation(ctx, user)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}
	return am, http.StatusOK, nil
}
