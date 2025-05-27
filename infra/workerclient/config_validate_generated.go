// NOTE: automatically generated file -- DO NOT EDIT

package workerclient

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.Type == "" {
		return ucerr.Friendlyf(nil, "Config.Type can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
