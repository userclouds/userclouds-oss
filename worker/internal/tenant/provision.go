package tenant

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
)

// ProvisionTenantURLs provisions tenant URLs
func ProvisionTenantURLs(ctx context.Context, companyConfigStorage *companyconfig.Storage, tenantID uuid.UUID, addEKSURLs, deleteURLs, dryRun bool) error {
	tenant, err := companyConfigStorage.GetTenant(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := tenantProvisioning.ProvisionRegionalTenantURLs(ctx, companyConfigStorage, tenant, addEKSURLs); err != nil {
		return ucerr.Wrap(err)
	}
	if !deleteURLs {
		return nil
	}
	return ucerr.Wrap(tenantProvisioning.CleanupRegionalTenantURLs(ctx, companyConfigStorage, tenant, addEKSURLs, dryRun))
}
