// NOTE: automatically generated file -- DO NOT EDIT

package main

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o companyFile) Validate() error {
	if err := o.Company.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
