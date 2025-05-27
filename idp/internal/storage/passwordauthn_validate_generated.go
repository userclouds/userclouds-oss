// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PasswordAuthn) Validate() error {
	if err := o.UserBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "PasswordAuthn.Username (%v) can't be empty", o.ID)
	}
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "PasswordAuthn.Password (%v) can't be empty", o.ID)
	}
	return nil
}
