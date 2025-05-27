package userstore

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type transformableValue struct {
	columnName          string
	shouldTransform     bool
	transformType       policy.TransformType
	emptyConstraints    column.Constraints
	inputDataType       column.DataType
	outputDataType      column.DataType
	isArrayValue        bool
	value               any
	valueIndices        []int
	transformableInputs []string
}

func getDefaultDataTypes(
	dtm *storage.DataTypeManager,
	c storage.Column,
) (columnDataType *column.DataType, inputDataType *column.DataType, outputDataType *column.DataType, err error) {
	dataType := getTransformerDataType(dtm, c.DataTypeID)
	if dataType == nil {
		return nil, nil, nil, ucerr.Errorf("column '%s' has invalid data type '%v'", c.Name, c.DataTypeID)
	}

	return dataType, dataType, dataType, nil
}

func getTransformerDataType(
	dtm *storage.DataTypeManager,
	dataTypeID uuid.UUID,
) *column.DataType {
	dt := dtm.GetDataTypeByID(dataTypeID)
	if dt != nil {
		dt = dt.GetTransformerDataType()
	}

	return dt
}

func newNormalizableInputValue(
	ctx context.Context,
	dtm *storage.DataTypeManager,
	c storage.Column,
	normalizer storage.Transformer,
	isArrayValue bool,
	value any,
) (*transformableValue, error) {
	transformType := normalizer.TransformType.ToClient()

	columnDataType, inputDataType, outputDataType, err := getDefaultDataTypes(dtm, c)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	shouldTransform :=
		value != nil &&
			value != idp.MutatorColumnCurrentValue &&
			value != idp.MutatorColumnDefaultValue &&
			transformType != policy.TransformTypePassThrough

	if shouldTransform {
		inputDataType = getTransformerDataType(dtm, normalizer.InputDataTypeID)
		if inputDataType == nil {
			return nil,
				ucerr.Errorf(
					"normalizer '%s' has invalid input data type '%v'",
					normalizer.Name,
					normalizer.InputDataTypeID,
				)
		}

		if !normalizer.CanOutput(*columnDataType) {
			uclog.Warningf(
				ctx,
				"normalizer output data type '%v' does not match column data type '%s', using column data type",
				normalizer.OutputDataTypeID,
				columnDataType.Name,
			)
		}
	}

	return &transformableValue{
		columnName:      c.Name,
		shouldTransform: shouldTransform,
		transformType:   transformType,
		inputDataType:   *inputDataType,
		outputDataType:  *outputDataType,
		isArrayValue:    isArrayValue,
		value:           value,
		valueIndices:    []int{},
	}, nil
}

func newTransformableOutputValue(
	ctx context.Context,
	dtm *storage.DataTypeManager,
	col storage.Column,
	transformer storage.Transformer,
	isArrayValue bool,
	value any,
) (*transformableValue, error) {
	transformType := transformer.TransformType.ToClient()

	columnDataType, inputDataType, outputDataType, err := getDefaultDataTypes(dtm, col)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	shouldTransform :=
		value != nil &&
			transformType != policy.TransformTypePassThrough

	if shouldTransform {
		outputDataType = getTransformerDataType(dtm, transformer.OutputDataTypeID)
		if outputDataType == nil {
			return nil,
				ucerr.Errorf(
					"transformer '%s' has invalid output data type '%v'",
					transformer.Name,
					transformer.OutputDataTypeID,
				)
		}

		if transformer.CanInput(*columnDataType) {
			inputDataType = getTransformerDataType(dtm, transformer.InputDataTypeID)
			if inputDataType == nil {
				return nil,
					ucerr.Errorf(
						"transformer '%s' has invalid input data type '%v'",
						transformer.Name,
						transformer.InputDataTypeID,
					)
			}
		} else {
			uclog.Warningf(
				ctx,
				"transformer input data type '%v' cannot represent column data type '%s', using column data type",
				transformer.InputDataTypeID,
				columnDataType.Name,
			)
		}
	}

	return &transformableValue{
		columnName:      col.Name,
		shouldTransform: shouldTransform,
		transformType:   transformType,
		inputDataType:   *inputDataType,
		outputDataType:  *outputDataType,
		isArrayValue:    isArrayValue,
		value:           value,
		valueIndices:    []int{},
	}, nil
}

