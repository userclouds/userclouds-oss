// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UserMFAConfiguration) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.MFAChannels.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
