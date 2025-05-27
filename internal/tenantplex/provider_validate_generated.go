// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Provider) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Provider.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Provider.Name (%v) can't be empty", o.ID)
	}
	if o.Auth0 != nil {
		if err := o.Auth0.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.UC != nil {
		if err := o.UC.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.Cognito != nil {
		if err := o.Cognito.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
