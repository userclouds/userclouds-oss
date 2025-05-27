// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateTransformerRequest) Validate() error {
	if err := o.Transformer.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
