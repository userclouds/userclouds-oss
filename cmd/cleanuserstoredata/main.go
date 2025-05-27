package main

import (
	"context"
	"flag"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/worker"
)

func main() {
	ctx := context.Background()
	noDryRun := flag.Bool("no-dry-run", false, "Disable dry run mode, modify data. Default is dry run mode.")
	maxCandidates := flag.Int("max-candidates", 100, "Max number of candidates to clean up")
	flag.Parse()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "cleanuserstoredata")
	defer logtransports.Close()

	if flag.NArg() < 1 {
		uclog.Infof(ctx, "Usage: cleanuserstoredata [--no-dry-run] [--max-candidates xx] <tenant ID or name>")
		uclog.Infof(ctx, "tenant ID or name is the ID or name of the tenant for which we want to clean userstore data")
		uclog.Infof(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "Must specify tenant ID or name")
	}

	tenantIDOrName := flag.Arg(0)
	ccs := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, ccs, tenantIDOrName)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get tenant '%v': %v", tenantIDOrName, err)
	}
	tenantID := tenant.ID
	wc, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get worker client: %v", err)
	}

	dryRun := !*noDryRun
	uclog.Infof(ctx, "Sending clean UserStore data for tenant %v. dry run: %v", tenantID, dryRun)
	msg := worker.UserStoreDataCleanupMessage(tenantID, *maxCandidates, dryRun)
	if err := wc.Send(ctx, msg); err != nil {
		uclog.Fatalf(ctx, "failed to send message: %v", err)
	}
}
