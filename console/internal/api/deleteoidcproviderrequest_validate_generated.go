// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DeleteOIDCProviderRequest) Validate() error {
	if o.OIDCProviderName == "" {
		return ucerr.Friendlyf(nil, "DeleteOIDCProviderRequest.OIDCProviderName can't be empty")
	}
	return nil
}
