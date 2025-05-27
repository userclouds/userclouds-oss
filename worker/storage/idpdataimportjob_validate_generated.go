// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o IDPDataImportJob) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.ImportType == "" {
		return ucerr.Friendlyf(nil, "IDPDataImportJob.ImportType (%v) can't be empty", o.ID)
	}
	if o.Status == "" {
		return ucerr.Friendlyf(nil, "IDPDataImportJob.Status (%v) can't be empty", o.ID)
	}
	if o.S3Bucket == "" {
		return ucerr.Friendlyf(nil, "IDPDataImportJob.S3Bucket (%v) can't be empty", o.ID)
	}
	if o.ObjectKey == "" {
		return ucerr.Friendlyf(nil, "IDPDataImportJob.ObjectKey (%v) can't be empty", o.ID)
	}
	return nil
}
