// NOTE: automatically generated file -- DO NOT EDIT

package internal

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PageParametersRequest) Validate() error {
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "PageParametersRequest.SessionID can't be nil")
	}
	if err := o.PageType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
