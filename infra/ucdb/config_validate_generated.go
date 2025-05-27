// NOTE: automatically generated file -- DO NOT EDIT

package ucdb

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.User == "" {
		return ucerr.Friendlyf(nil, "Config.User can't be empty")
	}
	if err := o.Password.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.DBName == "" {
		return ucerr.Friendlyf(nil, "Config.DBName can't be empty")
	}
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "Config.Host can't be empty")
	}
	if o.Port == "" {
		return ucerr.Friendlyf(nil, "Config.Port can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
