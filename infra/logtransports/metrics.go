package logtransports

import "userclouds.com/infra/ucmetrics"

const logTransportSubsystem = ucmetrics.Subsystem("log")

var (
	failedCalls     = ucmetrics.CreateCounter(logTransportSubsystem, "failed_calls", "Number of failed API calls", "transport_type", "error_type")
	successfulCalls = ucmetrics.CreateCounter(logTransportSubsystem, "successful_calls", "Number of successful API calls", "transport_type")
	droppedCalls    = ucmetrics.CreateCounter(logTransportSubsystem, "dropped_calls", "Number of dropped backgroundTransport calls", "transport_type")
	queueSize       = ucmetrics.CreateGauge(logTransportSubsystem, "queue_size", "Size of the backgroundTransport queue", "transport_type")
	sentEventCount  = ucmetrics.CreateCounter(logTransportSubsystem, "sent_event_count", "Number of sent events", "transport_type")
)
