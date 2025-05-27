// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o User) Validate() error {
	if err := o.BaseUser.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
