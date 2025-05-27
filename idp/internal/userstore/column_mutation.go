package userstore

import (
	"context"
	"net/http"
	"sort"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/apiclient"
)

// columnMutation

type columnMutation struct {
	column           *storage.Column
	dataType         *column.DataType
	value            any
	valueAdditions   any
	valueDeletions   any
	purposeAdditions []uuid.UUID
	purposeDeletions []uuid.UUID
}

func newColumnMutation(
	column *storage.Column,
	dataType *column.DataType,
	purposeAdditions []uuid.UUID,
	purposeDeletions []uuid.UUID,
) columnMutation {
	return columnMutation{
		column:           column,
		dataType:         dataType,
		purposeAdditions: purposeAdditions,
		purposeDeletions: purposeDeletions,
	}
}

func (cm *columnMutation) getValue(
	ctx context.Context,
	tv *transformableValue,
	normalizedValues []string,
) (any, error) {
	if tv == nil {
		return nil, nil
	}

	value, err := tv.getValue(ctx, normalizedValues)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if value != nil {
		if stringVal, ok := value.(string); ok {
			value = stringVal
		}

		if value == idp.MutatorColumnDefaultValue {
			if !cm.column.HasDefaultValue() {
				return nil,
					ucerr.Friendlyf(
						nil,
						"column mutation column '%s' does not have a default value",
						cm.column.Name,
					)
			}
			value = cm.column.DefaultValue
		}
	}
	return value, nil
}

func (cm *columnMutation) Validate() error {
	if cm.column == nil {
		return ucerr.Friendlyf(nil, "column mutation column is nil")
	}

	uniquePurposeAdditions := set.NewUUIDSet(cm.purposeAdditions...)
	if len(cm.purposeAdditions) != uniquePurposeAdditions.Size() {
		return ucerr.Friendlyf(nil, "column '%s' mutation has duplicate purpose additions", cm.column.Name)
	}

	uniquePurposeDeletions := set.NewUUIDSet(cm.purposeDeletions...)
	if len(cm.purposeDeletions) != uniquePurposeDeletions.Size() {
		return ucerr.Friendlyf(nil, "column '%s' mutation has duplicate purpose deletions", cm.column.Name)
	}

	if cm.column.Attributes.Constraints.PartialUpdates {
		anyValueChange := false
		if cm.valueAdditions != nil {
			anyValueChange = true
			if cm.valueAdditions != idp.MutatorColumnCurrentValue && len(cm.purposeAdditions) == 0 {
				return ucerr.Friendlyf(nil, "column '%s' mutation has value addition with no purpose additions", cm.column.Name)
			}
		} else if len(cm.purposeAdditions) != 0 {
			return ucerr.Friendlyf(nil, "column '%s' mutation has purpose additions with no value addition", cm.column.Name)
		}

		if cm.valueDeletions != nil {
			anyValueChange = true
		} else if len(cm.purposeDeletions) != 0 {
			return ucerr.Friendlyf(nil, "column '%s' mutation has purpose deletions with no value deletion", cm.column.Name)
		}

		if !anyValueChange {
			return ucerr.Friendlyf(nil, "column '%s' mutation has no value addition or deletion", cm.column.Name)
		}
	} else if uniquePurposeAdditions.Intersection(uniquePurposeDeletions).Size() != 0 {
		return ucerr.Friendlyf(nil, "column '%s' mutation is trying to add and delete the same purpose", cm.column.Name)
	}

	return nil
}

func newNormalizableValue(
	ctx context.Context,
	dtm *storage.DataTypeManager,
	c storage.Column,
	n storage.Transformer,
	value any,
	normalizerParams []tokenizer.ExecuteTransformerParameters,
) (*transformableValue, []tokenizer.ExecuteTransformerParameters, error) {
	nv, err := newNormalizableInputValue(
		ctx,
		dtm,
		c,
		n,
		c.IsArray,
		value,
	)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if nv.shouldTransform {
		inputs, err := nv.getTransformableInputs(ctx)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		for _, input := range inputs {
			normalizerParams = append(
				normalizerParams,
				tokenizer.ExecuteTransformerParameters{
					Transformer: &n,
					Data:        input,
				},
			)
			nv.addValueIndex(len(normalizerParams) - 1)
		}
	}

	return nv, normalizerParams, nil
}

type mutationValue struct {
	value          *transformableValue
	valueAdditions *transformableValue
	valueDeletions *transformableValue
}

