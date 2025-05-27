// NOTE: automatically generated file -- DO NOT EDIT

package uctrace

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.CollectorHost == "" {
		return ucerr.Friendlyf(nil, "Config.CollectorHost can't be empty")
	}
	return nil
}
