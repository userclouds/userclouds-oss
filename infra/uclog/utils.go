package uclog

import (
	"context"
	"fmt"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
)

// ServiceStartupInfo is used to communicate server status in startup events
// TODO put this into a better place
type ServiceStartupInfo struct {
	Region      region.MachineRegion
	Hostname    string
	CodeVersion string
}

// These two types are shared between kinesis logger and readers of the kinesis stream. It doesn't make sense to create a package just for these two types.
// so for now keeping them here

// ToolPromptf logs a string at info level, but avoids a newline on the go logger (on dev console)
// Note that this bypasses all of the normal logging logic so we don't pollute everything with a
// weird "no newlines in console logging" bit, but it's really only used one place.
// Whenever we add eg. a file logger to Tools, we'll have to update this accordingly
func ToolPromptf(ctx context.Context, f string, args ...any) {
	// fail loudly if logging config doesn't match what we expect, since
	// we're working around the normal path here to make prompts easier to use,
	// but not pollute the logging infra with "no newline" flags
	if loggerInst.loggerState != loggerToolMode {
		Fatalf(context.Background(), "can't use ToolPromptf without InitForTools()")
	}

	fmt.Printf(f, args...)
}

// DebugfPII logs a string with optional format-string parsing, except when running in prod.
// by default these are internal-to-Userclouds logs
func DebugfPII(ctx context.Context, f string, args ...any) {
	if universe.Current().IsProd() {
		return
	}
	logWithLevelf(ctx, LogLevelDebug, f, args...)
}

// InfofPII logs a string with optional format-string parsing, except when running in prod.
// by default these are internal-to-Userclouds logs
func InfofPII(ctx context.Context, f string, args ...any) {
	if universe.Current().IsProd() {
		return
	}
	logWithLevelf(ctx, LogLevelInfo, f, args...)
}
