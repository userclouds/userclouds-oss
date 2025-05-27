// NOTE: automatically generated file -- DO NOT EDIT

package logdb

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if err := o.DB.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
