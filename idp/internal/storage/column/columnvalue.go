package column

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Value is a helper struct that holds a userstore.Column's value, which can be passed in as any type but needs to
// be retrievable as the correct type for the storage layer.
type Value struct {
	dataType    DataType
	constraints Constraints
	isArray     bool
	initialized bool

	str            string
	strArray       []string
	b              bool
	bArray         []bool
	i              int
	iArray         []int
	t              time.Time
	tArray         []time.Time
	uuid           uuid.UUID
	uuidArray      []uuid.UUID
	composite      userstore.CompositeValue
	compositeArray []userstore.CompositeValue
}

// Get returns the Value's value as the correct type for the storage layer.
func (cv Value) Get(ctx context.Context) any {
	if !cv.initialized {
		uclog.Errorf(ctx, "call to Value.Get() on uninitialized Value")
		return nil
	}

	if cv.isArray {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			return cv.strArray
		case datatype.Boolean.ID:
			return cv.bArray
		case datatype.Timestamp.ID, datatype.Date.ID:
			return cv.tArray
		case datatype.Integer.ID:
			return cv.iArray
		case datatype.UUID.ID:
			return cv.uuidArray
		case datatype.Composite.ID:
			return cv.compositeArray
		}
	} else {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			return cv.str
		case datatype.Boolean.ID:
			return cv.b
		case datatype.Timestamp.ID, datatype.Date.ID:
			return cv.t
		case datatype.Integer.ID:
			return cv.i
		case datatype.UUID.ID:
			return cv.uuid
		case datatype.Composite.ID:
			return cv.composite
		}
	}

	// should never get here
	return nil
}

// GetStrings returns a slice of string representations of the Value's value
func (cv Value) GetStrings(ctx context.Context) ([]string, error) {
	if !cv.initialized {
		uclog.Errorf(ctx, "call to Value.GetStrings() on uninitialized Value")
		return nil, nil
	}

	var strs []string

	if cv.isArray {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			strs = append(strs, cv.strArray...)
		case datatype.Boolean.ID:
			for _, b := range cv.bArray {
				strs = append(strs, strconv.FormatBool(b))
			}
		case datatype.Integer.ID:
			for _, i := range cv.iArray {
				strs = append(strs, strconv.Itoa(i))
			}
		case datatype.Timestamp.ID:
			for _, t := range cv.tArray {
				strs = append(strs, timeToString(t))
			}
		case datatype.Date.ID:
			for _, t := range cv.tArray {
				strs = append(strs, dateToString(t))
			}
		case datatype.UUID.ID:
			for _, id := range cv.uuidArray {
				strs = append(strs, id.String())
			}
		case datatype.Composite.ID:
			for _, c := range cv.compositeArray {
				bStr, err := json.Marshal(c)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				strs = append(strs, string(bStr))
			}
		}
	} else {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			strs = append(strs, cv.str)
		case datatype.Boolean.ID:
			strs = append(strs, strconv.FormatBool(cv.b))
		case datatype.Integer.ID:
			strs = append(strs, strconv.Itoa(cv.i))
		case datatype.Timestamp.ID:
			strs = append(strs, timeToString(cv.t))
		case datatype.Date.ID:
			strs = append(strs, dateToString(cv.t))
		case datatype.UUID.ID:
			strs = append(strs, cv.uuid.String())
		case datatype.Composite.ID:
			bStr, err := json.Marshal(cv.composite)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			strs = append(strs, string(bStr))
		}
	}

	return strs, nil
}

// GetString returns a string representation of the Value's value
func (cv Value) GetString(ctx context.Context) (string, error) {
	if !cv.initialized {
		uclog.Errorf(ctx, "call to Value.GetString() on uninitialized Value")
		return "", nil
	}

	if cv.isArray {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			bStr, err := json.Marshal(cv.strArray)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.Boolean.ID:
			bStr, err := json.Marshal(cv.bArray)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.Integer.ID:
			bStr, err := json.Marshal(cv.iArray)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.Timestamp.ID:
			bStr, err := json.Marshal(timesToStrings(cv.tArray))
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.Date.ID:
			bStr, err := json.Marshal(datesToStrings(cv.tArray))
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.UUID.ID:
			bStr, err := json.Marshal(cv.uuidArray)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		case datatype.Composite.ID:
			bStr, err := json.Marshal(cv.compositeArray)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		}
	} else {
		switch cv.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			return cv.str, nil
		case datatype.Boolean.ID:
			return strconv.FormatBool(cv.b), nil
		case datatype.Integer.ID:
			return strconv.Itoa(cv.i), nil
		case datatype.Timestamp.ID:
			return timeToString(cv.t), nil
		case datatype.Date.ID:
			return dateToString(cv.t), nil
		case datatype.UUID.ID:
			return cv.uuid.String(), nil
		case datatype.Composite.ID:
			bStr, err := json.Marshal(cv.composite)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			return string(bStr), nil
		}
	}

	// should never get here
	return "", nil
}

