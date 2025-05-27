// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Accessor) Validate() error {
	if len(o.Name) < 1 || len(o.Name) > 256 {
		return ucerr.Friendlyf(nil, "Accessor.Name length has to be between 1 and 256 (length: %v)", len(o.Name))
	}
	if err := o.DataLifeCycleState.Validate(); err != nil {
		return ucerr.Wrap(err)
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
