// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateColumnRetentionDurationRequest) Validate() error {
	if err := o.RetentionDuration.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
