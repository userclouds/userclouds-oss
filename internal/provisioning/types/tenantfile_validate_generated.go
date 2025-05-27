// NOTE: automatically generated file -- DO NOT EDIT

package types

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantFile) Validate() error {
	if err := o.Tenant.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.PlexConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Protocol == "" {
		return ucerr.Friendlyf(nil, "TenantFile.Protocol can't be empty")
	}
	if o.SubDomain == "" {
		return ucerr.Friendlyf(nil, "TenantFile.SubDomain can't be empty")
	}
	if o.TenantDBCfg != nil {
		if err := o.TenantDBCfg.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
