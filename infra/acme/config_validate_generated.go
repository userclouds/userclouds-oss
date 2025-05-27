// NOTE: automatically generated file -- DO NOT EDIT

package acme

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.DirectoryURL == "" {
		return ucerr.Friendlyf(nil, "Config.DirectoryURL can't be empty")
	}
	if o.AccountURL == "" {
		return ucerr.Friendlyf(nil, "Config.AccountURL can't be empty")
	}
	if err := o.PrivateKey.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
