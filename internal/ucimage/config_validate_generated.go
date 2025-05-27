// NOTE: automatically generated file -- DO NOT EDIT

package ucimage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "Config.Host can't be empty")
	}
	if o.S3Bucket == "" {
		return ucerr.Friendlyf(nil, "Config.S3Bucket can't be empty")
	}
	return nil
}
