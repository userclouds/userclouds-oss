package pagination

import (
	"userclouds.com/infra/ucerr"
)

// Cursor is an opaque string that represents a place to start iterating from.
type Cursor string

// Cursor sentinel values.
const (
	CursorBegin Cursor = ""    // Default cursor value which indicates the beginning of a collection
	CursorEnd   Cursor = "end" // Special cursor value which indicates the end of the collection
)

// Direction indicates that results should be fetched forward through the view starting after the cursor
// (not including the cursor), or backward up to (but not including) the cursor.
type Direction int

// Direction can either be forward or backward.
const (
	DirectionForward  Direction = 0 // Default
	DirectionBackward Direction = 1
)

// Key is a comma-separated list of fields in the collection in which a view can be sorted
type Key string

// KeyType represents the type of a key; cursor keys, ordering keys, and filter keys must all be of a supported type
type KeyType string

// Valid KeyTypes
const (
	ArrayKeyType             KeyType = "Array"
	BoolKeyType              KeyType = "Boolean"
	IntKeyType               KeyType = "Int"
	NullableBoolKeyType      KeyType = "NullableBoolean"
	NullableIntKeyType       KeyType = "NullableInt"
	NullableStringKeyType    KeyType = "NullableString"
	NullableTimestampKeyType KeyType = "NullableTimestamp"
	NullableUUIDKeyType      KeyType = "NullableUUID"
	StringKeyType            KeyType = "String"
	TimestampKeyType         KeyType = "Timestamp"
	UUIDArrayKeyType         KeyType = "UUIDArray"
	UUIDKeyType              KeyType = "UUID"
)

// Validate implements the Validatable interface
func (kt KeyType) Validate() error {
	switch kt {
	case ArrayKeyType:
	case BoolKeyType:
	case IntKeyType:
	case NullableBoolKeyType:
	case NullableIntKeyType:
	case NullableStringKeyType:
	case NullableTimestampKeyType:
	case NullableUUIDKeyType:
	case StringKeyType:
	case TimestampKeyType:
	case UUIDArrayKeyType:
	case UUIDKeyType:
	default:
		return ucerr.Errorf("KeyType is unsupported: '%v'", kt)
	}

	return nil
}

// KeyTypes is a map from pagination keys to their associated KeyTypes
type KeyTypes map[string]KeyType

// Validate implements the Validatable interface
func (kt KeyTypes) Validate() error {
	if len(kt) == 0 {
		return ucerr.New("There must be at least one pagination key")
	}

	for k, t := range kt {
		if err := t.Validate(); err != nil {
			return ucerr.Errorf("key '%s' has invalid key type '%v'", k, t)
		}
	}

	return nil
}

// PageableType is an interface that must be implemented for all pageable result types
type PageableType interface {
	GetCursor(Key) Cursor
	GetPaginationKeys() KeyTypes
}

// Order is a direction in which a view on a collection can be sorted, mapping to ASC/DESC in SQL.
type Order string

// Order can be either ascending or descending.
const (
	OrderAscending  Order = "ascending" // Default
	OrderDescending Order = "descending"
)

// Validate implements the Validatable interface for the Order type
func (o Order) Validate() error {
	if o != OrderAscending && o != OrderDescending {
		return ucerr.Errorf("Order is unrecognized: %d", o)
	}

	return nil
}

// Version represents the version of the pagination request and reply wire format. It will
// be incremented any time that the wire format has changed.
type Version int

// Deprecated pagination versions
const (
	Version1 Version = 1 // cursor format is "id"
)

// Supported pagination versions
const (
	Version2 Version = 2 // cursor format is "key1:id1,...,keyN:idN"
	Version3 Version = 3 // filter option now supported in client
)

// Validate implements the Validatable interface for the Version type
func (v Version) Validate() error {
	switch v {
	case Version1:
		return ucerr.Errorf("version '%v' is no longer supported", v)
	case Version2:
	case Version3:
	default:
		return ucerr.Errorf("version '%v' is unsupported", v)
	}

	return nil
}
