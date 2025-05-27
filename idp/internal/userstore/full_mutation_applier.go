package userstore

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
)

const currentOrdering = -1

type orderedValue struct {
	value     any
	orderings []int
}

func getOrderedValues[T any](
	ma *fullMutationApplier,
	values ...T,
) (map[any]orderedValue, error) {
	orderedValues := map[any]orderedValue{}
	ordering := 0

	for _, v := range values {
		key, err := ma.dataType.GetComparableValue(v)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		ov, found := orderedValues[key]
		if !found {
			ov = orderedValue{value: v}
		}
		ordering++
		ov.orderings = append(ov.orderings, ordering)
		orderedValues[key] = ov
	}

	return orderedValues, nil
}

// fullMutationApplier is used to create an appliedMutation
// for a given User and columnMutation, determining which column
// values should be added, which column values are unchanged, and
// which column values should be removed. It also determines the
// current set of shared purposes, the updated set of purposes for
// each value, and the set of column values with updated orderings.

type fullMutationApplier struct {
	column           *storage.Column
	dataType         *column.DataType
	orderedValues    map[any]orderedValue
	purposeAdditions []uuid.UUID
	purposeDeletions []uuid.UUID
	useCurrentValue  bool
}

func newFullMutationApplier(c *storage.Column, dt *column.DataType) fullMutationApplier {
	return fullMutationApplier{
		column:        c,
		dataType:      dt,
		orderedValues: map[any]orderedValue{},
	}
}

func (ma *fullMutationApplier) applyMutation(ctx context.Context, user *storage.User) (*appliedMutation, int, error) {
	am := newAppliedMutation()

	// classify current column values and current purposes

	if currentValues, found := user.ColumnValues[ma.column.Name]; found {
		if len(currentValues) == 0 {
			return nil,
				http.StatusInternalServerError,
				ucerr.Errorf(
					"user '%v' has no values for column '%s'",
					user.ID,
					ma.column.Name,
				)
		}

		if err := ma.validateImmutability(currentValues); err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}

		for valueID, columnValue := range currentValues {
			// aggregate purposes and verify they match for each current value

			purposes := set.NewUUIDSet()
			for _, purpose := range columnValue.ConsentedPurposes {
				purposes.Insert(purpose.Purpose)
			}
			if purposes.Size() != len(columnValue.ConsentedPurposes) {
				return nil,
					http.StatusInternalServerError,
					ucerr.Errorf(
						"user '%v' has non-unique purposes for column '%s'",
						user.ID,
						ma.column.Name,
					)
			}

			// We use uuid.Nil as the index for the set of purposes shared by all values (they should all be the same for columns w/o partial updates)
			if am.currentPurposesByID[uuid.Nil].Size() == 0 {
				am.currentPurposesByID[uuid.Nil] = purposes
			} else if !am.currentPurposesByID[uuid.Nil].Equal(purposes) {
				uclog.Warningf(ctx,
					"user '%v' has non-matching purposes for column '%s', reducing to intersection",
					user.ID,
					ma.column.Name,
				)
				purposes = am.currentPurposesByID[uuid.Nil].Intersection(purposes)
				for valueID := range am.currentPurposesByID {
					am.currentPurposesByID[valueID] = purposes
				}
			}
			am.currentPurposesByID[valueID] = purposes

			// classify the current value, determining whether it should remain unchanged, be removed, or have its ordering updated

			ordering, err := ma.classifyValue(columnValue.Value)
			if err != nil {
				return nil, http.StatusInternalServerError, ucerr.Wrap(err)
			}
			switch ordering {
			case currentOrdering:
				am.currentValues[valueID] = columnValue
			case 0:
				am.removedValues[valueID] = columnValue
			default:
				if columnValue.Ordering != ordering {
					columnValue.Ordering = ordering
					am.updatedOrderingValueIDs.Insert(valueID)
				}
				am.currentValues[valueID] = columnValue
			}
		}
	}

	// add new purposes and remove deleted purposes

	for valueID, currentPurposes := range am.currentPurposesByID {
		updatedPurposes := set.NewUUIDSet(currentPurposes.Items()...)
		updatedPurposes.Insert(ma.purposeAdditions...)
		for _, deletedPurpose := range ma.purposeDeletions {
			updatedPurposes.Evict(deletedPurpose)
		}
		am.updatedPurposesByID[valueID] = updatedPurposes

		if updatedPurposes.Size() == 0 {
			// remove the current value
			if currentValue, found := am.currentValues[valueID]; found {
				am.removedValues[valueID] = currentValue
				delete(am.currentValues, valueID)
			}
		}
	}

	// uuid.Nil is the index for the set of purposes shared by all values (note they are all be the same for columns w/o partial updates)
	updatedPurposes := am.updatedPurposesByID[uuid.Nil]
	if updatedPurposes.Size() > 0 {
		am.addedValues = ma.getAddedValues(updatedPurposes)
	}

	return &am, http.StatusOK, nil
}

