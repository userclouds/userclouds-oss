// NOTE: automatically generated file -- DO NOT EDIT

package config

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SQLShimConfig) Validate() error {
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
