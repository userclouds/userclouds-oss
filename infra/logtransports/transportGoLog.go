package logtransports

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/color"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Basic transport redirecting event stream to the Go logger

func init() {
	registerDecoder(TransportTypeGo, func(value []byte) (TransportConfig, error) {
		var g GoTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &g); err == nil && g.Type == TransportTypeGo {
			return &g, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// TransportTypeGo defines the Go transport
const TransportTypeGo TransportType = "go"

// GoTransportConfig defines go logger client config
type GoTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
	PrefixFlag            int  `yaml:"prefix_flag" json:"prefix_flag"`
	SupportsColor         bool `yaml:"supports_color" json:"supports_color"`
	NoRequestIDs          bool `yaml:"no_request_ids" json:"no_request_ids"`
}

// GetType implements TransportConfig
func (c GoTransportConfig) GetType() TransportType {
	return TransportTypeGo
}

// IsSingleton implements TransportConfig
func (c GoTransportConfig) IsSingleton() bool {
	return true
}

// GetTransport implements TransportConfig
func (c GoTransportConfig) GetTransport(_ service.Service, _ jsonclient.Option, _ string) uclog.Transport {
	return newGoTransport(&c)
}

// Validate implements Validateable
func (c *GoTransportConfig) Validate() error {
	return nil
}

// DefaultPrefixVal is a constant indicating that default Go prefix should be used
const DefaultPrefixVal = 0

// NoPrefixVal is a constant indicating that no prefix should be used
const NoPrefixVal = -1

const goTransportName = "GoTransport"

const (
	defaultColor = color.Default
	errorColor   = color.BrightRed
	warningColor = color.Yellow
)

type logTransport struct {
	config GoTransportConfig
}

func newGoTransport(c *GoTransportConfig) *logTransport {
	var t = logTransport{}
	t.config = *c
	return &t
}

func (t *logTransport) Init() (*uclog.TransportConfig, error) {
	// Configure the logger
	log.SetOutput(os.Stdout)

	// confusingly, golang log package uses a prefix of 0 to mean no prefix,
	// but we want our default to be the default prefix, so we need to switch
	// these constants here to make our default actually be the default (0)
	// TODO (sgarrity 8/23): there has to be a better factoring here generally
	// for this sort-of-random edge case of the go logger in tools.
	if t.config.PrefixFlag == NoPrefixVal {
		log.SetFlags(0)
	} else if t.config.PrefixFlag != DefaultPrefixVal {
		log.SetPrefix(fmt.Sprintf("YOU NEED TO UPDATE THE LOGGING FLAGS FOR %v ", t.config.PrefixFlag))
	}

	return &uclog.TransportConfig{Required: t.config.Required, MaxLogLevel: t.config.MaxLogLevel}, nil
}

func (t *logTransport) Write(ctx context.Context, event uclog.LogEvent) {
	// Go transport doesn't record counters or payloads so just record the message if there is one
	if event.Message == "" || event.LogLevel > t.config.MaxLogLevel {
		return
	}

	messageColor := defaultColor

	switch event.LogLevel {
	case uclog.LogLevelError:
		messageColor = errorColor
	case uclog.LogLevelWarning:
		messageColor = warningColor
	}

	message := event.Message
	if !t.config.NoRequestIDs && !event.RequestID.IsNil() {
		message = fmt.Sprintf("%v: %s", event.RequestID, message)
	}

	// TODO: there's a cleaner factoring here but
	if messageColor != defaultColor && t.config.SupportsColor {
		log.Printf("%s%s%s%s%s", color.ANSIEscapeColor, messageColor, message, color.ANSIEscapeColor, defaultColor) // lint: ucwrapper-safe
	} else {
		log.Println(message) // lint: ucwrapper-safe
	}
}

func (t *logTransport) GetStats() uclog.LogTransportStats {
	return uclog.LogTransportStats{Name: t.GetName(), QueueSize: 0, DroppedEventCount: 0, SentEventCount: 0, FailedAPICallsCount: 0}
}

func (t *logTransport) GetName() string {
	return goTransportName
}

func (t *logTransport) Flush() error {
	return nil
}

func (t *logTransport) Close() {
}
