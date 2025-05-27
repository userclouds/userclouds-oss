// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ModifiedMessageTypeMessageElements) Validate() error {
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "ModifiedMessageTypeMessageElements.TenantID can't be nil")
	}
	if o.AppID.IsNil() {
		return ucerr.Friendlyf(nil, "ModifiedMessageTypeMessageElements.AppID can't be nil")
	}
	if err := o.MessageType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
