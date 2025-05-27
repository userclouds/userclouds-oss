package uclog

import (
	"context"
	"fmt"
	"os"
)

//  A set of wrappers that log messages at a pre-set level

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelError, f, args...)
	// Because os.Exit doesn't run deferred functions close the transports before calling it so the
	// last messages end up in the log
	Close()
	os.Exit(1)
}

// Errorf logs an error with optional format-string parsing
func Errorf(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelError, f, args...)
}

// Warningf logs a string at info level (default visible in user console)
func Warningf(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelWarning, f, args...)
}

// Infof logs a string at info level (default visible in user console)
func Infof(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelInfo, f, args...)
}

// Debugf logs a string with optional format-string parsing
// by default these are internal-to-Userclouds logs
func Debugf(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelDebug, f, args...)
}

// Verbosef is the loudest
// Originally introduced to log DB queries / timing without killing dev console
func Verbosef(ctx context.Context, f string, args ...any) {
	logWithLevelf(ctx, LogLevelVerbose, f, args...)
}

func logWithLevelf(ctx context.Context, level LogLevel, f string, args ...any) {
	s := fmt.Sprintf(f, args...)
	s = fmt.Sprintf("[%s] %s", level.GetPrefix(), s)
	Log(ctx, LogEvent{LogLevel: level, Code: EventCodeNone, Message: s, Count: 1})
}

// A set of wrappers that log counter events

// IncrementEvent records a UserClouds event without message or payload
func IncrementEvent(ctx context.Context, eventName string) {
	e := LogEvent{LogLevel: LogLevelNonMessage, Name: eventName, Count: 1}
	Log(ctx, e)
}

// IncrementEventWithPayload logs event related to security that carry a payload
func IncrementEventWithPayload(ctx context.Context, eventName string, payload string) {
	Log(ctx, LogEvent{LogLevel: LogLevelNonMessage, Name: eventName, Payload: payload, Count: 1})
}
