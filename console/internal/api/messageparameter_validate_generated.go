// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MessageParameter) Validate() error {
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "MessageParameter.Name can't be empty")
	}
	if o.DefaultValue == "" {
		return ucerr.Friendlyf(nil, "MessageParameter.DefaultValue can't be empty")
	}
	return nil
}
