// NOTE: automatically generated file -- DO NOT EDIT

package featureflags

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.APIKey.IsEmpty() {
		return ucerr.Friendlyf(nil, "Config.APIKey can't be empty")
	}
	return nil
}
