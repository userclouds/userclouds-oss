package testhelpers

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog"
)

// RunScript runs a script in the root of the repo
func RunScript(ctx context.Context, t *testing.T, scriptPath string) {
	t.Helper()
	RunCommand(ctx, t, "bash", "-c", scriptPath)
}

// RunCommand runs a command in the root of the repo
func RunCommand(ctx context.Context, t *testing.T, cmd string, args ...string) {
	t.Helper()
	c := exec.Command(cmd, args...)
	c.Dir = getRepoRoot(t)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	uclog.Infof(ctx, "Running %s %s", cmd, strings.Join(args, " "))
	assert.NoErr(t, c.Run())
}

func getRepoRoot(t *testing.T) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	stdout, err := cmd.StdoutPipe()
	assert.NoErr(t, err)
	assert.NoErr(t, cmd.Start())
	output, err := io.ReadAll(stdout)
	assert.NoErr(t, err)
	// Trim the newline from the command output
	return strings.TrimSpace(string(output))
}
