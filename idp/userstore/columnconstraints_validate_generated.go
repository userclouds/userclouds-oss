// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ColumnConstraints) Validate() error {
	for _, item := range o.Fields {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
