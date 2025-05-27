// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Session) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
