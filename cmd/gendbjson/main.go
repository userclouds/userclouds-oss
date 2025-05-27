package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/gendbjson"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "gendbjson")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Fatalf(ctx, "Usage: gendbjson [type name]")
	}

	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	gendbjson.Run(ctx, p, wd, os.Args...)
}
