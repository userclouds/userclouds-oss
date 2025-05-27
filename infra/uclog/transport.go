package uclog

import (
	"context"
)

// This defines the interface that the logger expect to sends data to the raw event store
// The transport layer delegates error handling to the logger
// The transport layer is expected to handle buffering optimization as needed for that transport
// The transport layer has to be thread safe with respect to all methods defined in transport interface

// TransportConfig defines the shared config for log transports
type TransportConfig struct {
	Required    bool     `yaml:"required" json:"required"`
	MaxLogLevel LogLevel `yaml:"max_log_level" json:"max_log_level"`
}

// LogTransportStats contains statistics about transport operation
type LogTransportStats struct {
	Name                string
	QueueSize           int64
	DroppedEventCount   int64
	SentEventCount      int64
	FailedAPICallsCount int64
}

// Transport defines the interface loggers implement
type Transport interface {
	Init() (*TransportConfig, error)
	Write(ctx context.Context, event LogEvent)
	GetStats() LogTransportStats
	GetName() string
	Flush() error
	Close()
}
