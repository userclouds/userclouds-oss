package uclog

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
)

// LogLevel represent the urgency level of the logging event
type LogLevel int

// Different log levels
const (
	LogLevelNone       LogLevel = -1
	LogLevelNonMessage LogLevel = 0
	LogLevelError      LogLevel = 1
	LogLevelWarning    LogLevel = 2
	LogLevelInfo       LogLevel = 3
	LogLevelDebug      LogLevel = 4
	LogLevelVerbose    LogLevel = 5
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelNone:
		return "none"
	case LogLevelNonMessage:
		return "nonMessage"
	case LogLevelError:
		return "error"
	case LogLevelWarning:
		return "warn"
	case LogLevelInfo:
		return "info"
	case LogLevelDebug:
		return "debug"
	case LogLevelVerbose:
		return "verbose"
	default:
		return fmt.Sprintf("unknown - %d", l)
	}
}

// GetPrefix returns the single character prefix for the log level used to identify the log level in logs
func (l LogLevel) GetPrefix() string {
	switch l {
	case LogLevelNone:
		fallthrough
	case LogLevelNonMessage:
		return "N"
	case LogLevelVerbose:
		return "V"
	case LogLevelDebug:
		return "D"
	case LogLevelInfo:
		return "I"
	case LogLevelWarning:
		return "W"
	case LogLevelError:
		return "E"
	default:
		return "U"
	}
}

// GetLogLevel returns the log level for a given string log level name
func GetLogLevel(name string) (LogLevel, error) {
	switch strings.ToLower(name) {
	case "none":
		return LogLevelNone, nil
	case "nonmessage":
		return LogLevelNonMessage, nil
	case "error":
		return LogLevelError, nil
	case "warning":
		return LogLevelWarning, nil
	case "info":
		return LogLevelInfo, nil
	case "debug":
		return LogLevelDebug, nil
	case "verbose":
		return LogLevelVerbose, nil
	default:
		return LogLevelNone, ucerr.Errorf("unknown log level %s", name)
	}
}

// Two baseline events that have to be in the map
const (
	EventCodeNone    EventCode = 1
	EventCodeUnknown EventCode = -1
)

// EventNameNone is name of default non counter logging event i.e. message only
const EventNameNone string = "System.unused"

// EventNameUnknown is name of default event which is not found in the map
const EventNameUnknown string = "System.unknown"

// EventCode is a unique code for each event
type EventCode int

// LogEvent is a structured event that is passed to the logger to be recorded
type LogEvent struct {
	LogLevel  LogLevel  // Level of logging Error - Warning - Debug - Info
	Name      string    // String name of the event
	Code      EventCode // Unique code for this event of this type
	Count     int       // Reporting multiple events at once
	Message   string    // Message associated with the event
	Payload   string    // Optional payload associated with a counter event
	RequestID uuid.UUID // Request ID if this event is associated with a request (nil otherwise)
	UserAgent string    // User-Agent header from the request
	// Identity of the sender
	TenantID uuid.UUID
}

// LogEventTypeInfo is contains information about a particular event type
type LogEventTypeInfo struct {
	Name           string
	NormalizedName string
	Code           EventCode
	Service        service.Service
	URL            string
	Ignore         bool // Don't send event to the server (only process locally)
	Category       EventCategory
	Subcategory    string
}

// EventMetadataMap is contains information about a particular event type
type EventMetadataMap struct {
	Version int
	Map     map[string]LogEventTypeInfo
}

// EventCategory identifies the category of the event
type EventCategory string

// Different event categories
const (
	EventCategoryUnknown       EventCategory = "Unknown"
	EventCategorySystem        EventCategory = "System"
	EventCategoryCall          EventCategory = "Call"
	EventCategoryDuration      EventCategory = "Duration"
	EventCategoryInputError    EventCategory = "InputError"
	EventCategoryInternalError EventCategory = "InternalError"
	EventCategoryResultSuccess EventCategory = "ResultSuccess"
	EventCategoryResultFailure EventCategory = "ResultFailure"
	EventCategoryCount         EventCategory = "Count"

	EventCategoryTODO EventCategory = "TODO" // these are auto-generated events that need classified
)

