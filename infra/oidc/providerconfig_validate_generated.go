// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ProviderConfig) Validate() error {
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "ProviderConfig.Name can't be empty")
	}
	if o.Description == "" {
		return ucerr.Friendlyf(nil, "ProviderConfig.Description can't be empty")
	}
	if o.IssuerURL == "" {
		return ucerr.Friendlyf(nil, "ProviderConfig.IssuerURL can't be empty")
	}
	if err := o.ClientSecret.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.DefaultScopes == "" {
		return ucerr.Friendlyf(nil, "ProviderConfig.DefaultScopes can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
