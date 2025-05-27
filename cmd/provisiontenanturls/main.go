package main

import (
	"context"
	"flag"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/worker"
)

func main() {
	ctx := context.Background()
	addEKSURLS := flag.Bool("eks", false, "add eks specific regional URLs. e.ge. my-tenant-name.tenant-aws-us-west-2-eks.userclouds.com")
	deleteURLs := flag.Bool("delete", false, "Delete tenant URLs and are not in current region or that have eks in the URL when eks flag is not set")
	noDryRun := flag.Bool("no-dry-run", false, "Disable dry run mode, modify data. Default is dry run mode.")

	flag.Parse()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "provisiontenanturls")
	defer logtransports.Close()

	if flag.NArg() < 1 {
		uclog.Infof(ctx, "Usage: cleanuserstoredata  <tenant ID or name>")
		uclog.Infof(ctx, "tenant ID or name is the ID or name of the tenant for which we want to provision tenant URLs, or '*all*' to provision URLs for all tenants.")
		uclog.Infof(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "Must specify tenant ID or name")
	}
	wc, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get worker client: %v", err)
	}
	ccs := cmdline.GetCompanyStorage(ctx)

	var tenantIDs []uuid.UUID
	if flag.Arg(0) == "*all*" {
		tenantIDs, err = cmdline.GetAllTenantIDs(ctx, ccs)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to get all tenants: %v", err)
		}
	} else {
		tenantIDOrName := flag.Arg(0)
		tenant, err := cmdline.GetTenantByIDOrName(ctx, ccs, tenantIDOrName)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to get tenant '%v': %v", tenantIDOrName, err)
		}
		tenantIDs = []uuid.UUID{tenant.ID}
	}

	for _, tenantID := range tenantIDs {
		uclog.Infof(ctx, "Sending Provision Tenant URLs for tenant %v.", tenantID)
		msg := worker.ProvisionTenantURLsMessage(tenantID, *addEKSURLS, *deleteURLs, !*noDryRun)
		if err := wc.Send(ctx, msg); err != nil {
			uclog.Fatalf(ctx, "failed to send message: %v", err)
		}
	}
}
