// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AccessorSearchIndex) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.UserSearchIndexID.IsNil() {
		return ucerr.Friendlyf(nil, "AccessorSearchIndex.UserSearchIndexID (%v) can't be nil", o.ID)
	}
	if err := o.QueryType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
