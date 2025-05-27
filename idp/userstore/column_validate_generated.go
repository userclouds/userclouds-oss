// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Column) Validate() error {
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "Column.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	if err := o.DataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.IndexType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Constraints.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
