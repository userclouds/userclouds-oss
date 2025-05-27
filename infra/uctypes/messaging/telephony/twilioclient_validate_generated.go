// NOTE: automatically generated file -- DO NOT EDIT

package telephony

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o twilioClient) Validate() error {
	if o.client == nil {
		return ucerr.Friendlyf(nil, "twilioClient.client can't be nil")
	}
	return nil
}
