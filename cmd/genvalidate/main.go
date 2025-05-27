package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genvalidate"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genvalidate")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Fatalf(ctx, "Usage: genvalidate [type name]")
	}
	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to getwd: %v", err)
	}
	genvalidate.Run(ctx, p, wd, os.Args...)
}
