// NOTE: automatically generated file -- DO NOT EDIT

package ucdb

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o VersionBaseModel) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
