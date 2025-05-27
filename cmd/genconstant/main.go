package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genconstant"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genconstant")
	defer logtransports.Close()

	if len(os.Args) != 2 {
		uclog.Fatalf(ctx, "Usage: genconstant [typename]")
	}

	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genconstant.Run(ctx, p, wd, os.Args...)
}
