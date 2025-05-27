// NOTE: automatically generated file -- DO NOT EDIT

package ucopensearch

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.URL == "" {
		return ucerr.Friendlyf(nil, "Config.URL can't be empty")
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
