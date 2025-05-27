// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o InspectTokenRequest) Validate() error {
	if o.Token == "" {
		return ucerr.Friendlyf(nil, "InspectTokenRequest.Token can't be empty")
	}
	return nil
}
