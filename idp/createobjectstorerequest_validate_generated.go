// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateObjectStoreRequest) Validate() error {
	if err := o.ObjectStore.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
