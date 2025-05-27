package column

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	uctime "userclouds.com/infra/uctypes/timestamp"
)

// DataCoercer is used to coerce data to its concrete type
type DataCoercer struct {
	dataType    DataType
	constraints Constraints
	dti         *dataTypeInfo
}

// NewDataCoercer creates a new data coercer for the data type and associated constraints
func NewDataCoercer(dataType DataType, constraints Constraints) (*DataCoercer, error) {
	dti := dts.getDataType(dataType)
	if err := dti.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &DataCoercer{
		dataType:    dataType,
		constraints: constraints,
		dti:         dti,
	}, nil
}

// If type coercion fails, we try to convert the value to the specified type
// by first converting to a string and then attempting to json-unmarshal the
// string. This may be necessary because the value is a string from either the
// database or from a json-marshaled client call. Another possibility is that
// a go client caller passed in a type that has a string representation that
// could be unmarshaled into the correct struct (e.g., possibly some time variant).
func (dc DataCoercer) asString(v any, isArray bool) (string, error) {
	var s string
	if vType := reflect.TypeOf(v); vType == nil {
		if isArray {
			s = "[]"
		} else {
			s = ""
		}
	} else if vType.Kind() == reflect.Map || vType.Kind() == reflect.Slice {
		bStr, err := json.Marshal(v)
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		s = string(bStr)
	} else {
		s = fmt.Sprintf("%s", v)
	}

	if isArray && s == "" {
		s = "[]"
	}

	return s, nil
}

func (dc DataCoercer) dateFromString(s string) (time.Time, error) {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		d, err = time.Parse(time.RFC3339, s)
	}
	return dc.timeToDate(d), ucerr.Wrap(err)
}

func (dc DataCoercer) datesFromString(s string) ([]time.Time, error) {
	var dates []time.Time

	var dateStrings []string
	err := json.Unmarshal([]byte(s), &dateStrings)
	if err == nil {
		for _, ds := range dateStrings {
			var date time.Time
			date, err = time.Parse(time.DateOnly, ds)
			if err != nil {
				dates = []time.Time{}
				break
			}
			dates = append(dates, date)
		}
	}

	if err != nil {
		err = json.Unmarshal([]byte(s), &dates)
	}

	return dc.timesToDates(dates), ucerr.Wrap(err)
}

// GetConcreteType returns the concrete type for the data coercer
func (dc DataCoercer) GetConcreteType() uuid.UUID {
	return dc.dataType.ConcreteDataTypeID
}

func (dc DataCoercer) timeToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (dc DataCoercer) timesToDates(times []time.Time) []time.Time {
	dates := make([]time.Time, len(times))
	for i, t := range times {
		dates[i] = dc.timeToDate(t)
	}
	return dates
}

func populateUniqueIDs[T any](dc DataCoercer, values ...T) ([]T, error) {
	if !dc.constraints.UniqueIDRequired {
		return values, nil
	}

	uniqueIDs := set.NewStringSet()
	for _, value := range values {
		id, err := dc.dataType.GetUniqueID(value)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if id != "" {
			uniqueIDs.Insert(id)
		}
	}

	if uniqueIDs.Size() == len(values) {
		return values, nil
	}

	for i, value := range values {
		id, err := dc.dataType.GetUniqueID(value)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if id != "" {
			continue
		}

		for {
			id := uuid.Must(uuid.NewV4()).String()
			if !uniqueIDs.Contains(id) {
				v, err := dc.dataType.setUniqueID(value, id)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}

				updatedValue, ok := v.(T)
				if !ok {
					return nil,
						ucerr.Errorf(
							"updated value '%v' does not have the expected type '%T'",
							updatedValue,
							value,
						)
				}

				uniqueIDs.Insert(id)
				values[i] = updatedValue
				break
			}
		}
	}

	return values, nil
}

// ToBool attempts to coerce a value to a bool
func (dc DataCoercer) ToBool(v any) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, ucerr.Wrap(dc.validateBools(b))
	}

	var b bool
	s, err := dc.asString(v, false)
	if err != nil {
		return b, ucerr.Wrap(err)
	}

	if s != "" {
		b, err = strconv.ParseBool(s)
	}
	if err == nil {
		err = dc.validateBools(b)
	}
	return b, ucerr.Wrap(err)
}

// ToBools attempts to coerce a value to a collection of bools
func (dc DataCoercer) ToBools(v any) ([]bool, error) {
	if bools, ok := v.([]bool); ok {
		return bools, ucerr.Wrap(dc.validateBools(bools...))
	}

	var bools []bool
	s, err := dc.asString(v, true)
	if err != nil {
		return bools, ucerr.Wrap(err)
	}

	err = json.Unmarshal([]byte(s), &bools)
	if err == nil {
		err = dc.validateBools(bools...)
	}
	return bools, ucerr.Wrap(err)
}

