package events

import (
	"context"

	"userclouds.com/infra/uclog"
)

// Event codes
// 0 .. 99 reserved for logging system
// 100 .. 600 HTTP codes
// 600 .. 1000 reserved for logging system
// > 1000 service events
const (
	EventHTTPRequest            uclog.EventCode = 99
	EventNone                   uclog.EventCode = 1
	EventSecIPBlocked           uclog.EventCode = 1602
	EventSecIPFail              uclog.EventCode = 1600
	EventSecIntFailure          uclog.EventCode = 1604
	EventSecUserBlocked         uclog.EventCode = 1603
	EventSecUserFail            uclog.EventCode = 1601
	EventSecIPBlockedHostHeader uclog.EventCode = 1605
	EventUnknown                uclog.EventCode = -1

	CustomEventShift int = 16
)

// GetLogEventTypes maps string to data structures TODO temporary until the strings move to event description table
func GetLogEventTypes() map[string]uclog.LogEventTypeInfo {
	return eventMap
}

// RegisterEventType allows another package to register its own events
func RegisterEventType(eventType string, eventInfo uclog.LogEventTypeInfo) {
	if _, ok := eventMap[eventType]; ok {
		uclog.Warningf(context.Background(), "Event type %s already registered", eventType)
	}
	eventMap[eventType] = eventInfo
}

var eventMap = map[string]uclog.LogEventTypeInfo{
	"Event.HTTPRequest": {Name: "HTTP Request", Code: EventHTTPRequest, Service: "", Category: uclog.EventCategorySystem},

	"System.unused":                  {Name: "unused", Code: EventNone, Service: "", Category: uclog.EventCategorySystem},
	"System.unknown":                 {Name: "Unknown event", Code: EventUnknown, Service: "", Category: uclog.EventCategorySystem},
	"Security.IPFail":                {Name: "Call Failure by IP", Code: EventSecIPFail, Service: "", Category: uclog.EventCategorySystem},
	"Security.UserFail":              {Name: "Call Failure by User", Code: EventSecUserFail, Service: "", Category: uclog.EventCategorySystem},
	"Security.IPBlocked":             {Name: "Call Blocked by IP", Code: EventSecIPBlocked, Service: "", Category: uclog.EventCategorySystem},
	"Security.UserBlocked":           {Name: "Call Blocked by User", Code: EventSecUserBlocked, Service: "", Category: uclog.EventCategorySystem},
	"Security.InternalError":         {Name: "Security Code Failure", Code: EventSecIntFailure, Service: "", Category: uclog.EventCategorySystem},
	"Security.IPBlockedInHostHeader": {Name: "Call Blocked by IP in Host Header", Code: EventSecIPBlockedHostHeader, Service: "", Category: uclog.EventCategorySystem},
}

const mapVersion = 1

// GetEventMetaDataMap returns a event map and associated metadata
func GetEventMetaDataMap(m *map[string]uclog.LogEventTypeInfo) uclog.EventMetadataMap {
	return uclog.EventMetadataMap{Version: mapVersion, Map: *m}
}

// IsStartUpEvent returns true if passed in eventCode corresponds to a shifted service statup event
// TODO: why is this not structured? fix me.
func IsStartUpEvent(eventCode uclog.EventCode) bool {
	if eventCode == EventPlexStartup ||
		eventCode == EventIDPStartup ||
		eventCode == EventConsoleStartup ||
		eventCode == EventAuthzStartup ||
		eventCode == EventTokenizerStartup {
		return true
	}
	return false
}

// IsDuration returns true if passed in eventCode corresponds to a service statup event
func IsDuration(eventCode uclog.EventCode, em *map[uclog.EventCode]uclog.LogEventTypeInfo) bool {
	return checkEventType(eventCode, em, uclog.EventCategoryDuration)
}

// IsInputError returns true if passed in eventCode corresponds to an input error
func IsInputError(eventCode uclog.EventCode, em *map[uclog.EventCode]uclog.LogEventTypeInfo) bool {
	return checkEventType(eventCode, em, uclog.EventCategoryInputError)
}

// IsInternalError returns true if passed in eventCode corresponds to an internal error
func IsInternalError(eventCode uclog.EventCode, em *map[uclog.EventCode]uclog.LogEventTypeInfo) bool {
	return checkEventType(eventCode, em, uclog.EventCategoryInternalError)
}

func checkEventType(eventCode uclog.EventCode, em *map[uclog.EventCode]uclog.LogEventTypeInfo, eventType uclog.EventCategory) bool {
	info, ok := (*em)[eventCode]

	if ok {
		return info.Category == eventType // TODO define the set of types
	}

	return false
}
