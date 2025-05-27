package logtransports

import (
	"context"
	"runtime/debug"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
)

const serverURL = "https://logserver.userclouds.com"

// InitLoggerAndTransportsForSDK sets up logging transports for SDK
func InitLoggerAndTransportsForSDK(config *Config, tokenSource jsonclient.Option, name service.Service, machineName string) {
	registerJSONClientLogger()
	transports := initConfigInfoInTransports(name, machineName, config, tokenSource)
	uclog.InitForService(name, transports, nil)
}

// InitLoggingSDK sets up logging transport for SDK
func InitLoggingSDK(auth *ucjwt.Config, rawLogs bool, machineName string) {
	registerJSONClientLogger()
	tokenSource, err := jsonclient.ClientCredentialsForURL(auth.TenantURL, auth.ClientID, auth.ClientSecret, nil)
	if err != nil {
		uclog.Fatalf(context.Background(), "Failed to get token source: %v", err)
	}

	var transports []uclog.Transport = make([]uclog.Transport, 0, 1)

	lstc := &LogServerTransportConfig{
		TransportConfig: uclog.TransportConfig{
			Required:    false,
			MaxLogLevel: 5,
		},
		TenantID:      auth.TenantID,
		LogServiceURL: serverURL,
		SendRawData:   rawLogs,
	}

	transports = append(transports,
		newTransportBackgroundIOWrapper(
			newLogServerTransport(lstc, tokenSource, service.SDK, machineName)))

	uclog.InitForService(service.SDK, transports, nil)
}

var recoverPanic = false

// Close performs any additional clean up and then call uclog.Close()
func Close() {
	// we only want this behavior in our services, not the SDK
	if recoverPanic {
		// This assumes that uclog.Close is the first function we defer in main so it is safe to call recover() and force exit
		// This works panics on the main thread, but doesn't work if a panic occurs in a go routine. In that case,
		// deferred function on the main thread do not get run. So go routines need to be wrapped in async.Execute(..) if
		// we want to capture the panics there
		if r := recover(); r != nil {
			// This will not return
			uclog.Fatalf(context.Background(), "Panic: %v Stack %s", r, string(debug.Stack()))
		}
	}
	uclog.Close()
}

type jsonclientLogger struct{}

// Debugf implements jsonclient.Logger
func (l *jsonclientLogger) Debugf(ctx context.Context, format string, args ...any) {
	uclog.Debugf(ctx, format, args...)
}

// Warningf implements jsonclient.Logger
func (l *jsonclientLogger) Warningf(ctx context.Context, format string, args ...any) {
	uclog.Warningf(ctx, format, args...)
}

// Registers uclog as the jsonclient logger when we use it in our service infra,
func registerJSONClientLogger() {
	jsonclient.RegisterLogger(&jsonclientLogger{})
}
