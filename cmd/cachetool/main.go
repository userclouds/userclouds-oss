package main

import (
	"context"
	"os"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/cmdline"
	"userclouds.com/worker"
)

const (
	logCmd   = "log"
	clearCmd = "clear"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "cachetool")
	defer logtransports.Close()

	if len(os.Args) < 3 {
		uclog.Infof(ctx, "Usage: cachetool <command> <cache type> <tenant ID or name> ")
		uclog.Infof(ctx, "command is one of the following: clear (clear keys) or log (logs keys that would be cleared)")
		uclog.Infof(ctx, "cache type is on of the following: %v", worker.AllCacheTypes)
		uclog.Infof(ctx, "tenant ID or name is the ID or name of the tenant to clear the cache for authz or userstore")
		uclog.Infof(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "unknown usage")
	}
	cmd := os.Args[1]
	if cmd != logCmd && cmd != clearCmd {
		uclog.Fatalf(ctx, "Command must be log or clear. Unknown command: %v", cmd)
	}
	cacheType := os.Args[2]
	wc, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get worker client: %v", err)
	}

	if len(os.Args) == 4 && os.Args[3] == "all" {
		if err := sendCmdToAllTenants(ctx, wc, cmd, cacheType); err != nil {
			uclog.Fatalf(ctx, "Failed to clear cache for all tenants: %v", err)
		}
		return
	}
	var tenantID uuid.UUID
	if len(os.Args) == 4 {
		tenantIDOrName := os.Args[3]
		ccs := cmdline.GetCompanyStorage(ctx)
		tenant, err := cmdline.GetTenantByIDOrName(ctx, ccs, tenantIDOrName)
		if err != nil {
			uclog.Fatalf(ctx, "Failed to get tenant '%v': %v", tenantIDOrName, err)
		}
		tenantID = tenant.ID
	} else {
		tenantID = uuid.Nil
	}

	if err := sendCmdToTenant(ctx, wc, cmd, cacheType, tenantID); err != nil {
		uclog.Fatalf(ctx, "Failed to clear cache for tenant %v: %v", tenantID, err)
	}
}

func sendCmdToAllTenants(ctx context.Context, wc workerclient.Client, cmd string, cacheType string) error {
	tenantIDs, err := cmdline.GetAllTenantIDs(ctx, cmdline.GetCompanyStorage(ctx))
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, tid := range tenantIDs {
		if err := sendCmdToTenant(ctx, wc, cmd, cacheType, tid); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func sendCmdToTenant(ctx context.Context, wc workerclient.Client, cmd string, cacheType string, tenantID uuid.UUID) error {
	var msg worker.Message
	if cmd == logCmd {
		msg = worker.CreateLogCacheMessage(worker.CacheType(cacheType), tenantID)
		uclog.Infof(ctx, "Sending log cache message for cache type %v tenantID: %v", cacheType, tenantID)
	} else {
		msg = worker.CreateClearCacheMessage(worker.CacheType(cacheType), tenantID)
		uclog.Infof(ctx, "Sending clear cache message for cache type %v tenantID: %v", cacheType, tenantID)
	}
	if err := wc.Send(ctx, msg); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
