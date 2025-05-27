// NOTE: automatically generated file -- DO NOT EDIT

package main

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CacheOnlyConfig) Validate() error {
	if err := o.CacheConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
