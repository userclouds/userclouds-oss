// genstringconstenum is a poorly named tool to generate an array var with all of the
// valid constants of type arg[1], for eg validation

package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genstringconstenum"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "gentstringconstenum")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Fatalf(ctx, "Usage: genstringconstenum [type name]")
	}

	p := generate.GetPackage()
	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genstringconstenum.Run(ctx, p, wd, os.Args...)
}
