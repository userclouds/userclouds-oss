// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ParameterChange) Validate() error {
	if err := o.Name.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
