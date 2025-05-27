package main

import (
	"context"
	"flag"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/plex/helpers"
	"userclouds.com/worker"
)

func main() {
	ctx := context.Background()
	direct := flag.Bool("direct", false, "If true, do not use worker. Default is false.")
	noDryRun := flag.Bool("no-dry-run", false, "Disable dry run mode, modify data. Default is dry run mode.")
	maxCandidates := flag.Int("max-candidates", 100, "Max number of candidates to clean up")
	flag.Parse()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "cleanplextokens")
	defer logtransports.Close()

	if flag.NArg() < 1 {
		uclog.Infof(ctx, "Usage: cleanplextokens [--direct] [--no-dry-run] [--max-candidates xx] <tenant ID or name>")
		uclog.Infof(ctx, "tenant ID or name is the ID or name of the tenant for which we want to clean plex tokens")
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

	dryRun := !*noDryRun

	if *direct {
		ti, err := cmdline.GetTenantInternalByID(ctx, ccs, tenantID)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to get tenant internal '%v': %v", tenantID, err)
		}

		tenantDB, err := tenantdb.ConnectWithConfig(ctx, &ti.TenantDBConfig)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to connect to tenant '%v': %v", tenantID, err)
		}

		if err := helpers.CleanPlexTokensForTenant(ctx, tenantDB, nil, *maxCandidates, dryRun); err != nil {
			uclog.Fatalf(ctx, "Error cleaning plex tokens for tenant '%v': %v", tenantID, err)
		}
		return
	}

	wc, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get worker client: %v", err)
	}

	uclog.Infof(ctx, "Sending clean plex tokens for tenant %v. dry run: %v", tenantID, dryRun)
	msg := worker.PlexTokenDataCleanupMessage(tenantID, *maxCandidates, dryRun)
	if err := wc.Send(ctx, msg); err != nil {
		uclog.Fatalf(ctx, "failed to send message: %v", err)
	}
}
