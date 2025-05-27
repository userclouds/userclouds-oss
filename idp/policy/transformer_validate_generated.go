// NOTE: automatically generated file -- DO NOT EDIT

package policy

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Transformer) Validate() error {
	if len(o.Name) < 1 || len(o.Name) > 128 {
		return ucerr.Friendlyf(nil, "Transformer.Name length has to be between 1 and 128 (length: %v)", len(o.Name))
	}
	if err := o.InputDataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.OutputDataType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.TransformType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
