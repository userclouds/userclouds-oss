package main

import (
	"context"

	migrateuser "userclouds.com/idp/migration/user"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "cleanupusercolumns")
	defer logtransports.Close()

	cleaner := migrateuser.NewColumnCleaner(ctx, cmdline.GetCompanyStorage(ctx))
	cleaner.CleanUpAllTenants()
}