func newMutationValue(
	ctx context.Context,
	dtm *storage.DataTypeManager,
	c storage.Column,
	n storage.Transformer,
	vp idp.ValueAndPurposes,
	normalizerParams []tokenizer.ExecuteTransformerParameters,
) (*mutationValue, []tokenizer.ExecuteTransformerParameters, error) {
	if err := c.ValidateMutation(vp); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if c.Attributes.Constraints.PartialUpdates {
		valueAdditions, normalizerParams, err :=
			newNormalizableValue(ctx, dtm, c, n, vp.ValueAdditions, normalizerParams)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		valueDeletions, normalizerParams, err :=
			newNormalizableValue(ctx, dtm, c, n, vp.ValueDeletions, normalizerParams)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		mv := mutationValue{
			valueAdditions: valueAdditions,
			valueDeletions: valueDeletions,
		}

		return &mv, normalizerParams, nil
	}

	value, normalizerParams, err := newNormalizableValue(ctx, dtm, c, n, vp.Value, normalizerParams)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	mv := mutationValue{
		value: value,
	}

	return &mv, normalizerParams, nil
}

func (mv mutationValue) Validate() error {
	if mv.value != nil {
		if err := mv.value.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if mv.valueAdditions != nil {
		if err := mv.valueAdditions.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if mv.valueDeletions != nil {
		if err := mv.valueDeletions.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func getColumnMutations(
	ctx context.Context,
	s *storage.Storage,
	columns []storage.Column,
	m *storage.Mutator,
	req idp.ExecuteMutatorRequest,
) ([]columnMutation, int, error) {
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	purposeCache, err := newPurposeCache(ctx, s)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	normalizerMap, err := s.GetTransformersMap(ctx, m.NormalizerIDs)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	var columnMutations []columnMutation
	var mutationValues []mutationValue
	var normalizerParams []tokenizer.ExecuteTransformerParameters

	for i, columnID := range m.ColumnIDs {
		var mutationColumn *storage.Column
		for _, column := range columns {
			if column.ID == columnID {
				if column.Attributes.Immutable {
					return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "column '%s' is immutable", column.Name)
				}
				mutationColumn = &column
				break
			}
		}
		if mutationColumn == nil {
			logMutatorConfigError(ctx, m.ID, m.Version)
			return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "column with ID '%v' does not exist", columnID)
		}

		mutation, found := req.RowData[mutationColumn.Name]
		if !found {
			keys := make([]string, 0, len(req.RowData))
			for k := range req.RowData {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return nil,
				http.StatusBadRequest,
				ucerr.Friendlyf(
					nil,
					"request RowData map missing column %s (%s): got keys %v",
					columnID,
					mutationColumn.Name,
					keys,
				)
		}

		normalizer := normalizerMap[m.NormalizerIDs[i]]

		mutationValue, mutationNormalizerParams, err :=
			newMutationValue(ctx, dtm, *mutationColumn, *normalizer, mutation, normalizerParams)
		if err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}

		if err := mutationValue.Validate(); err != nil {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}

		mutationValues = append(mutationValues, *mutationValue)
		normalizerParams = mutationNormalizerParams

		purposeAdditions := []uuid.UUID{}
		for _, purposeAddition := range mutation.PurposeAdditions {
			id, err := purposeCache.getPurposeID(purposeAddition)
			if err != nil {
				return nil, http.StatusBadRequest, ucerr.Wrap(err)
			}
			purposeAdditions = append(purposeAdditions, id)
		}

		purposeDeletions := []uuid.UUID{}
		for _, purposeDeletion := range mutation.PurposeDeletions {
			id, err := purposeCache.getPurposeID(purposeDeletion)
			if err != nil {
				return nil, http.StatusBadRequest, ucerr.Wrap(err)
			}
			purposeDeletions = append(purposeDeletions, id)
		}

		mutationDataType := dtm.GetDataTypeByID(mutationColumn.DataTypeID)
		if mutationDataType == nil {
			return nil,
				http.StatusInternalServerError,
				ucerr.Friendlyf(
					nil,
					"column '%s' has invalid data type '%v'",
					mutationColumn.Name,
					mutationColumn.DataTypeID,
				)
		}

		columnMutations = append(
			columnMutations,
			newColumnMutation(mutationColumn, mutationDataType, purposeAdditions, purposeDeletions),
		)
	}

	if len(columnMutations) != len(req.RowData) {
		return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "request RowData has the wrong number of columns (%d) for mutator (%d)", len(req.RowData), len(columnMutations))
	}

	var normalizedValues []string
	if len(normalizerParams) > 0 {
		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}

		te := tokenizer.NewTransformerExecutor(s, authzClient)
		defer te.CleanupExecution()
		normalizedValues, _, err = te.Execute(ctx, normalizerParams...)
		if err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
	}

	for i, mv := range mutationValues {
		cm := columnMutations[i]

		value, err := cm.getValue(ctx, mv.value, normalizedValues)
		if err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
		cm.value = value

		valueAdditions, err := cm.getValue(ctx, mv.valueAdditions, normalizedValues)
		if err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
		cm.valueAdditions = valueAdditions

		valueDeletions, err := cm.getValue(ctx, mv.valueDeletions, normalizedValues)
		if err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
		cm.valueDeletions = valueDeletions

		if err := cm.Validate(); err != nil {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}

		columnMutations[i] = cm
	}

	return columnMutations, http.StatusOK, nil
}
