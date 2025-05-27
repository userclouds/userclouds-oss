// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o BaseUserColumnValue) Validate() error {
	if err := o.VersionBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.ColumnID.IsNil() {
		return ucerr.Friendlyf(nil, "BaseUserColumnValue.ColumnID can't be nil")
	}
	if err := o.Column.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.UserID.IsNil() {
		return ucerr.Friendlyf(nil, "BaseUserColumnValue.UserID can't be nil")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
