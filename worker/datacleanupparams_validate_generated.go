// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataCleanupParams) Validate() error {
	if o.MaxCandidates == 0 {
		return ucerr.Friendlyf(nil, "DataCleanupParams.MaxCandidates can't be 0")
	}
	return nil
}
