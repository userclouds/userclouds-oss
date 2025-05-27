package main

import (
	"context"
	"os"
	"time"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/multirun"
	"userclouds.com/infra/namespace/color"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/servicecolors"
	"userclouds.com/tools/uiinitdata"
)

var cmds = []multirun.Command{
	{Bin: "bin/devlb", Color: color.BrightYellow},
	{Bin: "bin/logserver"},
	{Bin: "bin/plex"},
	{Bin: "bin/authz"},
	{Bin: "bin/checkattribute"},
	{Bin: "bin/idp"},
	{Bin: "bin/console"},
	{Bin: "bin/worker"},
}

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "devbox", logtransports.NoPrefix())
	defer logtransports.Close()

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		go func() {
			time.Sleep(15 * time.Second) // give time for other services to start and make sure the warning is visible
			uclog.Warningf(ctx, "******************************************************************************")
			uclog.Warningf(ctx, "AWS creds not set for dev - please run `make ensure-secrets-dev`")
			uclog.Warningf(ctx, "******************************************************************************")
		}()
	}
	for i, cmd := range cmds {
		if cmds[i].Color == "" {
			cmds[i].Color = servicecolors.MustGetColor(ctx, cmd.GetName())
		}
	}

	if err := uiinitdata.LoadAndInjectAppInitData(ctx, "console/consoleui/build/index.html", false, false); err != nil {
		uclog.Fatalf(ctx, "failed to inject init data: %v", err)
	}

	if err := multirun.SetupCommands(ctx, cmds); err != nil {
		uclog.Fatalf(ctx, "failed to setup commands: %v", err)
	}
	env := multirun.NewEnv(ctx, cmds)
	for _, c := range cmds {
		multirun.WrapOutputs(c, env)
	}
	if err := multirun.Run(ctx, cmds, env); err != nil {
		uclog.Fatalf(ctx, "failed to run: %v", err)
	}
}
