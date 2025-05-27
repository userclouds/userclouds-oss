// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataImportParams) Validate() error {
	if o.JobID.IsNil() {
		return ucerr.Friendlyf(nil, "DataImportParams.JobID can't be nil")
	}
	return nil
}
