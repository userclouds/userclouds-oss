// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateTokenRequest) Validate() error {
	if err := o.TransformerRID.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.AccessPolicyRID.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
