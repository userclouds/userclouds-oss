// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ColumnValueRetentionDuration) Validate() error {
	if err := o.VersionBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.DurationType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.DurationUnit.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
