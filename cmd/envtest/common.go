package main

import (
	"context"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/resourcecheck"
)

func testResourceCheck(ctx context.Context, svc string, client *jsonclient.Client) error {
	var resp resourcecheck.Response
	if err := client.Get(ctx, "/resourcecheck", &resp); err != nil {
		return ucerr.Errorf("failed to call %v resource check endpoint: %v", svc, err)
	}
	return nil
}

func testHealthCheck(ctx context.Context, svc string, client *jsonclient.Client) error {
	var resp uclog.LocalStatus
	if err := client.Get(ctx, "/healthcheck", &resp); err != nil {
		return ucerr.Errorf("failed to call %v health check endpoint: %v", svc, err)
	}

	uclog.Debugf(ctx, "Found %d loggers", len(resp.LoggerStats))

	failed := false
	for _, ls := range resp.LoggerStats {
		if ls.FailedAPICallsCount > 0 {
			failed = true
			uclog.Errorf(ctx, "Logger %s under %v failed %d API calls", ls.Name, svc, ls.FailedAPICallsCount)
		}
	}

	if failed {
		return ucerr.Errorf("Failed API calls detected in %v", svc)
	}

	return nil
}
