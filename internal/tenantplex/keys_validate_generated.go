// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Keys) Validate() error {
	if o.KeyID == "" {
		return ucerr.Friendlyf(nil, "Keys.KeyID can't be empty")
	}
	if err := o.PrivateKey.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.PublicKey == "" {
		return ucerr.Friendlyf(nil, "Keys.PublicKey can't be empty")
	}
	return nil
}
