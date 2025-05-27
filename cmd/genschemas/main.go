package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genschemas"
)

// this command exists to generate our DB schemas up to max(migrations) so that we can
// apply them directly in tests, rather than migrating forward every time (migration tests
// obviously won't follow this rule) for performance reasons.

// there's a weird tradeoff here between running this as a test or codegen
// codegen is more natural in many ways (it's generating code, after all), but
// we are using testdb for the actual generation. Without running in a test env,
// we have to spin up our own temp DB instances etc, which is a hassle.

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "genschemas")
	defer logtransports.Close()

	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genschemas.Run(ctx, p, wd, os.Args...)
}
