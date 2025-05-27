package main

import (
	"context"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
)

func healthChecks(ctx context.Context, uv universe.Universe, stopOnError bool) {
	userstoreHealthCheck(ctx, uv, stopOnError)
	// TODO (sgarrity 1/24): add authz once we expose pathed healthchecks
}

func userstoreHealthCheck(ctx context.Context, uv universe.Universe, stopOnError bool) {
	userstoreURL, err := cmdline.GetURLForUniverse(uv, "", service.IDP)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get Userstore URL: %v", err)
	}
	uclog.Infof(ctx, "Userstore base URL: %v", userstoreURL)
	client := jsonclient.New(userstoreURL)

	if err := testHealthCheck(ctx, "userstore", client); err != nil {
		if stopOnError {
			uclog.Fatalf(ctx, "Userstore: Health check failed: %v", err)
		} else {
			logErrorf(ctx, err, "Userstore: Health check failed: %v", err)
		}
	}

	if err := testResourceCheck(ctx, "userstore", client); err != nil {
		if stopOnError {
			uclog.Fatalf(ctx, "Userstore: Resource check failed: %v", err)
		} else {
			logErrorf(ctx, err, "Userstore: Resource check failed: %v", err)
		}
	}
}
