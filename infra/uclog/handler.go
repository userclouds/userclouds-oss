package uclog

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
)

// This file contains functionality for validating that all handlers are correctly instrumented and that is reflected in the event map
// downloaded from the server. Add validation for new per handler documentation in validateHandler

// GetServiceName returns the name for the overall service (used for calculating the handler name)
func GetServiceName() string {
	loggerInst.loggerConfigMutex.RLock()
	defer loggerInst.loggerConfigMutex.RUnlock()

	return loggerInst.serviceName
}

// AddHandlerForValidation this function helps with debugging missing/new handlers that need to be instrumented
func AddHandlerForValidation(ctx context.Context, handlerName string) {
	loggerInst.loggerConfigMutex.Lock()
	loggerInst.registeredHandlers = append(loggerInst.registeredHandlers, handlerName)
	loggerInst.loggerConfigMutex.Unlock()

	// Check if the map is already downloaded in which case validate this one handler since
	// ValidateMap has almost always already ran. There is a small race if this code runs in the window
	// between map being downloaded and ValidateMap being called which would lead to this handlerName
	// being validated twice. This can only happen for a single handler because of the writer lock above

	loggerInst.loggerConfigMutex.RLock()
	defer loggerInst.loggerConfigMutex.RUnlock()

	m := getLogEventTypesMap(uuid.Nil)
	if len(m) > 0 {
		validateHandler(ctx, handlerName, m)
	}
}

// Validate all the currently registered handlers. This function is called after the event map is downloaded or updated
// If additional handlers are added after this call, they will be validated one by one as they are added
func validateHandlerMap(ctx context.Context) {
	m := getLogEventTypesMap(uuid.Nil)
	for _, handlerName := range loggerInst.registeredHandlers {
		validateHandler(ctx, handlerName, m)
	}
}

var excludedHandlers = []string{
	// Skip initialization handlers created by code gen that not called during operation
	".initServiceHandlers.func",
	".initOpenAPIHandler.func",
	".AddResourceCheckEndpoint.",
	".getReadinessHandler.",
	//Internal server endpoint(s)
	".configYAML",
	// Worker endpoints
	"syncall",
	"checkcnames",
}

func validateHandler(ctx context.Context, handlerName string, m map[string]LogEventTypeInfo) {
	for _, excludedHandler := range excludedHandlers {
		if strings.Contains(handlerName, excludedHandler) {
			return
		}
	}
	e, ok := m[handlerName+".Count"]
	if !ok {
		Warningf(ctx, "Missing Count event for handler %s. Please run provisioning.", handlerName)
	} else if e.Category != "Call" {
		Warningf(ctx, "Unexpected event type instead of Call for handler %s", handlerName)
	}
	e, ok = m[handlerName+".Duration"]
	if !ok {
		Warningf(ctx, "Missing Duration event for handler %s. Please run provisioning.", handlerName)
	} else if e.Category != "Duration" {
		Warningf(ctx, "Unexpected event type instead of Call for handler %s", handlerName)
	}
}
