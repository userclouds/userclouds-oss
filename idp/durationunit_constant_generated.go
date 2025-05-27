// NOTE: automatically generated file -- DO NOT EDIT

package idp

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t DurationUnit) MarshalText() ([]byte, error) {
	switch t {
	case DurationUnitDay:
		return []byte("day"), nil
	case DurationUnitHour:
		return []byte("hour"), nil
	case DurationUnitIndefinite:
		return []byte("indefinite"), nil
	case DurationUnitMonth:
		return []byte("month"), nil
	case DurationUnitWeek:
		return []byte("week"), nil
	case DurationUnitYear:
		return []byte("year"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown DurationUnit value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *DurationUnit) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "day":
		*t = DurationUnitDay
	case "hour":
		*t = DurationUnitHour
	case "indefinite":
		*t = DurationUnitIndefinite
	case "month":
		*t = DurationUnitMonth
	case "week":
		*t = DurationUnitWeek
	case "year":
		*t = DurationUnitYear
	default:
		return ucerr.Friendlyf(nil, "unknown DurationUnit value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *DurationUnit) Validate() error {
	switch *t {
	case DurationUnitDay:
		return nil
	case DurationUnitHour:
		return nil
	case DurationUnitIndefinite:
		return nil
	case DurationUnitMonth:
		return nil
	case DurationUnitWeek:
		return nil
	case DurationUnitYear:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown DurationUnit value '%s'", *t)
	}
}

// Enum implements Enum
func (t DurationUnit) Enum() []any {
	return []any{
		"day",
		"hour",
		"indefinite",
		"month",
		"week",
		"year",
	}
}

// AllDurationUnits is a slice of all DurationUnit values
var AllDurationUnits = []DurationUnit{
	DurationUnitDay,
	DurationUnitHour,
	DurationUnitIndefinite,
	DurationUnitMonth,
	DurationUnitWeek,
	DurationUnitYear,
}
