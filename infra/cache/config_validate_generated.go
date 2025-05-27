// NOTE: automatically generated file -- DO NOT EDIT

package cache

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	for _, item := range o.RedisCacheConfig {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
