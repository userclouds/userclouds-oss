// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MFARequest) Validate() error {
	if err := o.UserBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Code == "" {
		return ucerr.Friendlyf(nil, "MFARequest.Code (%v) can't be empty", o.ID)
	}
	if err := o.SupportedChannelTypes.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
