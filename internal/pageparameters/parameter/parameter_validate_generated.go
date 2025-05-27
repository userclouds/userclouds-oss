// NOTE: automatically generated file -- DO NOT EDIT

package parameter

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Parameter) Validate() error {
	if err := o.Name.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
