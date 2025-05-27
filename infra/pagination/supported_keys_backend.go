package pagination

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/uuidarray"
)

func getValidatedBoolean(s string) (bool, error) {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return b, ucerr.Wrap(err)
	}

	return b, nil
}

func getValidatedInt(s string) (int64, error) {
	i64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return i64, ucerr.Wrap(err)
	}

	return i64, nil
}

var unescapedQuotePattern = regexp.MustCompile(`([^\\]'|[^\\]")`)

func getValidatedString(s string) (string, error) {
	// we don't allow unescaped single or double quotes in the string
	if unescapedQuotePattern.MatchString(s) {
		return s, ucerr.Errorf("string '%s' cannot have any unescaped single or double-quotes", s)
	}

	return s, nil
}

func getValidatedTimestamp(s string) (time.Time, error) {
	i64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, ucerr.Wrap(err)
	}
	return time.UnixMicro(i64).UTC(), nil
}

func getValidatedUUID(uuidAsString string) (uuid.UUID, error) {
	id, err := uuid.FromString(uuidAsString)
	if err != nil {
		return id, ucerr.Wrap(err)
	}
	return id, nil
}

// getValidArrayOperatorValue returns a value appropriate for array operations
func (kt KeyTypes) getValidArrayOperatorValue(key string, value string) (any, error) {
	t, found := kt[key]
	if !found {
		return nil, ucerr.Errorf("key '%s' is unsupported", key)
	}

	switch t {
	case ArrayKeyType:
		v, err := getValidatedString(value)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return fmt.Sprintf(`"%s"`, v), nil
	case UUIDArrayKeyType:
		id, err := getValidatedUUID(value)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return uuidarray.UUIDArray{id}, nil
	case BoolKeyType:
	case IntKeyType:
	case NullableBoolKeyType:
	case NullableIntKeyType:
	case NullableStringKeyType:
	case NullableTimestampKeyType:
	case NullableUUIDKeyType:
	case StringKeyType:
	case TimestampKeyType:
	case UUIDKeyType:
	default:
		return nil, ucerr.Errorf("key '%s' has an unsupported key type '%v'", key, t)
	}

	return nil, ucerr.Errorf("key '%s' is of type '%v' which does not support array operator values", key, t)
}

func (kt KeyTypes) getValidCursorExactValue(key string, value string) (any, error) {
	exactValue, err := kt.getValidExactValue(key, value, true)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return exactValue, nil
}

// getValidExactValue returns a value appropriate for testing for an exact match
func (kt KeyTypes) getValidExactValue(key string, value string, allowNil bool) (any, error) {
	t, found := kt[key]
	if !found {
		return nil, ucerr.Errorf("key '%s' is unsupported", key)
	}

	if allowNil && kt.isNullableKey(key) && value == "" {
		return nil, nil
	}

	switch t {
	case BoolKeyType, NullableBoolKeyType:
		return getValidatedBoolean(value)
	case IntKeyType, NullableIntKeyType:
		return getValidatedInt(value)
	case StringKeyType, NullableStringKeyType:
		return getValidatedString(value)
	case TimestampKeyType, NullableTimestampKeyType:
		return getValidatedTimestamp(value)
	case UUIDKeyType, NullableUUIDKeyType:
		return getValidatedUUID(value)
	case ArrayKeyType:
	case UUIDArrayKeyType:
	default:
		return nil, ucerr.Errorf("key '%s' has an unsupported key type '%v'", key, t)
	}

	return nil, ucerr.Errorf("key '%s' is of type '%v' which does not support exact values", key, t)
}

func (kt KeyTypes) getValidFilterExactValue(key string, value string) (any, error) {
	exactValue, err := kt.getValidExactValue(key, value, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return exactValue, nil
}

// getValidNonExactValue returns a value appropriate for testing for a non-exact match;
// this is only supported for the string type, which supports pattern matching filters
func (kt KeyTypes) getValidNonExactValue(key string, value string) (any, error) {
	t, found := kt[key]
	if !found {
		return nil, ucerr.Errorf("key '%s' is unsupported", key)
	}

	switch t {
	case StringKeyType:
		return getValidatedString(value)
	case ArrayKeyType:
	case BoolKeyType:
	case IntKeyType:
	case NullableBoolKeyType:
	case NullableIntKeyType:
	case NullableStringKeyType:
	case NullableTimestampKeyType:
	case NullableUUIDKeyType:
	case TimestampKeyType:
	case UUIDArrayKeyType:
	case UUIDKeyType:
	default:
		return nil, ucerr.Errorf("key '%s' has an unsupported key type '%v'", key, t)
	}

	return nil, ucerr.Errorf("key '%s' is of type '%v' which does not support non-exact values", key, t)
}

func (kt KeyTypes) isNullableKey(key string) bool {
	t, found := kt[key]
	if !found {
		return false
	}

	switch t {
	case NullableBoolKeyType:
	case NullableIntKeyType:
	case NullableStringKeyType:
	case NullableTimestampKeyType:
	case NullableUUIDKeyType:
	default:
		return false
	}

	return true
}

func (kt KeyTypes) isValidArrayOperatorValue(key string, value string) error {
	_, err := kt.getValidArrayOperatorValue(key, value)
	return ucerr.Wrap(err)
}

func (kt KeyTypes) isValidCursorExactValue(key string, value string) error {
	_, err := kt.getValidCursorExactValue(key, value)
	return ucerr.Wrap(err)
}

func (kt KeyTypes) isValidFilterExactValue(key string, value string) error {
	_, err := kt.getValidFilterExactValue(key, value)
	return ucerr.Wrap(err)
}

func (kt KeyTypes) isValidFinalSortKey(key string) bool {
	// we require that the final sort key in a multi-key sort key is the
	// UUID id field, which all collections that can be paginated against
	// contain. This is necessary because we need a stable ordering of
	// valies to ensure that pagination completes, and the id field is
	// guaranteed to have a non-nil value and be unique for the collection.

	if key != "id" {
		return false
	}

	t, found := kt[key]
	if !found {
		return false
	}

	return t == UUIDKeyType
}

func (kt KeyTypes) isValidNonExactValue(key string, value string) error {
	_, err := kt.getValidNonExactValue(key, value)
	return ucerr.Wrap(err)
}
