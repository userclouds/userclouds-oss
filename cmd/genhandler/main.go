package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genhandler"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelInfo, uclog.LogLevelVerbose, "genhandler")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Debugf(ctx, "Usage: genhandler <service_path> <handler_info>*")
		uclog.Debugf(ctx, "  <handler_info> = { collection,[name],[authorizer],[path] | nestedcollection,[name],[authorizer],[path],[parent_name] | [action],[name],[path] }")
		uclog.Debugf(ctx, "  action = { GET | POST | PUT | DELETE } ")
		uclog.Fatalf(ctx, "Expected list of handler info to generate")
	}

	p := generate.GetPackage()

	wd, err := os.Getwd()
	if err != nil {
		uclog.Fatalf(ctx, "failed to get wd: %v", err)
	}

	genhandler.Run(ctx, p, wd, os.Args...)
}
