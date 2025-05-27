// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CompositeField) Validate() error {
	if err := o.DataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "CompositeField.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	return nil
}
