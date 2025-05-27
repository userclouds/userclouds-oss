// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MFAChannelRequest) Validate() error {
	if err := o.MFAChannel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
