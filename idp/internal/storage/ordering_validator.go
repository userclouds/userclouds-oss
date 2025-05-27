package storage

import (
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// OrderingValidator is used to validate that a set of column consented values are all ordered properly
type OrderingValidator struct {
	dlcs                     column.DataLifeCycleState
	numValuesPerColumn       map[string]int
	uniqueOrderingsPerColumn map[string]set.Set[int]
}

// NewOrderingValidator creates a new ordering validator for the data life cycle state
func NewOrderingValidator(dlcs column.DataLifeCycleState) OrderingValidator {
	return OrderingValidator{
		dlcs:                     dlcs,
		numValuesPerColumn:       map[string]int{},
		uniqueOrderingsPerColumn: map[string]set.Set[int]{},
	}
}

// AddValue adds a column consented value to the ordering validator
func (ov *OrderingValidator) AddValue(ccv ColumnConsentedValue) {
	// we do not validate the ordering for soft-deleted values, since they
	// can include duplicates if the same value is deleted and retained
	// more than once

	if ov.dlcs == column.DataLifeCycleStateSoftDeleted {
		return
	}

	numValuesForColumn, found := ov.numValuesPerColumn[ccv.ColumnName]
	if !found {
		numValuesForColumn = 0
	}
	ov.numValuesPerColumn[ccv.ColumnName] = numValuesForColumn + 1

	uniqueOrderingsForColumn, found := ov.uniqueOrderingsPerColumn[ccv.ColumnName]
	if !found {
		uniqueOrderingsForColumn = set.NewIntSet()
	}
	uniqueOrderingsForColumn.Insert(ccv.Ordering)
	ov.uniqueOrderingsPerColumn[ccv.ColumnName] = uniqueOrderingsForColumn
}

// Validate implements Validateable
func (ov OrderingValidator) Validate() error {
	for columnName, numValuesForColumn := range ov.numValuesPerColumn {
		if ov.uniqueOrderingsPerColumn[columnName].Size() != numValuesForColumn {
			return ucerr.Errorf("column '%v' does not have unique Ordering for each value", columnName)
		}
	}

	return nil
}
