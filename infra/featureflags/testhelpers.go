package featureflags

import (
	"context"

	statsig "github.com/statsig-io/go-sdk"

	"userclouds.com/infra/secret"
)

func initForTests() {
	ctx := context.Background()
	notInitWarningLogged = true
	// Calling Init here so the calling test doesn't have to.
	if !isInitialized(ctx) {
		Init(ctx, &Config{APIKey: secret.NewTestString("no-soup-for-you")})
	}
}

// EnableFlagsForTest enables a flags for the duration of a test.
func EnableFlagsForTest(flags ...Flag) {
	setFlagsForTest(true, flags)
}

// DisableFlagsForTest disables a flags for the duration of a test.
func DisableFlagsForTest(flags ...Flag) {
	setFlagsForTest(false, flags)
}

func setFlagsForTest(value bool, flags []Flag) {
	// see: https://docs.statsig.com/server/golangSDK#local-overrides (we use local mode in tests & CI)
	for _, flag := range flags {
		statsig.OverrideGate(string(flag), value)
	}
}
