// NOTE: automatically generated file -- DO NOT EDIT

package cache

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o RegionalRedisConfig) Validate() error {
	if err := o.RedisConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
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
