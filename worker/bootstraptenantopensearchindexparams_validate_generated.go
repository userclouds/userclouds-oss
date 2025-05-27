// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o BootstrapTenantOpenSearchIndexParams) Validate() error {
	if o.IndexID.IsNil() {
		return ucerr.Friendlyf(nil, "BootstrapTenantOpenSearchIndexParams.IndexID can't be nil")
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
