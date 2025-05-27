// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o LogCacheParams) Validate() error {
	if err := o.CacheType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
