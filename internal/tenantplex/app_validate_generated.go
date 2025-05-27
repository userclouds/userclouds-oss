// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o App) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "App.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "App.Name (%v) can't be empty", o.ID)
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "App.ClientID (%v) can't be empty", o.ID)
	}
	if err := o.ClientSecret.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.TokenValidity.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	for i, item := range o.ProviderAppIDs {
		if item.IsNil() {
			return ucerr.Friendlyf(nil, "App.ProviderAppIDs[%d] (%v) can't be nil", i, o.ID)
		}
	}
	if err := o.PageParameters.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	for _, item := range o.GrantTypes {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.SAMLIDP != nil {
		if err := o.SAMLIDP.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
