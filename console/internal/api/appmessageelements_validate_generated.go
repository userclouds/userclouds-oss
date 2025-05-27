// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AppMessageElements) Validate() error {
	if o.AppID.IsNil() {
		return ucerr.Friendlyf(nil, "AppMessageElements.AppID can't be nil")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
