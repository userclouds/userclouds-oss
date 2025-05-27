// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if err := o.CompanyDB.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
