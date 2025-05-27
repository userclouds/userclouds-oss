// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Message) Validate() error {
	if err := o.SourceRegion.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.FinalizeTenantCNAME != nil {
		if err := o.FinalizeTenantCNAME.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.CheckTenantCNAME != nil {
		if err := o.CheckTenantCNAME.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.ImportAuth0Apps != nil {
		if err := o.ImportAuth0Apps.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.ClearCache != nil {
		if err := o.ClearCache.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.LogCache != nil {
		if err := o.LogCache.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.TenantDNS != nil {
		if err := o.TenantDNS.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.DataImportParams != nil {
		if err := o.DataImportParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.PlexTokenDataCleanup != nil {
		if err := o.PlexTokenDataCleanup.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.UserStoreDataCleanup != nil {
		if err := o.UserStoreDataCleanup.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.TenantURLProvisioningParams != nil {
		if err := o.TenantURLProvisioningParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.IngestSqlshimDatabaseSchemasParams != nil {
		if err := o.IngestSqlshimDatabaseSchemasParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.ProvisionTenantOpenSearchIndexParams != nil {
		if err := o.ProvisionTenantOpenSearchIndexParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.BootstrapTenantOpenSearchIndexParams != nil {
		if err := o.BootstrapTenantOpenSearchIndexParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.UpdateTenantOpenSearchIndexParams != nil {
		if err := o.UpdateTenantOpenSearchIndexParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.NoOpParams != nil {
		if err := o.NoOpParams.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
