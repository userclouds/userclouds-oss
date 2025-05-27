// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ClientCredentialsTokenSource) Validate() error {
	if o.TokenURL == "" {
		return ucerr.Friendlyf(nil, "ClientCredentialsTokenSource.TokenURL can't be empty")
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "ClientCredentialsTokenSource.ClientID can't be empty")
	}
	if o.ClientSecret == "" {
		return ucerr.Friendlyf(nil, "ClientCredentialsTokenSource.ClientSecret can't be empty")
	}
	return nil
}
