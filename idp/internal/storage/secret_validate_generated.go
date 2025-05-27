// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Secret) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Secret.Name (%v) can't be empty", o.ID)
	}
	if o.Value.IsEmpty() {
		return ucerr.Friendlyf(nil, "Secret.Value can't be empty")
	}
	return nil
}
