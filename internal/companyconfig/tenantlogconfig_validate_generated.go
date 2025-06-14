// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantLogConfig) Validate() error {
	if err := o.LogDB.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
