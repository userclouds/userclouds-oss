// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Certificate) Validate() error {
	if o.Certificate == "" {
		return ucerr.Friendlyf(nil, "Certificate.Certificate can't be empty")
	}
	if err := o.PrivateKey.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
