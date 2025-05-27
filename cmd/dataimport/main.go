package main

import (
	"context"
	"os"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/dataimport"
	"userclouds.com/internal/tenantmap"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "dataimport")
	defer logtransports.Close()

	if len(os.Args) < 3 {
		uclog.Debugf(ctx, "Usage: dataimport <tenantID> <filePath>")
		uclog.Debugf(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "Expected tenantID and filePath, instead got %d: %v", len(os.Args), os.Args)
	}

	tenantID := uuid.Must(uuid.FromString(os.Args[1]))
	ccs := cmdline.GetCompanyStorage(ctx)
	tm := tenantmap.NewStateMap(ccs, nil)
	ts, err := tm.GetTenantStateForID(ctx, tenantID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to connect to tenant %s: %v", tenantID, err)
	}

	if err := dataimport.ImportDataFromFile(ctx, os.Args[2], ts); err != nil {
		uclog.Fatalf(ctx, "Failed to import data: %v", err)
	}
}
