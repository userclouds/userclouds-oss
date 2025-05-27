package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genpageable"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genpageable")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Fatalf(ctx, "Usage: genpageable [type name]")
	}
	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to getwd: %v", err)
	}
	genpageable.Run(ctx, p, wd, os.Args...)
}
