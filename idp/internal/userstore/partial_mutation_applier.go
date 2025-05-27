package userstore

import (
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

type keyedValue struct {
	key   any
	value any
}

func getKeyedValues[T any](
	ma *partialMutationApplier,
	values ...T,
) ([]keyedValue, error) {
	var keyedValues []keyedValue
	uniqueKeys := map[any]bool{}

	for _, v := range values {
		key, err := ma.column.Attributes.Constraints.GetUniqueKey(*ma.dataType, v)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if uniqueKeys[key] {
			return nil, ucerr.Friendlyf(nil, "value '%v' key '%v' is not unique", v, key)
		}
		uniqueKeys[key] = true

		keyedValues = append(keyedValues, keyedValue{key: key, value: v})
	}

	return keyedValues, nil
}

type partialMutationApplier struct {
	column                           *storage.Column
	dataType                         *column.DataType
	addedValues                      []keyedValue
	deletedValueKeys                 map[any]bool
	purposeAdditions                 []uuid.UUID
	purposeDeletions                 []uuid.UUID
	applyPurposeAdditionsToAllValues bool
	applyPurposeDeletionsToAllValues bool
}

func newPartialMutationApplier(c *storage.Column, dt *column.DataType) partialMutationApplier {
	return partialMutationApplier{
		column:           c,
		dataType:         dt,
		deletedValueKeys: map[any]bool{},
	}
}

func (ma *partialMutationApplier) applyMutation(user *storage.User) (*appliedMutation, int, error) {
	am := newAppliedMutation()

	addedValuesByKey := map[any]keyedValue{}
	for _, addedValue := range ma.addedValues {
		addedValuesByKey[addedValue.key] = addedValue
	}

	var updatedValues []any

	currentValues, found := user.ColumnValues[ma.column.Name]
	if found {
		if len(currentValues) == 0 {
			return nil,
				http.StatusInternalServerError,
				ucerr.Errorf(
					"user '%v' has no values for column '%s'",
					user.ID,
					ma.column.Name,
				)
		}

		for _, cv := range currentValues {
			currentPurposes := cv.GetPurposeIDs()
			updatedPurposes := set.NewUUIDSet(currentPurposes...)

			key, err := ma.column.Attributes.Constraints.GetUniqueKey(*ma.dataType, cv.Value)
			if err != nil {
				return nil,
					http.StatusBadRequest,
					ucerr.Wrap(err)
			}

			if ma.isDeletedValueKey(key) {
				if len(ma.purposeDeletions) == 0 {
					updatedPurposes = set.NewUUIDSet()
				} else {
					for _, deletedPurpose := range ma.purposeDeletions {
						updatedPurposes.Evict(deletedPurpose)
					}
				}
			}

			if ma.applyPurposeAdditionsToAllValues {
				updatedPurposes.Insert(ma.purposeAdditions...)
			} else {
				av, found := addedValuesByKey[key]
				if found {
					delete(addedValuesByKey, key)
					updatedPurposes.Insert(ma.purposeAdditions...)

					equivalent, err := ma.dataType.AreEquivalent(cv.Value, av.value)
					if err != nil {
						return nil, http.StatusInternalServerError, ucerr.Wrap(err)
					}

					if !equivalent {
						// make sure value change does not violate immutability

						if ma.column.Attributes.Constraints.ImmutableRequired {
							return nil,
								http.StatusBadRequest,
								ucerr.Friendlyf(
									nil,
									"cannot modify value '%v'",
									cv.Value,
								)
						}

						// add new value, retaining existing ordering and associating
						// updated purposes with the new value

						am.addedValues = append(
							am.addedValues,
							addedValue{
								value: storage.ColumnConsentedValue{
									Version:    cv.Version,
									ColumnName: ma.column.Name,
									Value:      av.value,
									Ordering:   cv.Ordering,
								},
								purposes: updatedPurposes,
							},
						)
						updatedValues = append(updatedValues, av.value)

						// remove current value

						updatedPurposes = set.NewUUIDSet()
					}
				}
			}

			if updatedPurposes.Size() > 0 {
				am.currentPurposesByID[cv.ID] = set.NewUUIDSet(currentPurposes...)
				am.updatedPurposesByID[cv.ID] = updatedPurposes
				am.currentValues[cv.ID] = cv
				updatedValues = append(updatedValues, cv.Value)
			} else {
				am.removedValues[cv.ID] = cv
			}
		}
	}

	if len(addedValuesByKey) > 0 {
		ordering := 0
		for _, currentValue := range currentValues {
			if currentValue.Ordering > ordering {
				ordering = currentValue.Ordering
			}
		}

		for _, av := range ma.addedValues {
			if _, found := addedValuesByKey[av.key]; found {
				ordering++
				am.addedValues = append(
					am.addedValues,
					addedValue{
						value: storage.ColumnConsentedValue{
							ColumnName: ma.column.Name,
							Ordering:   ordering,
							Value:      av.value,
						},
						purposes: set.NewUUIDSet(ma.purposeAdditions...),
					},
				)
				updatedValues = append(updatedValues, av.value)
			}
		}
	}

	// validate the updated values if they must be unique

	if ma.column.Attributes.Constraints.UniqueRequired {
		var cv column.Value
		if err := cv.Set(
			*ma.dataType,
			ma.column.Attributes.Constraints,
			true,
			updatedValues,
		); err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
	}

	return &am, http.StatusOK, nil
}

func (ma *partialMutationApplier) getKeyedValues(value any) ([]keyedValue, error) {
	if !ma.column.Attributes.Constraints.PartialUpdates {
		return nil,
			ucerr.Friendlyf(
				nil,
				"column '%s' does not have PartialUpdates enabled",
				ma.column.Name,
			)
	}

	dc, err := column.NewDataCoercer(*ma.dataType, ma.column.Attributes.Constraints)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var keyedValues []keyedValue

	switch dc.GetConcreteType() {
	case datatype.String.ID:
		var strings []string
		if strings, err = dc.ToStrings(value); err == nil {
			keyedValues, err = getKeyedValues(ma, strings...)
		}
	case datatype.Boolean.ID:
		var bools []bool
		if bools, err = dc.ToBools(value); err == nil {
			keyedValues, err = getKeyedValues(ma, bools...)
		}
	case datatype.Integer.ID:
		var ints []int
		if ints, err = dc.ToInts(value); err == nil {
			keyedValues, err = getKeyedValues(ma, ints...)
		}
	case datatype.Timestamp.ID:
		var timestamps []time.Time
		if timestamps, err = dc.ToTimestamps(value); err == nil {
			keyedValues, err = getKeyedValues(ma, timestamps...)
		}
	case datatype.Date.ID:
		var dates []time.Time
		if dates, err = dc.ToDates(value); err == nil {
			keyedValues, err = getKeyedValues(ma, dates...)
		}
	case datatype.UUID.ID:
		var ids []uuid.UUID
		if ids, err = dc.ToUUIDs(value); err == nil {
			keyedValues, err = getKeyedValues(ma, ids...)
		}
	case datatype.Composite.ID:
		var cvs []userstore.CompositeValue
		if cvs, err = dc.ToCompositeValues(value); err == nil {
			keyedValues, err = getKeyedValues(ma, cvs...)
		}
	}

	if err != nil {
		return nil,
			ucerr.Friendlyf(err, "invalid column: '%s', data type: '%s', isArray: %t value: '%s'",
				ma.column.Name,
				ma.dataType.Name,
				ma.column.IsArray,
				value,
			)
	}

	return keyedValues, nil
}

func (ma *partialMutationApplier) isDeletedValueKey(valueKey any) bool {
	if ma.applyPurposeDeletionsToAllValues {
		return true
	}

	return ma.deletedValueKeys[valueKey]
}

func (ma *partialMutationApplier) setMutation(cm columnMutation) error {
	if cm.valueAdditions != nil {
		ma.purposeAdditions = append(ma.purposeAdditions, cm.purposeAdditions...)

		if cm.valueAdditions == idp.MutatorColumnCurrentValue {
			ma.applyPurposeAdditionsToAllValues = true
		} else {
			addedValues, err := ma.getKeyedValues(cm.valueAdditions)
			if err != nil {
				return ucerr.Wrap(err)
			}
			ma.addedValues = addedValues
		}
	}

	if cm.valueDeletions != nil {
		ma.purposeDeletions = append(ma.purposeDeletions, cm.purposeDeletions...)

		if cm.valueDeletions == idp.MutatorColumnCurrentValue {
			ma.applyPurposeDeletionsToAllValues = true
		} else {
			deletedValues, err := ma.getKeyedValues(cm.valueDeletions)
			if err != nil {
				return ucerr.Wrap(err)
			}
			for _, deletedValue := range deletedValues {
				ma.deletedValueKeys[deletedValue.key] = true
			}
		}
	}

	return nil
}
