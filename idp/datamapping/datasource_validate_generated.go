// NOTE: automatically generated file -- DO NOT EDIT

package datamapping

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataSource) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "DataSource.Name (%v) can't be empty", o.ID)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Config.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Metadata.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
