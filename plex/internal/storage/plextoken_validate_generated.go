// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PlexToken) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "PlexToken.ClientID (%v) can't be empty", o.ID)
	}
	if o.AuthCode == "" {
		return ucerr.Friendlyf(nil, "PlexToken.AuthCode (%v) can't be empty", o.ID)
	}
	if o.AccessToken == "" {
		return ucerr.Friendlyf(nil, "PlexToken.AccessToken (%v) can't be empty", o.ID)
	}
	if o.Scopes == "" {
		return ucerr.Friendlyf(nil, "PlexToken.Scopes (%v) can't be empty", o.ID)
	}
	return nil
}