func (ma *fullMutationApplier) classifyValue(v any) (int, error) {
	if ma.useCurrentValue {
		return currentOrdering, nil
	}

	key, err := ma.dataType.GetComparableValue(v)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	ov, found := ma.orderedValues[key]
	if !found || len(ov.orderings) == 0 {
		return 0, nil
	}

	ordering := ov.orderings[0]
	ov.orderings = ov.orderings[1:]
	ma.orderedValues[key] = ov
	return ordering, nil
}

func (ma *fullMutationApplier) getAddedValues(purposes set.Set[uuid.UUID]) []addedValue {
	if ma.useCurrentValue {
		return nil
	}

	var addedValues []addedValue
	for _, ov := range ma.orderedValues {
		for _, ordering := range ov.orderings {
			addedValues = append(
				addedValues,
				addedValue{
					value: storage.ColumnConsentedValue{
						ColumnName: ma.column.Name,
						Ordering:   ordering,
						Value:      ov.value,
					},
					purposes: purposes,
				},
			)
		}
	}

	return addedValues
}

func (ma *fullMutationApplier) getOrderedValues(value any) (map[any]orderedValue, error) {
	dc, err := column.NewDataCoercer(*ma.dataType, ma.column.Attributes.Constraints)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var orderedValues map[any]orderedValue

	if ma.column.IsArray {
		switch dc.GetConcreteType() {
		case datatype.String.ID:
			var strings []string
			if strings, err = dc.ToStrings(value); err == nil {
				orderedValues, err = getOrderedValues(ma, strings...)
			}
		case datatype.Boolean.ID:
			var bools []bool
			if bools, err = dc.ToBools(value); err == nil {
				orderedValues, err = getOrderedValues(ma, bools...)
			}
		case datatype.Integer.ID:
			var ints []int
			if ints, err = dc.ToInts(value); err == nil {
				orderedValues, err = getOrderedValues(ma, ints...)
			}
		case datatype.Timestamp.ID:
			var timestamps []time.Time
			if timestamps, err = dc.ToTimestamps(value); err == nil {
				orderedValues, err = getOrderedValues(ma, timestamps...)
			}
		case datatype.Date.ID:
			var dates []time.Time
			if dates, err = dc.ToDates(value); err == nil {
				orderedValues, err = getOrderedValues(ma, dates...)
			}
		case datatype.UUID.ID:
			var ids []uuid.UUID
			if ids, err = dc.ToUUIDs(value); err == nil {
				orderedValues, err = getOrderedValues(ma, ids...)
			}
		case datatype.Composite.ID:
			var cvs []userstore.CompositeValue
			if cvs, err = dc.ToCompositeValues(value); err == nil {
				orderedValues, err = getOrderedValues(ma, cvs...)
			}
		}
	} else {
		switch dc.GetConcreteType() {
		case datatype.String.ID:
			var s string
			if s, err = dc.ToString(value); err == nil {
				orderedValues, err = getOrderedValues(ma, s)
			}
		case datatype.Boolean.ID:
			var b bool
			if b, err = dc.ToBool(value); err == nil {
				orderedValues, err = getOrderedValues(ma, b)
			}
		case datatype.Integer.ID:
			var i int
			if i, err = dc.ToInt(value); err == nil {
				orderedValues, err = getOrderedValues(ma, i)
			}
		case datatype.Timestamp.ID:
			var t time.Time
			if t, err = dc.ToTimestamp(value); err == nil {
				orderedValues, err = getOrderedValues(ma, t)
			}
		case datatype.Date.ID:
			var d time.Time
			if d, err = dc.ToDate(value); err == nil {
				orderedValues, err = getOrderedValues(ma, d)
			}
		case datatype.UUID.ID:
			var id uuid.UUID
			if id, err = dc.ToUUID(value); err == nil {
				orderedValues, err = getOrderedValues(ma, id)
			}
		case datatype.Composite.ID:
			var cv userstore.CompositeValue
			if cv, err = dc.ToCompositeValue(value); err == nil {
				orderedValues, err = getOrderedValues(ma, cv)
			}
		}
	}

	if err != nil {
		return nil,
			ucerr.Friendlyf(err, "invalid column: '%s', data type: '%s', isArray: %t, value: '%s'",
				ma.column.Name,
				ma.dataType.Name,
				ma.column.IsArray,
				value,
			)
	}

	return orderedValues, nil
}

