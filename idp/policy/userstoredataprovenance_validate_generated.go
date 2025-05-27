// NOTE: automatically generated file -- DO NOT EDIT

package policy

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UserstoreDataProvenance) Validate() error {
	if o.UserID.IsNil() {
		return ucerr.Friendlyf(nil, "UserstoreDataProvenance.UserID can't be nil")
	}
	if o.ColumnID.IsNil() {
		return ucerr.Friendlyf(nil, "UserstoreDataProvenance.ColumnID can't be nil")
	}
	return nil
}
