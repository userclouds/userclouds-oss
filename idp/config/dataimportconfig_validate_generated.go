// NOTE: automatically generated file -- DO NOT EDIT

package config

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o DataImportConfig) Validate() error {
	if o.DataImportS3Bucket == "" {
		return ucerr.Friendlyf(nil, "DataImportConfig.DataImportS3Bucket can't be empty")
	}
	return nil
}
