// NOTE: automatically generated file -- DO NOT EDIT

package internal

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if err := o.Endpoint.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Log.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
