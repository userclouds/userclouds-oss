// NOTE: automatically generated file -- DO NOT EDIT

package policy

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Secret) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Secret.ID (%v) can't be nil", o.ID)
	}
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "Secret.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	return nil
}
