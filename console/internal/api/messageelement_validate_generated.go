// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MessageElement) Validate() error {
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.DefaultValue == "" {
		return ucerr.Friendlyf(nil, "MessageElement.DefaultValue can't be empty")
	}
	return nil
}
