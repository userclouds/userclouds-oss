// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UserSelectorConfig) Validate() error {
	if o.WhereClause == "" {
		return ucerr.Friendlyf(nil, "UserSelectorConfig.WhereClause can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
