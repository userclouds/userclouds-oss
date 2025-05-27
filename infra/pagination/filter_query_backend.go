package pagination

import (
	"fmt"

	"userclouds.com/infra/ucerr"
)

func (fq *FilterQuery) stringValue() string {
	return fmt.Sprintf("%v", fq.value)
}

func (fq *FilterQuery) queryFields(supportedKeys KeyTypes, queryFields []any) ([]any, error) {
	if fq.value == nested {
		return fq.leftChild.queryFields(supportedKeys, queryFields)
	}

	switch op := operator(fq.stringValue()); {
	case op.isComparisonOperator():
		value, err := supportedKeys.getValidFilterExactValue(fq.leftChild.stringValue(), fq.rightChild.stringValue())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		queryFields = append(queryFields, value)
		return queryFields, nil
	case op.isPatternOperator():
		value, err := supportedKeys.getValidNonExactValue(fq.leftChild.stringValue(), fq.rightChild.stringValue())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		queryFields = append(queryFields, value)
		return queryFields, nil
	case op.isArrayOperator():
		value, err := supportedKeys.getValidArrayOperatorValue(fq.leftChild.stringValue(), fq.rightChild.stringValue())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		queryFields = append(queryFields, value)
		return queryFields, nil
	case op.isLogicalOperator():
		queryFields, err := fq.leftChild.queryFields(supportedKeys, queryFields)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return fq.rightChild.queryFields(supportedKeys, queryFields)
	default:
		return nil, ucerr.Errorf("unsupported filter query operator '%v'", op)
	}
}

func (fq *FilterQuery) queryString(paramIndex int) (string, int) {
	if fq.value == nested {
		s, paramIndex := fq.leftChild.queryString(paramIndex)
		return fmt.Sprintf("(%s)", s), paramIndex
	}

	switch op := operator(fq.stringValue()); {
	case op.isLeafOperator():
		s := fmt.Sprintf("%s %s $%d", fq.leftChild.stringValue(), op.queryString(), paramIndex)
		return s, paramIndex + 1
	case op.isLogicalOperator():
		left, paramIndex := fq.leftChild.queryString(paramIndex)
		right, paramIndex := fq.rightChild.queryString(paramIndex)
		return fmt.Sprintf("%s %s %s", left, op.queryString(), right), paramIndex
	default:
		return "", paramIndex
	}
}

// IsValid validates the parsed filter query using the specified KeyTypes
func (fq *FilterQuery) IsValid(supportedKeys KeyTypes) error {
	if fq.value == nested {
		if fq.leftChild == nil {
			return ucerr.New("nested filter query is missing leftChild")
		}

		if err := fq.leftChild.IsValid(supportedKeys); err != nil {
			return ucerr.Wrap(err)
		}

		return nil
	}

	if fq.leftChild == nil {
		return ucerr.New("query does not have a leftChild")
	}

	if fq.rightChild == nil {
		return ucerr.New("query does not have a rightChild")
	}

	switch op := operator(fq.stringValue()); {
	case op.isComparisonOperator():
		if err := supportedKeys.isValidFilterExactValue(fq.leftChild.stringValue(), fq.rightChild.stringValue()); err != nil {
			return ucerr.Errorf("leaf query key '%v' and value '%v' are invalid: '%v'", fq.leftChild.value, fq.rightChild.value, err)
		}
	case op.isPatternOperator():
		if err := supportedKeys.isValidNonExactValue(fq.leftChild.stringValue(), fq.rightChild.stringValue()); err != nil {
			return ucerr.Errorf("leaf query key '%v' and value '%v' are invalid: '%v'", fq.leftChild.value, fq.rightChild.value, err)
		}
	case op.isArrayOperator():
		if err := supportedKeys.isValidArrayOperatorValue(fq.leftChild.stringValue(), fq.rightChild.stringValue()); err != nil {
			return ucerr.Errorf("leaf query key '%v' and value '%v' are invalid: '%v'", fq.leftChild.value, fq.rightChild.value, err)
		}
	case op.isLogicalOperator():
		if err := fq.leftChild.IsValid(supportedKeys); err != nil {
			return ucerr.Wrap(err)
		}

		if err := fq.rightChild.IsValid(supportedKeys); err != nil {
			return ucerr.Wrap(err)
		}
	default:
		return ucerr.Errorf("query has unsupported operator '%v'", op)
	}

	return nil
}
