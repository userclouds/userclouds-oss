package async

import (
	"context"
	"runtime/debug"

	"userclouds.com/infra/uclog"
)

// Execute a wrapper around go routine to ensure that we capture a panic in the logs
func Execute(fn func()) {
	go func() {
		defer recoverPanic()
		fn()
	}()
}

// recoverPanic only executes code in case of panic and ensures that our logs capture panics
func recoverPanic() {
	if r := recover(); r != nil {

		// This will not return
		uclog.Fatalf(context.Background(), "Panic: %v Stack %s", r, string(debug.Stack()))

	}
}
