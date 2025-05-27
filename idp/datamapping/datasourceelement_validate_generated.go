// NOTE: automatically generated file -- DO NOT EDIT

package datamapping

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataSourceElement) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.DataSourceID.IsNil() {
		return ucerr.Friendlyf(nil, "DataSourceElement.DataSourceID (%v) can't be nil", o.ID)
	}
	if err := o.Metadata.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
