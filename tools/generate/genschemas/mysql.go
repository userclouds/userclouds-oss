package genschemas

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/jmoiron/sqlx"

	"userclouds.com/infra/uclog"
)

// StartTemporaryMysql starts a temporary mysql instance
// and returns the command, the connection string, the docker container name, and the port
func StartTemporaryMysql(ctx context.Context, namePrefix string, portPrefix int) (*exec.Cmd, string, string, int) {
	cmd, name, port := StartContainer(ctx, namePrefix, portPrefix, 3306, "MYSQL_ROOT_PASSWORD", "mysql")

	// wait for the database to start
	uclog.Infof(ctx, "waiting for mysql to start")
	maxTries := 20
	for i := range maxTries {
		db, err := sqlx.Open("mysql", fmt.Sprintf("root:mysecretpassword@tcp(127.0.0.1:%d)/sys", port))
		if err != nil {
			// unusual error
			uclog.Fatalf(ctx, "failed to connect to mysql: %v", err)
		}

		// normally errors don't show up until ping
		if err := db.Ping(); err == nil {
			break
		}

		// give up if we've hit max
		if i == maxTries-1 {
			uclog.Fatalf(ctx, "failed to connect to mysql: %v", err)
		}
		time.Sleep(time.Second)
	}

	return cmd, fmt.Sprintf("mysql://root:mysecretpassword@127.0.0.1:%d/sys?sslmode=disable", port), name, port
}
