// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AccessPolicy) Validate() error {
	if err := o.SystemAttributeBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "AccessPolicy.Name can't be empty")
	}
	if err := o.PolicyType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Thresholds.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