func (ma *fullMutationApplier) setMutation(cm columnMutation) error {
	switch cm.value {
	case idp.MutatorColumnCurrentValue:
		// keep any current values, adjusting the purposes as requested
		ma.purposeAdditions = append(ma.purposeAdditions, cm.purposeAdditions...)
		ma.purposeDeletions = append(ma.purposeDeletions, cm.purposeDeletions...)
		ma.useCurrentValue = true
	case nil:
		// remove all specified purposes (whether additions or deletions) for any
		// existing values, effectively deleting the value for those purposes
		ma.purposeDeletions = append(ma.purposeDeletions, cm.purposeAdditions...)
		ma.purposeDeletions = append(ma.purposeDeletions, cm.purposeDeletions...)
	default:
		// apply any specified purpose adjustments and perform the requested
		// value adjustments
		ma.purposeAdditions = append(ma.purposeAdditions, cm.purposeAdditions...)
		ma.purposeDeletions = append(ma.purposeDeletions, cm.purposeDeletions...)
		orderedValues, err := ma.getOrderedValues(cm.value)
		if err != nil {
			return ucerr.Wrap(err)
		}
		ma.orderedValues = orderedValues
	}

	return nil
}

func (ma *fullMutationApplier) validateImmutability(existingValues map[uuid.UUID]storage.ColumnConsentedValue) error {
	if !ma.column.Attributes.Constraints.ImmutableRequired ||
		!ma.column.Attributes.Constraints.UniqueIDRequired ||
		len(ma.orderedValues) == 0 ||
		len(existingValues) == 0 {
		return nil
	}

	if ma.dataType.IsComposite() {
		valuesByID := map[string]any{}
		for _, orderedValue := range ma.orderedValues {
			id, err := ma.dataType.GetUniqueID(orderedValue.value)
			if err != nil {
				return ucerr.Wrap(err)
			}

			if id != "" {
				valuesByID[id] = orderedValue.value
			}
		}

		for _, existingValue := range existingValues {
			id, err := ma.dataType.GetUniqueID(existingValue.Value)
			if err != nil {
				return ucerr.Wrap(err)
			}

			if updatedValue, found := valuesByID[id]; found {
				equivalent, err := ma.dataType.AreEquivalent(updatedValue, existingValue.Value)
				if err != nil {
					return ucerr.Wrap(err)
				}

				if !equivalent {
					return ucerr.Friendlyf(nil, "cannot modify value '%v'", existingValue)
				}
			}
		}
	}

	return nil
}
