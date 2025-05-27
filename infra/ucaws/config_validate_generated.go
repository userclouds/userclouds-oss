// NOTE: automatically generated file -- DO NOT EDIT

package ucaws

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.AccessKeyID == "" {
		return ucerr.Friendlyf(nil, "Config.AccessKeyID can't be empty")
	}
	if o.AccessKeySecret == "" {
		return ucerr.Friendlyf(nil, "Config.AccessKeySecret can't be empty")
	}
	if o.Region == "" {
		return ucerr.Friendlyf(nil, "Config.Region can't be empty")
	}
	return nil
}