// GetAnyArray returns an array of type any for an array Value.
func (cv Value) GetAnyArray(ctx context.Context) []any {
	if !cv.initialized {
		uclog.Errorf(ctx, "call to Value.GetAnyArray() on uninitialized Value")
		return nil
	}

	if !cv.isArray {
		uclog.Errorf(ctx, "call to Value.GetAnyArray() on non-array Value")
		return nil
	}

	switch cv.dataType.ConcreteDataTypeID {
	case datatype.String.ID:
		ret := make([]any, len(cv.strArray))
		for i, v := range cv.strArray {
			ret[i] = v
		}
		return ret
	case datatype.Boolean.ID:
		ret := make([]any, len(cv.bArray))
		for i, v := range cv.bArray {
			ret[i] = v
		}
		return ret
	case datatype.Integer.ID:
		ret := make([]any, len(cv.iArray))
		for i, v := range cv.iArray {
			ret[i] = v
		}
		return ret
	case datatype.Timestamp.ID, datatype.Date.ID:
		ret := make([]any, len(cv.tArray))
		for i, v := range cv.tArray {
			ret[i] = v
		}
		return ret
	case datatype.UUID.ID:
		ret := make([]any, len(cv.uuidArray))
		for i, v := range cv.uuidArray {
			ret[i] = v
		}
		return ret
	case datatype.Composite.ID:
		ret := make([]any, len(cv.compositeArray))
		for i, v := range cv.compositeArray {
			ret[i] = v
		}
		return ret
	}

	// should never get here
	return nil
}

func (cv *Value) setFieldValue(
	dataTypeID uuid.UUID,
	v any,
) error {
	dataType, err := GetNativeDataType(dataTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := cv.set(*dataType, Constraints{}, false, v, false); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// Set sets the Value's value to the given value, which can be passed in as any type
func (cv *Value) Set(
	dataType DataType,
	constraints Constraints,
	isArray bool,
	v any,
) error {
	return ucerr.Wrap(
		cv.set(
			dataType,
			constraints,
			isArray,
			v,
			true,
		),
	)
}

func (cv *Value) set(
	dataType DataType,
	constraints Constraints,
	isArray bool,
	v any,
	validate bool,
) error {
	if err := cv.setType(dataType, constraints, isArray, validate); err != nil {
		return ucerr.Wrap(err)
	}

	dc, err := NewDataCoercer(dataType, cv.constraints)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if isArray {
		switch dc.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			cv.strArray, err = dc.ToStrings(v)
		case datatype.Boolean.ID:
			cv.bArray, err = dc.ToBools(v)
		case datatype.Integer.ID:
			cv.iArray, err = dc.ToInts(v)
		case datatype.Timestamp.ID:
			cv.tArray, err = dc.ToTimestamps(v)
		case datatype.Date.ID:
			cv.tArray, err = dc.ToDates(v)
		case datatype.UUID.ID:
			cv.uuidArray, err = dc.ToUUIDs(v)
		case datatype.Composite.ID:
			cv.compositeArray, err = dc.ToCompositeValues(v)
		}
	} else {
		switch dc.dataType.ConcreteDataTypeID {
		case datatype.String.ID:
			cv.str, err = dc.ToString(v)
		case datatype.Boolean.ID:
			cv.b, err = dc.ToBool(v)
		case datatype.Integer.ID:
			cv.i, err = dc.ToInt(v)
		case datatype.Timestamp.ID:
			cv.t, err = dc.ToTimestamp(v)
		case datatype.Date.ID:
			cv.t, err = dc.ToDate(v)
		case datatype.UUID.ID:
			cv.uuid, err = dc.ToUUID(v)
		case datatype.Composite.ID:
			cv.composite, err = dc.ToCompositeValue(v)
		}
	}

	if err != nil {
		return ucerr.Friendlyf(err, "data type: %v, constraints: %v, isArray: %t, value: %s", dataType, constraints, isArray, v)
	}

	return nil
}

// SetType sets the Value's type to the given type, used for array columns
func (cv *Value) SetType(
	dataType DataType,
	constraints Constraints,
	isArray bool,
) error {
	return ucerr.Wrap(cv.setType(dataType, constraints, isArray, true))
}

func (cv *Value) setType(
	dataType DataType,
	constraints Constraints,
	isArray bool,
	validate bool,
) error {
	if validate {
		if err := dataType.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if err := constraints.ValidateForDataType(dataType); err != nil {
			return ucerr.Wrap(err)
		}
	}

	cv.constraints = constraints
	cv.dataType = dataType
	cv.isArray = isArray
	cv.initialized = true

	return nil
}

// Append appends the given value to the Value's array value
func (cv *Value) Append(ctx context.Context, v any) error {
	if !cv.initialized {
		uclog.Errorf(ctx, "call to Value.Append() on uninitialized Value")
		return nil
	}

	if !cv.isArray {
		return ucerr.New("call to Value.Append() on non-array Value")
	}

	var _cv Value
	if err := _cv.set(cv.dataType, cv.constraints, false, v, false); err != nil {
		return ucerr.Wrap(err)
	}

	switch cv.dataType.ConcreteDataTypeID {
	case datatype.String.ID:
		cv.strArray = append(cv.strArray, _cv.Get(ctx).(string))
	case datatype.Boolean.ID:
		cv.bArray = append(cv.bArray, _cv.Get(ctx).(bool))
	case datatype.Integer.ID:
		cv.iArray = append(cv.iArray, _cv.Get(ctx).(int))
	case datatype.Timestamp.ID, datatype.Date.ID:
		cv.tArray = append(cv.tArray, _cv.Get(ctx).(time.Time))
	case datatype.UUID.ID:
		cv.uuidArray = append(cv.uuidArray, _cv.Get(ctx).(uuid.UUID))
	case datatype.Composite.ID:
		cv.compositeArray = append(cv.compositeArray, _cv.Get(ctx).(userstore.CompositeValue))
	}

	return nil
}
