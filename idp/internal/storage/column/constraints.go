package column

import (
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// Constraints represents the data type constraints for a column
type Constraints struct {
	ImmutableRequired bool `json:"immutable_required"`
	PartialUpdates    bool `json:"partial_updates"`
	UniqueIDRequired  bool `json:"unique_id_required"`
	UniqueRequired    bool `json:"unique_required"`
}

// AreDefault returns true if constraint defaults have not been changed
func (c Constraints) AreDefault() bool {
	return !c.ImmutableRequired &&
		!c.PartialUpdates &&
		!c.UniqueIDRequired &&
		!c.UniqueRequired
}

// NewConstraintsFromClient creates constraints from a client counterpart
func NewConstraintsFromClient(cc userstore.ColumnConstraints) Constraints {
	return Constraints{
		ImmutableRequired: cc.ImmutableRequired,
		PartialUpdates:    cc.PartialUpdates,
		UniqueIDRequired:  cc.UniqueIDRequired,
		UniqueRequired:    cc.UniqueRequired,
	}
}

func (c Constraints) extraValidate() error {
	if c.PartialUpdates {
		if !c.UniqueRequired && !c.UniqueIDRequired {
			return ucerr.Friendlyf(nil, "Cannot enable partial updates unless unique values or unique IDs are required")
		}
	}

	return nil
}

//go:generate gendbjson Constraints
//go:generate genvalidate Constraints

// Equals returns true if the column constraints are equal
func (c Constraints) Equals(other Constraints) bool {
	return c.ImmutableRequired == other.ImmutableRequired &&
		c.PartialUpdates == other.PartialUpdates &&
		c.UniqueIDRequired == other.UniqueIDRequired &&
		c.UniqueRequired == other.UniqueRequired
}

// GetUniqueKey returns a key that can uniquely identify a value
func (c Constraints) GetUniqueKey(dt DataType, v any) (any, error) {
	if c.UniqueIDRequired {
		if dt.IsComposite() {
			cv, ok := v.(userstore.CompositeValue)
			if !ok {
				return nil, ucerr.Errorf("expected '%T' but got '%v'", cv, v)
			}
			id, found := cv[idField.StructName]
			if !found {
				return nil, ucerr.Errorf("id field is missing: '%v'", cv)
			}
			return id, nil
		}

		return nil, ucerr.Errorf("data type '%v' does not support UniqueIDRequired", dt)
	}

	if c.UniqueRequired {
		v, err := dt.getComparableValue(v, true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return v, nil
	}

	return nil, ucerr.New("UniqueIDRequired or UniqueRequired must be enabled")
}

// ToClient converts the constraints to their client counterpart
func (c Constraints) ToClient() userstore.ColumnConstraints {
	return userstore.ColumnConstraints{
		ImmutableRequired: c.ImmutableRequired,
		PartialUpdates:    c.PartialUpdates,
		UniqueIDRequired:  c.UniqueIDRequired,
		UniqueRequired:    c.UniqueRequired,
	}
}

// ValidateForDataType will validate the constraints for the specified data type
func (c Constraints) ValidateForDataType(dt DataType) error {
	if dt.IsComposite() {
		if c.ImmutableRequired && !c.UniqueIDRequired {
			return ucerr.Friendlyf(nil, "Cannot require immutability if unique IDs not required")
		}
		if c.UniqueIDRequired && !dt.CompositeAttributes.IncludeID {
			return ucerr.Friendlyf(nil, "Cannot require unique IDs for data type '%s'", dt.Name)
		}
	} else {
		if c.UniqueIDRequired {
			return ucerr.Friendlyf(nil, "Cannot require unique IDs for data type '%s'", dt.Name)
		}
		if c.ImmutableRequired {
			return ucerr.Friendlyf(nil, "Cannot require immutability for data type '%s'", dt.Name)
		}
	}

	return nil
}