func (tv *transformableValue) addValueIndex(valueIndex int) {
	tv.valueIndices = append(tv.valueIndices, valueIndex)
}

func (tv *transformableValue) getTransformableInputs(ctx context.Context) ([]string, error) {
	var cv column.Value
	if err := cv.Set(tv.inputDataType, tv.emptyConstraints, tv.isArrayValue, tv.value); err != nil {
		return nil, ucerr.Wrap(err)
	}

	strs, err := cv.GetStrings(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if tv.transformType == policy.TransformTypeTokenizeByReference {
		// for tokenize by reference, we only want to transform one value, since
		// the resulting token represents a reference to all values associated with
		// data provenance
		strs = []string{strs[0]}
	}

	tv.transformableInputs = strs
	return strs, nil
}

func (tv transformableValue) getValue(ctx context.Context, transformedValues []string) (any, error) {
	var cv column.Value
	if tv.shouldTransform {
		if tv.transformType == policy.TransformTypeTokenizeByReference {
			// for tokenize by reference, we will only have transformed one value,
			// regardless of whether the transformable value was an array value
			vi := tv.valueIndices[0]
			if vi < 0 || vi >= len(transformedValues) {
				return nil,
					ucerr.Errorf(
						"invalid value index %d for column %s and %d transformed values",
						vi,
						tv.columnName,
						len(transformedValues),
					)
			}

			if err := cv.Set(tv.outputDataType, tv.emptyConstraints, false, transformedValues[vi]); err != nil {
				return nil, ucerr.Wrap(err)
			}
		} else if tv.isArrayValue {
			if err := cv.SetType(tv.outputDataType, tv.emptyConstraints, tv.isArrayValue); err != nil {
				return nil, ucerr.Wrap(err)
			}
			for i, vi := range tv.valueIndices {
				if vi < 0 || vi >= len(transformedValues) {
					return nil,
						ucerr.Errorf(
							"invalid value index %d for column %s and %d transformed values",
							vi,
							tv.columnName,
							len(transformedValues),
						)
				}

				// ignore transformed value if it has been filtered
				if tv.transformableInputs[i] != "" && transformedValues[vi] == "" {
					continue
				}

				if err := cv.Append(ctx, transformedValues[vi]); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
		} else {
			vi := tv.valueIndices[0]
			if vi < 0 || vi >= len(transformedValues) {
				return nil,
					ucerr.Errorf(
						"invalid value index %d for column %s and %d transformed values",
						vi,
						tv.columnName,
						len(transformedValues),
					)
			}

			if err := cv.Set(tv.outputDataType, tv.emptyConstraints, tv.isArrayValue, transformedValues[vi]); err != nil {
				return nil, ucerr.Wrap(err)
			}
		}
	} else if tv.value == nil ||
		tv.value == idp.MutatorColumnCurrentValue ||
		tv.value == idp.MutatorColumnDefaultValue {
		return tv.value, nil
	} else if err := cv.Set(tv.outputDataType, tv.emptyConstraints, tv.isArrayValue, tv.value); err != nil {
		return nil, ucerr.Wrap(err)
	}

	str, err := cv.GetString(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return str, nil
}

// Validate implements the Validateable interface
func (tv *transformableValue) Validate() error {
	if tv.columnName == "" {
		return ucerr.New("columnName must be non-empty")
	}

	if err := tv.inputDataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := tv.outputDataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if tv.shouldTransform {
		if len(tv.valueIndices) == 0 {
			return ucerr.Errorf("no values to transform for column %s", tv.columnName)
		}

		if !tv.isArrayValue && len(tv.valueIndices) != 1 {
			return ucerr.Errorf("%d values to transform for non-array column %s", len(tv.valueIndices), tv.columnName)
		}
	} else if len(tv.valueIndices) != 0 {
		return ucerr.Errorf("%d values to transform for non-transformable column %s", len(tv.valueIndices), tv.columnName)
	}

	return nil
}
