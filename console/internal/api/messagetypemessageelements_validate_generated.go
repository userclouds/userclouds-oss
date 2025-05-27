// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MessageTypeMessageElements) Validate() error {
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	keysForMessageParameters := map[string]bool{}
	for _, item := range o.MessageParameters {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		if _, found := keysForMessageParameters[item.Name]; found {
			return ucerr.Friendlyf(nil, "duplicate Name '%v' in MessageParameters", item.Name)
		}
		keysForMessageParameters[item.Name] = true
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
