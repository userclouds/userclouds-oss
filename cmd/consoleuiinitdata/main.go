package main

import (
	"context"
	"flag"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/uiinitdata"
)

const (
	indexFile = "console/consoleui/build/index.html"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "consoleuiinitdata", logtransports.NoPrefix())
	defer logtransports.Close()
	dryRunFlag := flag.Bool("dry-run", false, "Do everything but don't actually write the file")
	flag.Parse()
	dryRun := *dryRunFlag
	uv := universe.Current()
	// see: tools/packaging/build-and-upload.sh
	if os.Getenv("SKIP_UI_INIT_DATA_INJECTION") == "true" {
		uclog.Infof(ctx, "Skipping UI init data injection")
		return
	}
	if uv.IsCloud() || uv.IsDev() {
		uclog.Infof(ctx, "Injecting init data into console ui. universe: %v dry run: %v", uv, dryRun)
		if err := uiinitdata.LoadAndInjectAppInitData(ctx, indexFile, true, dryRun); err != nil {
			uclog.Fatalf(ctx, "failed to inject init data: %v", err)
		}
	} else if uv.IsOnPremOrContainer() {
		uclog.Infof(ctx, "Clear ucAppInitData from %s for '%v' universe", indexFile, uv)
		if err := uiinitdata.ClearUIUnitData(ctx, indexFile, dryRun); err != nil {
			uclog.Fatalf(ctx, "failed to inject init data: %v", err)
		}
	} else {
		uclog.Infof(ctx, "Skipping init data injection for '%v' universe", uv)
	}
}
