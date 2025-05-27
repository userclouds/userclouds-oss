package cmdline

import (
	"context"

	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
)

// GetCompanyStorage returns a companyconfig.Storage object
func GetCompanyStorage(ctx context.Context) *companyconfig.Storage {
	sd, err := companyconfig.GetServiceData(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't get service data: %v", err)
	}
	// TODO enable cache invalidation from tools once we run them in an environment that can connect to a cache
	// for now we will depend on clearing the caches on deploy
	// lrc, rrc := console.GetCacheData(ctx)
	companyStorage, err := companyconfig.NewStorageFromConfig(ctx, sd.DBCfg, nil)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't create new companystorage: %v", err)
	}
	return companyStorage
}
