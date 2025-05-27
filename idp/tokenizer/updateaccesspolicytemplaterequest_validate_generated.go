// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateAccessPolicyTemplateRequest) Validate() error {
	if err := o.AccessPolicyTemplate.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
