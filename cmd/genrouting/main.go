package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate/genrouting"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genrouting")
	defer logtransports.Close()

	if len(os.Args) > 1 {
		uclog.Fatalf(ctx, "Usage: genrouting")
	}

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genrouting.Run(ctx, wd)

}
