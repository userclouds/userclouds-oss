// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ColumnDataType) Validate() error {
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "ColumnDataType.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	if len(o.Description) < 1 || len(o.Description) > 128 {
		return ucerr.Friendlyf(nil, "ColumnDataType.Description length has to be between 1 and 128 (length: %v)", len(o.Description))
	}
	if err := o.CompositeAttributes.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
