package logtransports

// Transport directing event stream to stdout, where each event is printed as a JSON object

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func init() {
	registerDecoder(TransportTypeGoLogJSON, func(value []byte) (TransportConfig, error) {
		var k GoLogJSONTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &k); err == nil && k.Type == TransportTypeGoLogJSON {
			return &k, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// TransportTypeGoLogJSON defines the GoLogJSON transport
const TransportTypeGoLogJSON TransportType = "goLogJSON"

// GoLogJSONTransportConfig defines logger client config
type GoLogJSONTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
}

// GetType implements TransportConfig
func (c GoLogJSONTransportConfig) GetType() TransportType {
	return TransportTypeGoLogJSON
}

// IsSingleton implements TransportConfig
func (c GoLogJSONTransportConfig) IsSingleton() bool {
	return true
}

// GetUnwrappedTransport gets a transport that doesn't wrap the GoLogJSONTransport in a background I/O wrapper
func (c GoLogJSONTransportConfig) GetUnwrappedTransport(name service.Service, _ jsonclient.Option, _ string) uclog.Transport {
	return newGoLogJSONTransport(&c, name)
}

// GetWrappedTransport gets a transport that wraps the GoLogJSONTransport in a background I/O wrapper
func (c GoLogJSONTransportConfig) GetWrappedTransport(name service.Service, _ jsonclient.Option, _ string) uclog.Transport {
	return newTransportBackgroundIOWrapper(newGoLogJSONTransport(&c, name))
}

// GetTransport implements TransportConfig
func (c GoLogJSONTransportConfig) GetTransport(name service.Service, opt jsonclient.Option, s string) uclog.Transport {
	return c.GetUnwrappedTransport(name, opt, s)
}

// Validate implements Validateable
func (c *GoLogJSONTransportConfig) Validate() error {
	return nil
}

const (
	goLogJSONTransportName = "GoLogJSONTransport"
	// this needs to be set low because transportBackgroundIOWrapper will drop events if it doesn't flush often enough
	goLogJSONSendInterval time.Duration = 10 * time.Millisecond
)

type goLogJSONTransport struct {
	config         GoLogJSONTransportConfig
	service        service.Service
	sentEventCount int64
	failedAPICalls int64
}

func newGoLogJSONTransport(c *GoLogJSONTransportConfig, name service.Service) *goLogJSONTransport {
	return &goLogJSONTransport{config: *c, service: name}
}

func (t *goLogJSONTransport) init(ctx context.Context) (*uclog.TransportConfig, error) {
	c := &uclog.TransportConfig{Required: t.config.Required, MaxLogLevel: t.config.MaxLogLevel}
	return c, nil
}

func (t *goLogJSONTransport) Write(ctx context.Context, event uclog.LogEvent) {
	tm := time.Now().UTC()
	t.writeMessages(ctx, &logRecord{event: event, timestamp: tm}, tm, 1)
}

func (t *goLogJSONTransport) Close() {
	t.closeIOResources() // this is a no-op but just in case
}

func (t *goLogJSONTransport) Flush() error {
	t.flushIOResources() // this is functionally a no-op but just in case
	return nil
}

func (t *goLogJSONTransport) GetName() string {
	return t.getTransportName()
}

func (t *goLogJSONTransport) GetStats() uclog.LogTransportStats {
	return uclog.LogTransportStats{
		Name:                t.getTransportName(),
		QueueSize:           0,
		DroppedEventCount:   0,
		SentEventCount:      t.sentEventCount,
		FailedAPICallsCount: t.getFailedAPICallsCount(),
	}
}

func (t *goLogJSONTransport) Init() (*uclog.TransportConfig, error) {
	return t.init(context.Background())
}

// JSONLogLine defines the JSON format for a GoLogJSONTransport log line
type JSONLogLine struct {
	TimestampNS int64     `json:"time_ns"`
	LogLevel    string    `json:"level"`
	EventName   string    `json:"event_name"`
	Count       int       `json:"count"`   // Reporting multiple events at once
	Message     string    `json:"message"` // Message associated with the event
	Payload     string    `json:"payload"` // Optional payload associated with a counter event
	UserAgent   string    `json:"user_agent"`
	RequestID   uuid.UUID `json:"request_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
}

func (t *goLogJSONTransport) writeMessages(ctx context.Context, logRecords *logRecord, startTime time.Time, count int) {
	lines := []JSONLogLine{}

	currRecord := logRecords
	for currRecord != nil {
		lines = append(lines, JSONLogLine{
			TimestampNS: currRecord.timestamp.UnixNano(),
			LogLevel:    currRecord.event.LogLevel.String(),
			EventName:   currRecord.event.Name,
			Count:       currRecord.event.Count,
			Message:     currRecord.event.Message,
			Payload:     currRecord.event.Payload,
			UserAgent:   currRecord.event.UserAgent,
			RequestID:   currRecord.event.RequestID,
			TenantID:    currRecord.event.TenantID,
		})
		currRecord = currRecord.next
	}

	var outBytes []byte
	for _, line := range lines {
		jsonVal, err := json.Marshal(line)
		if err != nil {
			failedCalls.WithLabelValues(string(TransportTypeGoLogJSON), "json_marshal").Inc()
			uclog.Errorf(ctx, "Error marshalling JSON for %+v: %v\n", line, ucerr.Wrap(err))
			// Also print directly to stderr in case we can't get the above
			// error out due to the marshalling errors
			fmt.Fprintf(os.Stderr, "Error marshalling JSON for %+v: %v\n", line, ucerr.Wrap(err))
			t.failedAPICalls++
		}
		outBytes = append(outBytes, jsonVal...)
		outBytes = append(outBytes, byte('\n'))
	}

	// Print everything to stderr.
	// Note: We don't want to print errors/warnings to stderr and debug to
	// stdout because if stdout/stderr are buffered differently (as is usually
	// the case), then the output lines won't necessarily be in order (e.g. you
	// could have an error printed before a debug log that chronologically
	// preceded it).
	if _, err := os.Stderr.Write(outBytes); err != nil {
		uclog.Errorf(ctx, "Error writing output: %v\n", ucerr.Wrap(err))
		failedCalls.WithLabelValues(string(TransportTypeGoLogJSON), "write").Inc()
		t.failedAPICalls++
	} else {
		successfulCalls.WithLabelValues(string(TransportTypeGoLogJSON)).Inc()
		t.sentEventCount++
	}
}

func (t *goLogJSONTransport) getFailedAPICallsCount() int64 {
	return t.failedAPICalls
}

func (t *goLogJSONTransport) getIOInterval() time.Duration {
	return goLogJSONSendInterval
}

func (t *goLogJSONTransport) getMaxLogLevel() uclog.LogLevel {
	return t.config.MaxLogLevel
}

func (t *goLogJSONTransport) supportsCounters() bool {
	return false
}

func (t *goLogJSONTransport) getTransportName() string {
	return goLogJSONTransportName
}

func (t *goLogJSONTransport) flushIOResources() {
	// Shouldn't be necessary since stderr is unbuffered, but shouldn't hurt either
	os.Stderr.Sync()
}

func (t *goLogJSONTransport) closeIOResources() {
	// Nothing to do
}
