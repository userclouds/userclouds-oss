// NOTE: automatically generated file -- DO NOT EDIT

package main

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantRecords) Validate() error {
	for _, item := range o.Tenants {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
