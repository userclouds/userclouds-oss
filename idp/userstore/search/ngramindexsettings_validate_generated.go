// NOTE: automatically generated file -- DO NOT EDIT

package search

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o NgramIndexSettings) Validate() error {
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
