// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ShimObjectStore) Validate() error {
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "ShimObjectStore.Name (%v) can't be empty", o.ID)
	}
	if o.Type == "" {
		return ucerr.Friendlyf(nil, "ShimObjectStore.Type (%v) can't be empty", o.ID)
	}
	if o.Region == "" {
		return ucerr.Friendlyf(nil, "ShimObjectStore.Region (%v) can't be empty", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
