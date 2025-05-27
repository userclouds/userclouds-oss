package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genorm"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genorm")
	defer logtransports.Close()

	// we take the table name explicitly to avoid RoR-style magic
	if len(os.Args) < 3 {
		uclog.Debugf(ctx, "Usage: genorm [flags] <type name> <table name> <db name>")
		uclog.Debugf(ctx, "  <db name>: tenantdb | companyconfig | logdb")
		uclog.Debugf(ctx, "  --noget: don't generate Get()")
		uclog.Debugf(ctx, "  --nolist: don't generate List()")
		uclog.Debugf(ctx, "  --columnlistonly: don't generate methods, only the column list for verification")
		uclog.Fatalf(ctx, "Expected three arguments, instead got %d", len(os.Args))
	}

	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genorm.Run(ctx, p, wd, os.Args...)
}
