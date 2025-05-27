// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SQLShimDatabase) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "SQLShimDatabase.Name (%v) can't be empty", o.ID)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "SQLShimDatabase.Host (%v) can't be empty", o.ID)
	}
	if o.Port == 0 {
		return ucerr.Friendlyf(nil, "SQLShimDatabase.Port (%v) can't be 0", o.ID)
	}
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "SQLShimDatabase.Username (%v) can't be empty", o.ID)
	}
	if err := o.Password.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
