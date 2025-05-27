// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MFAChannel) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "MFAChannel.ID (%v) can't be nil", o.ID)
	}
	if err := o.ChannelType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.ChannelTypeID == "" {
		return ucerr.Friendlyf(nil, "MFAChannel.ChannelTypeID (%v) can't be empty", o.ID)
	}
	if o.ChannelName == "" {
		return ucerr.Friendlyf(nil, "MFAChannel.ChannelName (%v) can't be empty", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
