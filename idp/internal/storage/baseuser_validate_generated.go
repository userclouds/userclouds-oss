// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o BaseUser) Validate() error {
	if err := o.VersionBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
