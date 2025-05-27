// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Parameter) Validate() error {
	if err := o.Name.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
