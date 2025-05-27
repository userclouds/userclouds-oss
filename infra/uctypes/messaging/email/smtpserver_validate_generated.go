// NOTE: automatically generated file -- DO NOT EDIT

package email

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SMTPServer) Validate() error {
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "SMTPServer.Host can't be empty")
	}
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "SMTPServer.Username can't be empty")
	}
	if err := o.Password.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