// EventMetadataFetcher knows how to get the event metadata map
type EventMetadataFetcher interface {
	Init(updateHandler func(updatedMap *EventMetadataMap, tenantID uuid.UUID) error) error
	FetchEventMetadataForTenant(tenantID uuid.UUID)
	Close()
}

// getLogEventTypesMap returns the map of event types for a tenant allowing for resolving event names to codes
func getLogEventTypesMap(tenantID uuid.UUID) map[string]LogEventTypeInfo {
	loggerInst.eventMetadataMutex.RLock()
	m, ok := loggerInst.eventMetadata[tenantID]
	loggerInst.eventMetadataMutex.RUnlock()
	if !ok {
		loggerInst.eventMetadataMutex.Lock()
		m, ok = loggerInst.eventMetadata[tenantID]
		if !ok {
			m = EventMetadataMap{Version: 0, Map: make(map[string]LogEventTypeInfo)}
			loggerInst.eventMetadata[tenantID] = m
			// Schedule retrieval of the map if the fetcher is provided
			if loggerInst.eventMetadataFetcher != nil {
				loggerInst.eventMetadataFetcher.FetchEventMetadataForTenant(tenantID)
			}
		}
		loggerInst.eventMetadataMutex.Unlock()
	}
	return m.Map
}

// GetEventInfo returns the event type information for a given event
func GetEventInfo(event LogEvent) LogEventTypeInfo {
	return getEventInfoByName(event.Name, event.Code, event.TenantID)
}

// getEventInfoByName maps event name to event code
func getEventInfoByName(eventName string, currentCode EventCode, tenantID uuid.UUID) LogEventTypeInfo {
	m := getLogEventTypesMap(tenantID)
	t, ok := m[eventName]

	if !ok {
		// HTTP Codes are logged directly
		if strings.HasPrefix(eventName, "Event.HTTPResponse") {
			return LogEventTypeInfo{Name: eventName, Code: currentCode}
		}
		// We either haven't seen this event before or the map hasn't been loaded yet
		// Send it as unknown to the server
		return LogEventTypeInfo{Name: eventName, Code: EventCodeUnknown}
	}
	return t
}

// Validate validates that filled out event is consistent
func (e LogEvent) Validate() error {
	// It makes no sense to log counter event with zero count
	if e.Code != EventCodeNone && e.Count == 0 {
		return ucerr.New("can't log counter event with zero count")
	}
	// All counter events have to have a name
	if e.Code != EventCodeNone && e.Name == "" {
		return ucerr.New("counter events must have a name")
	}
	// All logevents with a message must specify a log level
	if e.Message != "" && (e.LogLevel == LogLevelNone || e.LogLevel == LogLevelNonMessage) {
		return ucerr.New("can't log message without log level")
	}

	return nil
}

// LogRecordArray represents a set of log messages/events from a same service/tenant/region/host combination
// It is used for on the wire representation
type LogRecordArray struct {
	Service  service.Service      `json:"s" yaml:"s"`
	TenantID uuid.UUID            `json:"t" yaml:"t"`
	Region   region.MachineRegion `json:"r" yaml:"r"`
	Host     string               `json:"h" yaml:"h"`
	Records  []LogRecordContent   `json:"c" yaml:"c"`
}

// LogRecordContent represents unique information in log event/message for a fixed service/tenant/region/host combination
// It is used for on the wire representation
type LogRecordContent struct {
	Message   string    `json:"m" yaml:"m"`
	Code      EventCode `json:"c" yaml:"c"`
	Payload   string    `json:"p" yaml:"p"`
	Timestamp int       `json:"t" yaml:"t"`
}
