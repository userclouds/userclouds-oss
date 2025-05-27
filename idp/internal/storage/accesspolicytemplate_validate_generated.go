// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AccessPolicyTemplate) Validate() error {
	if err := o.SystemAttributeBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "AccessPolicyTemplate.Name can't be empty")
	}
	if o.Function == "" {
		return ucerr.Friendlyf(nil, "AccessPolicyTemplate.Function can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
