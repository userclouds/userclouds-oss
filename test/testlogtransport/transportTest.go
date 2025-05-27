package testlogtransport

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog"
)

// InitLoggerAndTransportsForTests configures logging to use golang test logging
// TODO: once we simplify log config & init, this can get unified through the main Init() path,
// but I don't want to introduce yet another config fork for this
// TODO: is there a way to do this more automatically than per-test? TestMain doesn't have testing.T or .B
// TODO: the fact that this returns a *bytes.Buffer representing the log is pretty icky
func InitLoggerAndTransportsForTests(t *testing.T) *TransportTest {
	logLevel, err := uclog.GetLogLevel(os.Getenv("TEST_LOG_LEVEL"))
	if err != nil {
		logLevel = uclog.LogLevelDebug
	}
	return InitLoggerAndTransportsForTestsWithLevel(t, logLevel)
}

// InitLoggerAndTransportsForTestsWithLevel configures logging to use golang test logging with a specific log level
func InitLoggerAndTransportsForTestsWithLevel(t *testing.T, logLevel uclog.LogLevel) *TransportTest {
	ttc := uclog.TransportConfig{
		Required:    true,
		MaxLogLevel: logLevel,
	}
	tt := TransportTest{
		t:         t,
		config:    ttc,
		testEnded: false,
	}
	transports := []uclog.Transport{&tt}
	uclog.PreInit(transports)
	t.Cleanup(tt.Close)
	return &tt
}

type testLogRecord struct {
	timestamp time.Time
	event     uclog.LogEvent
}

// TransportTest is a test log transport
type TransportTest struct {
	t           *testing.T
	config      uclog.TransportConfig
	eventMutex  sync.Mutex
	logMutex    sync.Mutex
	Events      []testLogRecord
	LogMessages map[uclog.LogLevel][]string
	testEnded   bool
}

// Init initializes the test transport
func (tt *TransportTest) Init() (*uclog.TransportConfig, error) {
	tt.Events = make([]testLogRecord, 0)
	tt.LogMessages = make(map[uclog.LogLevel][]string)
	return &tt.config, nil
}

// Write writes an event
func (tt *TransportTest) Write(ctx context.Context, event uclog.LogEvent) {
	tt.t.Helper()

	lE := testLogRecord{event: event, timestamp: time.Now().UTC()}
	tt.eventMutex.Lock()
	if event.Code != uclog.EventCodeNone {
		tt.Events = append(tt.Events, lE)
	}
	tt.eventMutex.Unlock()

	tt.logMutex.Lock()
	if !tt.testEnded && event.Message != "" {
		tt.LogMessages[event.LogLevel] = append(tt.LogMessages[event.LogLevel], event.Message)
		tt.t.Log(event.Message)
	}
	tt.logMutex.Unlock()
}

// GetEventsLoggedByName checks if a particular event has been logged
func (tt *TransportTest) GetEventsLoggedByName(name string) []uclog.LogEvent {
	tt.eventMutex.Lock()
	eA := make([]uclog.LogEvent, 0)
	for _, e := range tt.Events {
		if e.event.Name == name {
			eA = append(eA, e.event)
		}
	}
	tt.eventMutex.Unlock()
	return eA
}

// AssertMessagesByLogLevel asserts that the number of messages logged at a particular level is as expected
func (tt *TransportTest) AssertMessagesByLogLevel(level uclog.LogLevel, expected int, opts ...assert.Option) {
	tt.t.Helper()
	tt.logMutex.Lock()
	got := len(tt.LogMessages[level])
	tt.logMutex.Unlock()
	opts = append(opts, assert.Errorf("Expected %d messages at level %v, got %d", expected, level, got))
	assert.Equal(tt.t, got, expected, opts...)
}

func (tt *TransportTest) getLogMessages() []string {
	tt.logMutex.Lock()
	defer tt.logMutex.Unlock()
	messages := make([]string, 0)
	for level := range tt.LogMessages {
		messages = append(messages, tt.LogMessages[level]...)
	}
	return messages
}

// AssertLogsContainString asserts that at least one log message contains a one of the strings passed to this function
func (tt *TransportTest) AssertLogsContainString(strs ...string) {
	tt.t.Helper()
	messages := tt.getLogMessages()
	for _, m := range messages {
		for _, s := range strs {
			if strings.Contains(m, s) {
				return
			}
		}
	}
	assert.Fail(tt.t, "No log message contains string(s): %v, checked: %v", strs, messages)
}

// AssertLogsDoesntContainString asserts that no log message contains a particular string
func (tt *TransportTest) AssertLogsDoesntContainString(s string) {
	tt.t.Helper()
	messages := tt.getLogMessages()
	for _, msg := range messages {
		assert.DoesNotContain(tt.t, msg, s)
	}
}

// ClearMessages clears all logged messages
func (tt *TransportTest) ClearMessages() {
	tt.logMutex.Lock()
	tt.LogMessages = make(map[uclog.LogLevel][]string)
	tt.logMutex.Unlock()
}

// ClearEvents clears all logged events
func (tt *TransportTest) ClearEvents() {
	tt.eventMutex.Lock()
	tt.Events = make([]testLogRecord, 0)
	tt.eventMutex.Unlock()
}

// GetStats  returns stats
func (tt *TransportTest) GetStats() uclog.LogTransportStats {
	return uclog.LogTransportStats{Name: tt.GetName(), QueueSize: 0, DroppedEventCount: 0, SentEventCount: 0, FailedAPICallsCount: 3146}
}

// GetName returns transport name
func (tt *TransportTest) GetName() string {
	return "TestTransport"
}

// Flush does nothing
func (tt *TransportTest) Flush() error {
	return nil
}

// Close prevents writing to the transport after a test ends
// see https://github.com/golang/go/issues/40343
func (tt *TransportTest) Close() {
	tt.logMutex.Lock()
	defer tt.logMutex.Unlock()
	tt.testEnded = true
}
