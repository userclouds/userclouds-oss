package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelError, uclog.LogLevelError, "auditlogview")
	defer logtransports.Close()

	if len(os.Args) < 5 {
		uclog.Errorf(ctx, "Usage: auditlogviewer <tenant URL> <client ID> <client secret> <command> [args]")
		uclog.Debugf(ctx, "  list <num entries> - list the last <num entries> in the audit log")
		uclog.Debugf(ctx, "")
		return
	}

	tenantURL := os.Args[1]
	clientID := os.Args[2]
	clientSecret := os.Args[3]
	command := os.Args[4]

	tokenSource, err := jsonclient.ClientCredentialsForURL(tenantURL, clientID, clientSecret, nil)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create token source: %v", err)
	}
	alc, err := auditlog.NewClient(tenantURL, tokenSource)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create audit log client: %v", err)
	}

	switch command {
	case "list":
		numEntries, err := strconv.Atoi(os.Args[5])
		if err != nil {
			uclog.Fatalf(ctx, "invalid number of entries '%v': %v", os.Args[5], err)
		}
		resp, err := alc.ListEntries(ctx, auditlog.Pagination(pagination.Limit(numEntries), pagination.SortKey("created,id"), pagination.SortOrder(pagination.OrderDescending)))
		if err != nil {
			uclog.Fatalf(ctx, "failed to list audit log entries: %v", err)
		}
		for _, entry := range resp.Data {
			fmt.Printf("Time: %s\nID: %s\nType: %s\nPayload: %v\n\n", entry.Created, entry.ID, entry.Type, entry.Payload)
		}
	default:
		uclog.Fatalf(ctx, "unknown command: %s", command)
	}
}
