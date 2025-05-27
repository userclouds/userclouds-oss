// NOTE: automatically generated file -- DO NOT EDIT

package ucsentry

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.Dsn == "" {
		return ucerr.Friendlyf(nil, "Config.Dsn can't be empty")
	}
	return nil
}
