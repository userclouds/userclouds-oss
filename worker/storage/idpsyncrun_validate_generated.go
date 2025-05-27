// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o IDPSyncRun) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
