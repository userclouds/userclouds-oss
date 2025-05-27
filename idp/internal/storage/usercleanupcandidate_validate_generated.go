// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UserCleanupCandidate) Validate() error {
	if err := o.UserBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.CleanupReason.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
