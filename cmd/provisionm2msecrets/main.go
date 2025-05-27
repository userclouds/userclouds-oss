package main

import (
	"context"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "provisionm2msecrets")
	defer logtransports.Close()

	ccs := cmdline.GetCompanyStorage(ctx)

	pager, err := companyconfig.NewTenantPaginatorFromOptions()
	if err != nil {
		uclog.Fatalf(ctx, "Failed to apply options: %v", err)
	}

	for {
		tenants, pr, err := ccs.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to list tenants: %v", err)
		}

		uclog.Infof(ctx, "got %d tenants", len(tenants))

		for _, tenant := range tenants {
			uclog.Infof(ctx, "Creating secret for tenant %s", tenant.ID)
			if err := m2m.CreateSecret(ctx, &tenant); err != nil {
				uclog.Fatalf(ctx, "Failed to create secret for tenant %s: %v", tenant.ID, err)
			}
		}

		uclog.Infof(ctx, "finished this page of tenants")

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	uclog.Infof(ctx, "Finished provisioning M2M secrets")
}
