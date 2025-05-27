package cmdline

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
)

// ReadConsole is a helper function to output a prompt and get user input
func ReadConsole(ctx context.Context, fstr string, args ...any) string {
	uclog.ToolPromptf(ctx, fstr, args...)
	rdr := bufio.NewReader(os.Stdin)
	r, err := rdr.ReadString('\n')
	r = strings.TrimSpace(r)
	if err != nil {
		uclog.Fatalf(ctx, "failed to read response from console: %v", err)
	}
	return r
}

// Confirm returns true/false based on input in response to prompt
func Confirm(ctx context.Context, fstr string, args ...any) bool {
	conf := ReadConsole(ctx, fstr, args...)
	return strings.ToLower(conf) == "y"
}

// ProductionConfirm makes you really confirm something in prod
// if you're not in prod, it just returns true :)
func ProductionConfirm(ctx context.Context, acknowledge, fstr string, args ...any) bool {
	if !universe.Current().IsProd() {
		return true
	}
	conf := ReadConsole(ctx, fmt.Sprintf("[PRODUCTION] %s", fstr), args...)
	return strings.EqualFold(conf, acknowledge)
}
