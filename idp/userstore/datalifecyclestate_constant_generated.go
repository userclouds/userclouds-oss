// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t DataLifeCycleState) MarshalText() ([]byte, error) {
	switch t {
	case DataLifeCycleStateDefault:
		return []byte(""), nil
	case DataLifeCycleStateLive:
		return []byte("live"), nil
	case DataLifeCycleStatePostDelete:
		return []byte("postdelete"), nil
	case DataLifeCycleStatePreDelete:
		return []byte("predelete"), nil
	case DataLifeCycleStateSoftDeleted:
		return []byte("softdeleted"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown DataLifeCycleState value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *DataLifeCycleState) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "":
		*t = DataLifeCycleStateDefault
	case "live":
		*t = DataLifeCycleStateLive
	case "postdelete":
		*t = DataLifeCycleStatePostDelete
	case "predelete":
		*t = DataLifeCycleStatePreDelete
	case "softdeleted":
		*t = DataLifeCycleStateSoftDeleted
	default:
		return ucerr.Friendlyf(nil, "unknown DataLifeCycleState value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *DataLifeCycleState) Validate() error {
	switch *t {
	case DataLifeCycleStateDefault:
		return nil
	case DataLifeCycleStateLive:
		return nil
	case DataLifeCycleStatePostDelete:
		return nil
	case DataLifeCycleStatePreDelete:
		return nil
	case DataLifeCycleStateSoftDeleted:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown DataLifeCycleState value '%s'", *t)
	}
}

// Enum implements Enum
func (t DataLifeCycleState) Enum() []any {
	return []any{
		"",
		"live",
		"postdelete",
		"predelete",
		"softdeleted",
	}
}

// AllDataLifeCycleStates is a slice of all DataLifeCycleState values
var AllDataLifeCycleStates = []DataLifeCycleState{
	DataLifeCycleStateDefault,
	DataLifeCycleStateLive,
	DataLifeCycleStatePostDelete,
	DataLifeCycleStatePreDelete,
	DataLifeCycleStateSoftDeleted,
}
