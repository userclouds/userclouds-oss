package tenantinfo

import (
	"context"

	"userclouds.com/console/internal"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
)

// GetConsoleTenantInfo returns info about the UserClouds tenant that the Console
// service uses (or an error)
func GetConsoleTenantInfo(ctx context.Context, companyStorage *companyconfig.Storage) (*companyconfig.TenantInfo, error) {
	cfg, err := internal.LoadConfig(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return companyStorage.GetTenantInfo(ctx, cfg.ConsoleTenantID)
}
