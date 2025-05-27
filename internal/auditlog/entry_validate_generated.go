// NOTE: automatically generated file -- DO NOT EDIT

package auditlog

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Entry) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if len(o.Type) < 1 || len(o.Type) > 64 {
		return ucerr.Friendlyf(nil, "Entry.Type length has to be between 1 and 64 (length: %v)", len(o.Type))
	}
	if o.Actor == "" {
		return ucerr.Friendlyf(nil, "Entry.Actor (%v) can't be empty", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
