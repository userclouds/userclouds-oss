// NOTE: automatically generated file -- DO NOT EDIT

package uiinitdata

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if err := o.Sentry.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
