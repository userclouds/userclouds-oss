package main

import (
	"context"
	"database/sql"
	"os"
	"time"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "dbstatsclient")
	defer logtransports.Close()

	if len(os.Args) < 4 {
		uclog.Fatalf(ctx, "Usage: go run cmd/dbstatsclient/main.go [url] [client id] [client secret]")
	}

	url := os.Args[1]

	ts, err := jsonclient.ClientCredentialsForURL(url, os.Args[2], os.Args[3], nil)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create token source: %v", err)
	}

	client := jsonclient.New(url, ts)

	var stats sql.DBStats
	for {
		if err := client.Get(ctx, "/tokenizer/dbstats", &stats); err != nil {
			uclog.Errorf(ctx, "get: %v", err)
		}
		uclog.Infof(ctx, "%+v", stats)

		time.Sleep(time.Second)
	}
}
