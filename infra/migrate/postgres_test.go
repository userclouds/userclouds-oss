package migrate

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
)

// GetTableSchema is a helper function to get the schema of a table in a test database
func GetTableSchema(ctx context.Context, t *testing.T, db *ucdb.DB, table string) string {
	// postgres doesn't support `SHOW CREATE TABLE`	so we run
	// pg_dump on the same container that's already hosting postgres for us
	cfg := testdb.TestConfig(t, db)
	pass, err := cfg.Password.Resolve(ctx)
	assert.NoErr(t, err)
	cmd := exec.Command("docker", "exec", "-t", "testdb",
		"env", fmt.Sprintf("PGPASSWORD=%s", pass),
		"pg_dump", "-U", cfg.User, "-d", cfg.DBName,
		"-t", table, "-s",
		"--no-comments", "--no-owner", "--no-privilege")
	output, err := cmd.Output()

	// this preserves previous behavior of of migrate_test.getSchema
	if strings.Contains(string(output), "pg_dump: error: no matching tables were found") {
		return "table does not exist"
	}
	assert.NoErr(t, err, assert.Errorf("failed to get schema for %s: %s", table, string(output)))

	// Extract just the CREATE TABLE statement
	r := regexp.MustCompile(`(?s)CREATE TABLE.*?;`)
	match := r.Find(output)
	assert.NotNil(t, match)

	ct := string(match)

	return ct
}
