// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TokenRecord) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Token == "" {
		return ucerr.Friendlyf(nil, "TokenRecord.Token (%v) can't be empty", o.ID)
	}
	if o.TransformerID.IsNil() {
		return ucerr.Friendlyf(nil, "TokenRecord.TransformerID (%v) can't be nil", o.ID)
	}
	if o.AccessPolicyID.IsNil() {
		return ucerr.Friendlyf(nil, "TokenRecord.AccessPolicyID (%v) can't be nil", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
