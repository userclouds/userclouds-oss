// NOTE: automatically generated file -- DO NOT EDIT

package cache

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o RedisConfig) Validate() error {
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "RedisConfig.Host can't be empty")
	}
	if o.Port == 0 {
		return ucerr.Friendlyf(nil, "RedisConfig.Port can't be 0")
	}
	if err := o.Password.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
