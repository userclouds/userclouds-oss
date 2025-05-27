// NOTE: automatically generated file -- DO NOT EDIT

package policy

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AccessPolicyTemplate) Validate() error {
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "AccessPolicyTemplate.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
