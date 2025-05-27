// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SAMLIDP) Validate() error {
	if o.Certificate == "" {
		return ucerr.Friendlyf(nil, "SAMLIDP.Certificate can't be empty")
	}
	if err := o.PrivateKey.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.MetadataURL == "" {
		return ucerr.Friendlyf(nil, "SAMLIDP.MetadataURL can't be empty")
	}
	if o.SSOURL == "" {
		return ucerr.Friendlyf(nil, "SAMLIDP.SSOURL can't be empty")
	}
	return nil
}
