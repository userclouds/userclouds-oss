// NOTE: automatically generated file -- DO NOT EDIT

package plex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateUserRequest) Validate() error {
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "CreateUserRequest.ClientID can't be empty")
	}
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "CreateUserRequest.Username can't be empty")
	}
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "CreateUserRequest.Password can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
