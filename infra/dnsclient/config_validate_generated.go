// NOTE: automatically generated file -- DO NOT EDIT

package dnsclient

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.HostAndPort == "" {
		return ucerr.Friendlyf(nil, "Config.HostAndPort can't be empty")
	}
	return nil
}
