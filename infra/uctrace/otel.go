package uctrace

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	otelSdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
)

// Constants
const (
	// Subsystem name for trace metrics
	traceSubsystem ucmetrics.Subsystem = "trace"

	// Connection state check interval
	connectionCheckInterval = 5 * time.Second
)

// Metrics for trace provider connection failures
var (
	traceProviderConnectionFailure = ucmetrics.CreateGauge(
		traceSubsystem,
		"provider_connection_failure",
		"Indicates if there is a failure connecting to the trace collector",
		"collector_host",
	)
)

// idGenerator uses the trace ID set in the context by request.Middleware for
// the OpenTelemetry trace IDs.
type idGenerator struct {
	sync.Mutex
	randSource *rand.Rand
}

var _ otelSdkTrace.IDGenerator = &idGenerator{}

// NewIDs sets the trace ID to the request ID from context if available,
// otherwise generates a random ID.
func (gen *idGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	gen.Lock()
	defer gen.Unlock()
	tid := trace.TraceID{}
	// Use request ID from context if available
	if ctxTid := request.GetRequestID(ctx); ctxTid != uuid.Nil {
		tid = ([16]byte)(ctxTid.Bytes())
	} else {
		_, _ = gen.randSource.Read(tid[:])
	}
	sid := trace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return tid, sid
}

// NewSpanID returns a non-zero span ID from a randomly-chosen sequence.
func (gen *idGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	gen.Lock()
	defer gen.Unlock()
	sid := trace.SpanID{}
	_, _ = gen.randSource.Read(sid[:])
	return sid
}

func newIDGenerator() otelSdkTrace.IDGenerator {
	gen := &idGenerator{}
	var rngSeed int64
	if err := binary.Read(crand.Reader, binary.LittleEndian, &rngSeed); err != nil {
		// Really need to be able to read from random... Thankfully this only
		// gets called at startup, so the service will fail to start before we
		// put it into the load balancer rotation
		panic(ucerr.Wrap(err))
	}
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.HostName(hostname),
			semconv.CloudRegion(string(region.Current())),
		))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return res, nil
}

func getOtlpConn(_ context.Context, config *Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(config.CollectorHost,
		// Note: disabling TLS requirement because we send trace locally:
		// Grafana Alloy running in cluster (in EKS)
		// Local grafana instance in dev.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, ucerr.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Start connection
	conn.Connect()

	// We'll rely on monitorConnState to handle connection state monitoring
	// and metric updates instead of checking the initial state here

	return conn, nil
}

// monitorConnState starts a goroutine that periodically checks the connection state
// and updates the metric. It returns a function that can be called to stop the monitoring.
func monitorConnState(ctx context.Context, conn *grpc.ClientConn, collectorHost string) func() {
	// Create a cancellable context for the goroutine
	monitorCtx, cancel := context.WithCancel(ctx)

	// Start a goroutine to periodically check the connection state
	go func() {
		ticker := time.NewTicker(connectionCheckInterval)
		defer ticker.Stop()

		// Track previous state to detect transitions
		prevState := conn.GetState()

		for {
			select {
			case <-monitorCtx.Done():
				return
			case <-ticker.C:
				state := conn.GetState()

				// Update metrics based on current state
				if state == connectivity.Ready {
					traceProviderConnectionFailure.WithLabelValues(collectorHost).Set(0)
				} else if state == connectivity.TransientFailure {
					traceProviderConnectionFailure.WithLabelValues(collectorHost).Set(1)

					// Only log when transitioning from Ready to TransientFailure
					if prevState == connectivity.Ready {
						uclog.Warningf(ctx, "failed to connect to collector: %s", state.String())
					}
				}

				// Update previous state for next check
				prevState = state
			}
		}
	}()

	// Return a function that can be called to stop the monitoring
	return cancel
}

func newTraceProvider(res *resource.Resource, config *Config) (*otelSdkTrace.TracerProvider, func(), error) {
	ctx := context.Background()
	conn, err := getOtlpConn(ctx, config)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// Start monitoring the connection state
	cancelMonitor := monitorConnState(ctx, conn, config.CollectorHost)

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		cancelMonitor() // Clean up the monitor goroutine
		return nil, nil, ucerr.Errorf("failed to create trace exporter: %w", err)
	}

	traceProvider := otelSdkTrace.NewTracerProvider(
		otelSdkTrace.WithBatcher(traceExporter,
			otelSdkTrace.WithBatchTimeout(time.Second*5)),
		otelSdkTrace.WithResource(res),
		otelSdkTrace.WithIDGenerator(newIDGenerator()),
	)

	return traceProvider, cancelMonitor, nil
}

// Init bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func Init(ctx context.Context, service service.Service, serviceVersion string, config *Config) (shutdown func(context.Context), err error) {
	if config == nil {
		return func(context.Context) {}, ucerr.New("tracing is not enabled")
	}

	var shutdownFuncs []func(context.Context) error
	var cancelMonitor func()

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) {
		// Cancel the connection monitor if it exists
		if cancelMonitor != nil {
			cancelMonitor()
		}

		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		if err != nil {
			fmt.Printf("Unexpected error shutting down tracing services! %v\n", ucerr.Wrap(err))
		}
	}

	// Setup resource.
	res, err := newResource((string)(service), serviceVersion)
	if err != nil {
		shutdown(ctx)
		return shutdown, ucerr.Wrap(err)
	}

	// Setup trace provider.
	tracerProvider, monitorCancel, err := newTraceProvider(res, config)
	if err != nil {
		shutdown(ctx)
		return shutdown, ucerr.Wrap(err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Store the monitor cancel function for cleanup
	cancelMonitor = monitorCancel

	// The connection monitoring is already started in newTraceProvider

	return shutdown, nil
}

// InitErrorMessageForDev is an error message to show if Init() fails on dev
const InitErrorMessageForDev = "Failed to initialize tracing. You can probably ignore this error on dev. If you want to be able to collect/view traces locally, run \"make grafana-start\"."
