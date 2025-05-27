package genschemas

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"testing"
	"time"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
)

type noopValidator struct{}

func (v *noopValidator) Validate(ctx context.Context, db *ucdb.DB) error {
	return nil
}

// StartTemporaryPostgres starts a temporary postgres instance
// and returns the command, the connection string, the docker container name, and the port
func StartTemporaryPostgres(ctx context.Context, namePrefix string, portPrefix int) (*exec.Cmd, string, string, int) {
	cmd, name, port := StartContainer(ctx, namePrefix, portPrefix, 5432, "POSTGRES_PASSWORD", "postgres")

	// wait for the db to be ready
	cfg := ucdb.Config{
		Host:      "127.0.0.1",
		Port:      fmt.Sprintf("%d", port),
		User:      "postgres",
		Password:  secret.NewTestString("mysecretpassword"),
		DBName:    "postgres",
		DBDriver:  "postgres",
		DBProduct: ucdb.Postgres,
	}

	tries := 0
	for {
		conn, err := ucdb.New(ctx, &cfg, &noopValidator{})
		if err == nil {
			if err := conn.Ping(); err == nil {
				if err := conn.Close(ctx); err != nil {
					uclog.Fatalf(ctx, "Error closing Postgres DB container: %v", err)
				}
				break
			}
		}
		tries++
		if tries > 40 {
			uclog.Fatalf(ctx, "Failed to connect to Postgres DB container")
		}
		time.Sleep(500 * time.Millisecond)
	}

	return cmd, fmt.Sprintf("postgres://postgres:mysecretpassword@127.0.0.1:%d/postgres?sslmode=disable", port), name, port
}

// ConfigFromConnectionString generates a DB config object from the given connection string
func ConfigFromConnectionString(t *testing.T, connStr string) ucdb.Config {
	connURL, err := url.Parse(connStr)
	assert.NoErr(t, err)

	cfg := ucdb.Config{
		User:      connURL.User.Username(),
		Host:      connURL.Hostname(),
		Port:      connURL.Port(),
		DBName:    connURL.Path,
		DBDriver:  ucdb.PostgresDriver,
		DBProduct: ucdb.Postgres,
	}

	if pass, ok := connURL.User.Password(); ok {
		cfg.Password = secret.NewTestString(pass)
	}

	return cfg
}
