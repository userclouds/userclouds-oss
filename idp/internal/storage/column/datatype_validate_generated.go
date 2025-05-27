// NOTE: automatically generated file -- DO NOT EDIT

package column

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataType) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "DataType.Name (%v) can't be empty", o.ID)
	}
	if o.Description == "" {
		return ucerr.Friendlyf(nil, "DataType.Description (%v) can't be empty", o.ID)
	}
	if o.ConcreteDataTypeID.IsNil() {
		return ucerr.Friendlyf(nil, "DataType.ConcreteDataTypeID (%v) can't be nil", o.ID)
	}
	if err := o.CompositeAttributes.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
