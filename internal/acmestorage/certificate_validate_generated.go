// NOTE: automatically generated file -- DO NOT EDIT

package acmestorage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Certificate) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.PrivateKey.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
