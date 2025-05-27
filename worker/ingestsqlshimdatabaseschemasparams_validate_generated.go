// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o IngestSqlshimDatabaseSchemasParams) Validate() error {
	if o.DatabaseID.IsNil() {
		return ucerr.Friendlyf(nil, "IngestSqlshimDatabaseSchemasParams.DatabaseID can't be nil")
	}
	return nil
}
