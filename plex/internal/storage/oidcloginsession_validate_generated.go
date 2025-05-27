// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o OIDCLoginSession) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "OIDCLoginSession.ClientID (%v) can't be empty", o.ID)
	}
	if o.ResponseTypes == "" {
		return ucerr.Friendlyf(nil, "OIDCLoginSession.ResponseTypes (%v) can't be empty", o.ID)
	}
	if o.RedirectURI == "" {
		return ucerr.Friendlyf(nil, "OIDCLoginSession.RedirectURI (%v) can't be empty", o.ID)
	}
	if o.State == "" {
		return ucerr.Friendlyf(nil, "OIDCLoginSession.State (%v) can't be empty", o.ID)
	}
	if o.Scopes == "" {
		return ucerr.Friendlyf(nil, "OIDCLoginSession.Scopes (%v) can't be empty", o.ID)
	}
	if err := o.OIDCProvider.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.AddAuthnProviderData.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
