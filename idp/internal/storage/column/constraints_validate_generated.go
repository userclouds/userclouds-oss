// NOTE: automatically generated file -- DO NOT EDIT

package column

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Constraints) Validate() error {
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
