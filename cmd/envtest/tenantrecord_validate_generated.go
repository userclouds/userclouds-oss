// NOTE: automatically generated file -- DO NOT EDIT

package main

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantRecord) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "TenantRecord.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "TenantRecord.Name (%v) can't be empty", o.ID)
	}
	if o.TenantURL == "" {
		return ucerr.Friendlyf(nil, "TenantRecord.TenantURL (%v) can't be empty", o.ID)
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "TenantRecord.ClientID (%v) can't be empty", o.ID)
	}
	if o.ClientSecret == "" {
		return ucerr.Friendlyf(nil, "TenantRecord.ClientSecret (%v) can't be empty", o.ID)
	}
	return nil
}
