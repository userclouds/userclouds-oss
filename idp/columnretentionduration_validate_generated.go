// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ColumnRetentionDuration) Validate() error {
	if err := o.DurationType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Duration.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.DefaultDuration != nil {
		if err := o.DefaultDuration.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.PurposeName != nil && *o.PurposeName == "" {
		return ucerr.Friendlyf(nil, "ColumnRetentionDuration.PurposeName (%v) can't be not nil and empty", o.ID)
	}
	return nil
}
