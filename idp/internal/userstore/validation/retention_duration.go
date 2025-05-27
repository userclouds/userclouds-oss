package validation

import (
	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/ucerr"
)

type retentionDurationValidator struct {
	dlcs column.DataLifeCycleState
}

func (v retentionDurationValidator) validateRetentionDuration(d idp.RetentionDuration) error {
	if err := d.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if v.dlcs == column.DataLifeCycleStateLive {
		if d.Unit != idp.DurationUnitIndefinite && d.Duration == 0 {
			return ucerr.Friendlyf(nil, "live retention duration cannot have a duration of 0")
		}
	} else if d.Unit == idp.DurationUnitIndefinite {
		return ucerr.Friendlyf(nil, "soft-deleted retention duration cannot have an indefinite duration")
	}

	return nil
}