// ToCompositeValue attempts to coerce a value to a CompositeValue
func (dc DataCoercer) ToCompositeValue(v any) (userstore.CompositeValue, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var compositeValue userstore.CompositeValue
	if err := json.Unmarshal(b, &compositeValue); err != nil {
		s, err := dc.asString(v, false)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if err = json.Unmarshal([]byte(s), &compositeValue); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	compositeValues, err := populateUniqueIDs(dc, compositeValue)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return compositeValues[0], ucerr.Wrap(dc.validateCompositeValues(compositeValues[0]))
}

// ToCompositeValues attempts to coerce a value to a collection of CompositeValues
func (dc DataCoercer) ToCompositeValues(v any) ([]userstore.CompositeValue, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var compositeValues []userstore.CompositeValue
	if err := json.Unmarshal(b, &compositeValues); err != nil {
		s, err := dc.asString(v, true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if err = json.Unmarshal([]byte(s), &compositeValues); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	compositeValues, err = populateUniqueIDs(dc, compositeValues...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return compositeValues, ucerr.Wrap(dc.validateCompositeValues(compositeValues...))
}

// ToDate attempts to coerce a value to a date
func (dc DataCoercer) ToDate(v any) (time.Time, error) {
	if timestamp, ok := v.(time.Time); ok {
		date := dc.timeToDate(timestamp)
		return date, ucerr.Wrap(dc.validateTimestamps(date))
	}

	var date time.Time
	s, err := dc.asString(v, false)
	if err != nil {
		return date, ucerr.Wrap(err)
	}

	date, err = dc.dateFromString(s)
	if err == nil {
		err = dc.validateTimestamps(date)
	}
	return date, ucerr.Wrap(err)
}

// ToDates attempts to coerce a value to a collection of dates
func (dc DataCoercer) ToDates(v any) ([]time.Time, error) {
	if timestamps, ok := v.([]time.Time); ok {
		dates := dc.timesToDates(timestamps)
		return dates, ucerr.Wrap(dc.validateTimestamps(dates...))
	}

	var dates []time.Time
	s, err := dc.asString(v, true)
	if err != nil {
		return dates, ucerr.Wrap(err)
	}

	dates, err = dc.datesFromString(s)
	if err == nil {
		err = dc.validateTimestamps(dates...)
	}
	return dates, ucerr.Wrap(err)
}

// ToInt attempts to coerce a value to an int
func (dc DataCoercer) ToInt(v any) (int, error) {
	if i, ok := v.(int); ok {
		return i, ucerr.Wrap(dc.validateInts(i))
	}

	if i, ok := v.(float64); ok {
		return int(i), ucerr.Wrap(dc.validateInts(int(i)))
	}

	var i int
	s, err := dc.asString(v, false)
	if err != nil {
		return i, ucerr.Wrap(err)
	}

	if s != "" {
		i, err = strconv.Atoi(s)
	}
	if err == nil {
		err = dc.validateInts(i)
	}
	return i, ucerr.Wrap(err)
}

// ToInts attempts to coerce a value to a collection of ints
func (dc DataCoercer) ToInts(v any) ([]int, error) {
	if ints, ok := v.([]int); ok {
		return ints, ucerr.Wrap(dc.validateInts(ints...))
	}

	if floats, ok := v.([]float64); ok {
		ints := make([]int, len(floats))
		for i, f := range floats {
			ints[i] = int(f)
		}
		return ints, ucerr.Wrap(dc.validateInts(ints...))
	}

	var ints []int
	s, err := dc.asString(v, true)
	if err != nil {
		return ints, ucerr.Wrap(err)
	}

	err = json.Unmarshal([]byte(s), &ints)
	if err == nil {
		err = dc.validateInts(ints...)
	}
	return ints, ucerr.Wrap(err)
}

// ToString attempts to coerce a value to a string
func (dc DataCoercer) ToString(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, ucerr.Wrap(dc.validateStrings(s))
	}

	s, err := dc.asString(v, false)
	if err == nil {
		err = dc.validateStrings(s)
	}
	return s, ucerr.Wrap(err)
}

// ToStrings attempts to coerce a value to a collection of strings
func (dc DataCoercer) ToStrings(v any) ([]string, error) {
	if strings, ok := v.([]string); ok {
		return strings, ucerr.Wrap(dc.validateStrings(strings...))
	}

	var strings []string
	s, err := dc.asString(v, true)
	if err != nil {
		return strings, ucerr.Wrap(err)
	}

	var objs []any
	err = json.Unmarshal([]byte(s), &objs)
	if err == nil {
		strings = make([]string, 0, len(objs))
		for _, obj := range objs {
			if s, ok := obj.(string); ok {
				strings = append(strings, s)
				continue
			}

			bs, err := json.Marshal(obj)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			strings = append(strings, string(bs))
		}
		err = dc.validateStrings(strings...)
	}
	return strings, ucerr.Wrap(err)
}

func (dc DataCoercer) toTimestamp(v any) (time.Time, error) {
	if timestamp, ok := v.(time.Time); ok {
		return timestamp, nil
	}

	var timestamp time.Time
	s, err := dc.asString(v, false)
	if err == nil {
		timestamp, err = time.Parse(time.RFC3339, s)

		if err != nil {
			const mySQLTimestampLayout = "2006-01-02 15:04:05.999"
			timestamp, err = time.Parse(mySQLTimestampLayout, s)
		}
	}

	return timestamp, ucerr.Wrap(err)
}

// ToTimestamp attempts to coerce a value to a timestamp
func (dc DataCoercer) ToTimestamp(v any) (time.Time, error) {
	timestamp, err := dc.toTimestamp(v)
	if err != nil {
		return timestamp, ucerr.Wrap(err)
	}

	timestamp = uctime.Normalize(timestamp)

	return timestamp, ucerr.Wrap(dc.validateTimestamps(timestamp))
}

func (dc DataCoercer) toTimestamps(v any) ([]time.Time, error) {
	if timestamps, ok := v.([]time.Time); ok {
		return timestamps, nil
	}

	var timestamps []time.Time
	s, err := dc.asString(v, true)
	if err == nil {
		err = json.Unmarshal([]byte(s), &timestamps)
	}

	return timestamps, ucerr.Wrap(err)
}

// ToTimestamps attempts to coerce a value to a collection of timestamps
func (dc DataCoercer) ToTimestamps(v any) ([]time.Time, error) {
	timestamps, err := dc.toTimestamps(v)
	if err != nil {
		return timestamps, ucerr.Wrap(err)
	}

	for i, timestamp := range timestamps {
		timestamps[i] = uctime.Normalize(timestamp)
	}

	return timestamps, ucerr.Wrap(dc.validateTimestamps(timestamps...))
}

// ToUUID attempts to coerce a value to a uuid
func (dc DataCoercer) ToUUID(v any) (uuid.UUID, error) {
	if id, ok := v.(uuid.UUID); ok {
		return id, ucerr.Wrap(dc.validateUUIDs(id))
	}

	var id uuid.UUID
	s, err := dc.asString(v, false)
	if err != nil {
		return id, ucerr.Wrap(err)
	}

	id, err = uuid.FromString(s)
	if err == nil {
		err = dc.validateUUIDs(id)
	}
	return id, ucerr.Wrap(err)
}

// ToUUIDs attempts to coerce a value to a collection of uuids
func (dc DataCoercer) ToUUIDs(v any) ([]uuid.UUID, error) {
	if ids, ok := v.([]uuid.UUID); ok {
		return ids, ucerr.Wrap(dc.validateUUIDs(ids...))
	}

	var ids []uuid.UUID
	s, err := dc.asString(v, true)
	if err != nil {
		return ids, ucerr.Wrap(err)
	}

	err = json.Unmarshal([]byte(s), &ids)
	if err == nil {
		err = dc.validateUUIDs(ids...)
	}
	return ids, ucerr.Wrap(err)
}

func validateValues[T any](
	valueValidatorMaker func(DataType, Constraints) func(T) error,
	dataType DataType,
	constraints Constraints,
	values []T,
) error {
	valueValidator := valueValidatorMaker(dataType, constraints)

	for _, value := range values {
		if err := valueValidator(value); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (dc DataCoercer) validateBools(bools ...bool) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.boolValidator,
			dc.dataType,
			dc.constraints,
			bools,
		),
	)
}

func (dc DataCoercer) validateCompositeValues(compositeValues ...userstore.CompositeValue) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.compositeValidator,
			dc.dataType,
			dc.constraints,
			compositeValues,
		),
	)
}

func (dc DataCoercer) validateInts(ints ...int) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.intValidator,
			dc.dataType,
			dc.constraints,
			ints,
		),
	)
}

func (dc DataCoercer) validateStrings(strings ...string) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.stringValidator,
			dc.dataType,
			dc.constraints,
			strings,
		),
	)
}

func (dc DataCoercer) validateTimestamps(timestamps ...time.Time) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.timestampValidator,
			dc.dataType,
			dc.constraints,
			timestamps,
		),
	)
}

func (dc DataCoercer) validateUUIDs(uuids ...uuid.UUID) error {
	return ucerr.Wrap(
		validateValues(
			dc.dti.uuidValidator,
			dc.dataType,
			dc.constraints,
			uuids,
		),
	)
}
