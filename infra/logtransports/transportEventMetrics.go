package logtransports

import (
	"context"
	"encoding/json"
	"strings"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
)

const (
	eventMetricsTransportName = "EventMetricsTransport"
	eventSubsystem            = ucmetrics.Subsystem("event")
	// TransportTypeEventMetrics defines the EventMetrics transport
	TransportTypeEventMetrics TransportType = "eventMetrics"
)

var (
	metricEventCount           = ucmetrics.CreateCounter(eventSubsystem, "count", "The total number of events", "event_name", "event_category", "event_subcategory", "tenant_id")
	metricEventDurationSeconds = ucmetrics.CreateHistogram(eventSubsystem, "duration_seconds", "time taken to event", "event_name", "event_subcategory", "tenant_id")
)

func init() {
	registerDecoder(TransportTypeEventMetrics, func(value []byte) (TransportConfig, error) {
		var tc EventMetricsTransportConfig
		// NB: we need to check the type here because the yaml decoder will happily decode an
		// empty struct, since dec.KnownFields(true) gets lost via the yaml.Unmarshaler
		// interface implementation
		if err := json.Unmarshal(value, &tc); err == nil && tc.Type == TransportTypeEventMetrics {
			return &tc, nil
		}
		return nil, ucerr.New("Unknown transport type")
	})
}

// EventMetricsTransportConfig is the configuration for the EventMetrics transport
type EventMetricsTransportConfig struct {
	Type                  TransportType `yaml:"type" json:"type"`
	uclog.TransportConfig `yaml:"transportconfig" json:"transportconfig"`
}

// GetType implements TransportConfig
func (c EventMetricsTransportConfig) GetType() TransportType {
	return TransportTypeEventMetrics
}

// IsSingleton implements TransportConfig
func (c EventMetricsTransportConfig) IsSingleton() bool {
	return true
}

// GetTransport implements TransportConfig
func (c EventMetricsTransportConfig) GetTransport(name service.Service, _ jsonclient.Option, _ string) uclog.Transport {
	return newEventMetricsTransport(&c, name)
}

// Validate implements Validateable
func (c *EventMetricsTransportConfig) Validate() error {
	return nil
}

type eventMetricsTransport struct {
	config EventMetricsTransportConfig
}

func newEventMetricsTransport(c *EventMetricsTransportConfig, _ service.Service) *eventMetricsTransport {
	return &eventMetricsTransport{config: *c}
}

func (t *eventMetricsTransport) Close() {
}
func (t *eventMetricsTransport) Flush() error {
	return nil
}

func (t *eventMetricsTransport) GetStats() uclog.LogTransportStats {
	return uclog.LogTransportStats{Name: t.GetName(), QueueSize: 0, DroppedEventCount: 0, SentEventCount: 0, FailedAPICallsCount: 0}
}

func (t *eventMetricsTransport) Write(ctx context.Context, event uclog.LogEvent) {
	if event.Code == uclog.EventCodeNone {
		return
	}
	eventType := uclog.GetEventInfo(event)
	if eventType.Code == uclog.EventCodeUnknown {
		return
	}
	var tenantID string
	if !event.TenantID.IsNil() {
		tenantID = event.TenantID.String()
	}
	if eventType.Category == uclog.EventCategoryDuration {
		// e.Count is in Milliseconds (see: idp/internal/userstore/eventhelpers.go logExecutionDuration)
		metricEventDurationSeconds.WithLabelValues(getName(eventType), eventType.Subcategory, tenantID).Observe(float64(event.Count) / 1000.0)
		return
	}
	metricEventCount.WithLabelValues(getName(eventType), string(eventType.Category), eventType.Subcategory, tenantID).Inc()
}

func getName(t uclog.LogEventTypeInfo) string {
	if t.NormalizedName != "" {
		return t.NormalizedName
	}
	return strings.ReplaceAll(t.Name, " ", "")
}

func (t *eventMetricsTransport) Init() (*uclog.TransportConfig, error) {
	return &t.config.TransportConfig, nil
}

func (t *eventMetricsTransport) GetName() string {
	return eventMetricsTransportName
}
