// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantConfig) Validate() error {
	if err := o.PlexMap.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.OIDCProviders.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.ExternalOIDCIssuers.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Keys.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.PageParameters.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
