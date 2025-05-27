package column

import (
	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
)

// DurationUnit is an internal enum for supported duration units
type DurationUnit int

// DurationUnit constants
const (
	DurationUnitIndefinite DurationUnit = 1
	DurationUnitYear       DurationUnit = 2
	DurationUnitMonth      DurationUnit = 3
	DurationUnitWeek       DurationUnit = 4
	DurationUnitDay        DurationUnit = 5
	DurationUnitHour       DurationUnit = 6
)

// Validate implements Validateable
func (du DurationUnit) Validate() error {
	switch du {
	case DurationUnitIndefinite:
	case DurationUnitYear:
	case DurationUnitMonth:
	case DurationUnitWeek:
	case DurationUnitDay:
	case DurationUnitHour:
	default:
		return ucerr.Friendlyf(nil, "Invalid DurationUnit: %d", du)
	}

	return nil
}

// DurationUnitFromClient converts a client duration unit to the internal representation
func DurationUnitFromClient(du idp.DurationUnit) DurationUnit {
	switch du {
	case idp.DurationUnitYear:
		return DurationUnitYear
	case idp.DurationUnitMonth:
		return DurationUnitMonth
	case idp.DurationUnitWeek:
		return DurationUnitWeek
	case idp.DurationUnitDay:
		return DurationUnitDay
	case idp.DurationUnitHour:
		return DurationUnitHour
	default:
		return DurationUnitIndefinite
	}
}

// ToClient converts the internal duration unit to the client representation
func (du DurationUnit) ToClient() idp.DurationUnit {
	switch du {
	case DurationUnitYear:
		return idp.DurationUnitYear
	case DurationUnitMonth:
		return idp.DurationUnitMonth
	case DurationUnitWeek:
		return idp.DurationUnitWeek
	case DurationUnitDay:
		return idp.DurationUnitDay
	case DurationUnitHour:
		return idp.DurationUnitHour
	default:
		return idp.DurationUnitIndefinite
	}
}
