// NOTE: automatically generated file -- DO NOT EDIT

package custompages

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CustomPage) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.PageName == "" {
		return ucerr.Friendlyf(nil, "CustomPage.PageName (%v) can't be empty", o.ID)
	}
	if o.PageSource == "" {
		return ucerr.Friendlyf(nil, "CustomPage.PageSource (%v) can't be empty", o.ID)
	}
	return nil
}
