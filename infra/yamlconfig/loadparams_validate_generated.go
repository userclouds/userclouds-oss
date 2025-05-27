// NOTE: automatically generated file -- DO NOT EDIT

package yamlconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o LoadParams) Validate() error {
	if err := o.Universe.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
