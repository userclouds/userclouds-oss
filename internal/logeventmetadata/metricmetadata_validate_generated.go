// NOTE: automatically generated file -- DO NOT EDIT

package logeventmetadata

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MetricMetadata) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.StringID == "" {
		return ucerr.Friendlyf(nil, "MetricMetadata.StringID (%v) can't be empty", o.ID)
	}
	if err := o.Attributes.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
