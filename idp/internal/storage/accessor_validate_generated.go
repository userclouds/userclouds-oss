// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Accessor) Validate() error {
	if err := o.SystemAttributeBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Accessor.Name can't be empty")
	}
	if err := o.DataLifeCycleState.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.AccessPolicyID.IsNil() {
		return ucerr.Friendlyf(nil, "Accessor.AccessPolicyID can't be nil")
	}
	if err := o.SelectorConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
