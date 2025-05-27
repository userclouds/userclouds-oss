// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateOIDCProviderRequest) Validate() error {
	if err := o.OIDCProvider.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
