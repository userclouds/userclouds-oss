package skeleton

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"
	"sigs.k8s.io/yaml"

	"userclouds.com/infra"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/region"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/infra/ucmetrics/endpoint"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/ucsentry"
)

const (
	// we will use this metric to make sure all services are running the same version (and alert if they don't)
	serviceSubsystem      = ucmetrics.Subsystem("service")
	maxRedisWaitTime      = time.Second * 10
	serverShutdownTimeout = 10 * time.Second
)

var versionMetric = ucmetrics.CreateGauge(serviceSubsystem, "version", "The version of the service", "version", "buildTime")

// In the base case we just want to return migrations to move forward, so don't validate code migrations
type noopValidator struct{}

// Server is a struct that holds the closers and the service
type Server struct {
	closers             []func(context.Context)
	service             serviceNamespace.Service
	serviceInstanceName string
	cacheConfig         *cache.Config
	svcConfigYAML       []byte
	startupTime         time.Time
	hostname            string
	region              region.MachineRegion
}

// RunServerArgs is a helper struct to run a server
type RunServerArgs struct {
	HandleBuilder            *builder.HandlerBuilder
	MountPoint               service.Endpoint
	InternalServerMountPoint *service.Endpoint
}

// InitServerArgs is a helper struct to init a server
type InitServerArgs struct {
	Service             serviceNamespace.Service
	FeatureFlagConfig   *featureflags.Config
	SentryConfig        *ucsentry.Config
	TracingConfig       *uctrace.Config
	CacheConfig         *cache.Config
	ConsoleTenantID     uuid.UUID
	ServiceConfig       infra.Validateable
	ServiceInstanceName string
}

// Validate implements ucdb.Validator
func (n noopValidator) Validate(_ context.Context, db *ucdb.DB) error {
	return nil
}

// Router sets up the very very base service that provides the minimal
// set of services required for secure recovery. Right now this is just migrating,
// but in the future it might be eg. provisioning a base authz role or something else.
// Named for a "skeleton crew" rather than a "skeleton key" :)
// Note this isn't currently used in idp because we have to validate the companyconfig db to
// even know how to load per-tenant DB configs
func Router(ctx context.Context, dbCfg *ucdb.Config) (*builder.HandlerBuilder, error) {
	db, err := ucdb.New(ctx, dbCfg, &noopValidator{})
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	hb := builder.NewHandlerBuilder()
	hb.HandleFunc("/migrations", service.CreateMigrationVersionHandler(db))

	return hb, nil
}

// logStartup unifies startup logging (on the way to a better service base system in general) so
// we always get host, region, events, etc
func (s Server) logStartup(ctx context.Context, svcEP service.Endpoint, metricsEP *service.Endpoint) {
	ipAddressPart := ""
	if kubernetes.IsKubernetes() {
		ipAddressPart = fmt.Sprintf(" (IP: %s)", kubernetes.GetPodIP())
	}

	sEvent := uclog.ServiceStartupInfo{Region: region.Current(), Hostname: service.GetMachineName(), CodeVersion: service.GetBuildHash()}
	var metrics string
	if metricsEP != nil {
		metrics = metricsEP.BaseURL()
	} else {
		metrics = "disabled"
	}
	var serviceName string
	if s.serviceInstanceName == "" {
		serviceName = string(s.service)
	} else {
		serviceName = fmt.Sprintf("%v[%s]", s.service, s.serviceInstanceName)
	}
	uclog.Infof(ctx, "%s listening on %s (internal server %s) machine %s%s in region %s with build %v", serviceName, svcEP.BaseURL(), metrics, sEvent.Hostname, ipAddressPart, sEvent.Region, sEvent.CodeVersion)
	payloadVal, _ := json.Marshal(sEvent)
	uclog.IncrementEventWithPayload(ctx, fmt.Sprintf("%s.Startup", s.service), string(payloadVal))
}

// InitServer initializes the server with the given args
func InitServer(ctx context.Context, args InitServerArgs) (Server, error) {
	registerJSONClientLogger()
	buildHash := service.GetBuildHash()
	ucsentry.Init(ctx, args.SentryConfig, buildHash, service.GetMachineName())
	featureflags.Init(ctx, args.FeatureFlagConfig)
	yamlCfg, err := getYamlDoc(args.ServiceConfig)
	if err != nil {
		return Server{}, ucerr.Wrap(err)
	}
	closersFuncs := make([]func(context.Context), 0, 3)
	closersFuncs = append(closersFuncs, ucsentry.Close)
	closersFuncs = append(closersFuncs, featureflags.Close)
	if args.TracingConfig == nil {
		uclog.Infof(ctx, "Tracing is disabled")
	} else {
		uclog.Infof(ctx, "Initializing tracing with config: %+v", args.TracingConfig)
		closeTracing, err := uctrace.Init(ctx, args.Service, buildHash, args.TracingConfig)
		uv := universe.Current()
		if err != nil {
			if uv.IsDev() {
				uclog.Warningf(ctx, "Failed to initialize tracing. config: %+v error: %v", args.TracingConfig, err)
				uclog.Warningf(ctx, uctrace.InitErrorMessageForDev)
			} else {
				uclog.Errorf(ctx, "Failed to initialize tracing. config: %+v error: %v", args.TracingConfig, err)
				sentry.CaptureException(ucerr.Errorf("Failed to initialize tracing. config: %+v error: %w", args.TracingConfig, err))
			}
		} else {
			closersFuncs = append(closersFuncs, closeTracing)
		}
	}
	uv := universe.Current()
	if uv.IsCloud() {
		if err := cache.InitRedisCertForCloud(); err != nil {
			return Server{}, ucerr.Wrap(err)
		}
	} else if uv.IsOnPrem() {
		// In on prem, redis runs locally in the cluster, but may not be available yet when the UC service starts,so we wait for it to be available.
		// if it is not available, we will fail the service start (crashes the pod/container) which will cause k8s to retry.
		if err := waitForRedis(ctx, args.CacheConfig); err != nil {
			return Server{}, ucerr.Wrap(err)
		}
	}
	srv := Server{
		service:             args.Service,
		serviceInstanceName: args.ServiceInstanceName,
		cacheConfig:         args.CacheConfig,
		svcConfigYAML:       yamlCfg,
		startupTime:         time.Now().UTC(),
		hostname:            service.GetMachineName(),
		region:              region.Current(),
		closers:             closersFuncs,
	}
	return srv, nil
}

func getYamlDoc(cfg infra.Validateable) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return yamlBytes, nil
}

func waitForRedis(ctx context.Context, cacheConfig *cache.Config) error {
	timeoutTime := time.Now().UTC().Add(maxRedisWaitTime)
	for {
		_, err := cache.GetRedisClient(ctx, cacheConfig.GetLocalRedis())
		if err == nil {
			break
		} else if time.Now().UTC().After(timeoutTime) {
			return ucerr.Errorf("failed to connect to redis: %v", err)
		}
		uclog.Warningf(ctx, "failed to connect to redis: %v, retrying", err)
		time.Sleep(time.Millisecond * 300)
	}
	return nil
}

func (s Server) close(ctx context.Context) {
	for _, closer := range s.closers {
		closer(ctx)
	}
}

// Run is a helper to run a server with a root handler and a service name
func (s Server) Run(ctx context.Context, args RunServerArgs) {
	defer s.close(ctx)
	s.logStartup(ctx, args.MountPoint, args.InternalServerMountPoint)
	ctxStop, stop := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer stop()
	if args.InternalServerMountPoint != nil {
		go s.startInternalServer(ctxStop, *args.InternalServerMountPoint)
	}
	versionMetric.WithLabelValues(service.GetBuildHash(), service.GetBuildTime()).Set(1)
	bldr := args.HandleBuilder.Handle("/healthcheck", service.BaseMiddleware.Apply(http.HandlerFunc(s.healthCheck)))
	service.AddGetDeployedEndpoint(bldr)
	rh := bldr.Build()
	http.Handle("/", rh)
	server := &http.Server{Addr: args.MountPoint.HostAndPort(), Handler: rh}
	runServer(ctxStop, "main", server)
}

// The internal server is used to serve metrics and other internal endpoints we don't want to expose via ALBs to the internet
// TODO: Move resource check, migrations, sentry test , deploy, etc... endpoints here
// this makes more sense when we eventually run services in k8s since tooling will be able to hit those endpoints easily so we don't have to expose them to the internet like we do today
func (s Server) startInternalServer(ctx context.Context, internalServerEP service.Endpoint) {
	uclog.Infof(ctx, "Starting internal server on %s", internalServerEP.BaseURL())
	mux := uchttp.NewServeMux()
	endpoint.AddMetricsEndpoint(mux)
	mux.HandleFunc("/config", s.configYAML)
	internalServer := &http.Server{Addr: internalServerEP.HostAndPort(), Handler: mux}
	runServer(ctx, "internal", internalServer)
}

func (s Server) configYAML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headers.ContentType, "text/x-yaml")
	if _, err := w.Write(s.svcConfigYAML); err != nil {
		uclog.Errorf(r.Context(), "failed to write config yaml: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hcs := HealthCheckStatus{
		LogStatus:   uclog.GetStatus(),
		Hostname:    s.hostname,
		Region:      s.region,
		StartupTime: s.startupTime,
		Cache:       cache.GetRedisStatus(ctx, s.cacheConfig),
		Service:     s.service,
	}
	jsonapi.Marshal(w, hcs, hcs.getHTTPCodeOption())
}

func runServer(ctx context.Context, serverName string, server *http.Server) {
	go func() {
		uclog.Infof(ctx, "[%s server] Starting", serverName)
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				uclog.Infof(ctx, "[%s server] shut down gracefully", serverName)
			} else {
				uclog.Errorf(ctx, "[%s server] crashed: %v %T", serverName, err, err)
			}
		}
	}()
	<-ctx.Done()
	start := time.Now().UTC()
	uclog.Infof(ctx, "[%s server] Shutting down", serverName)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		uclog.Errorf(ctx, "[%s server] could not shutdown within %v: %v", serverName, serverShutdownTimeout, err)
	}
	uclog.Infof(ctx, "[%s server] shut down in %v", serverName, time.Now().UTC().Sub(start))
}

// HealthCheckStatus is the status of the health check
type HealthCheckStatus struct {
	LogStatus   uclog.LocalStatus        `json:"logs"`
	Cache       cache.RedisStatus        `json:"cache"`
	Hostname    string                   `json:"hostname"` // for understanding the response
	Region      region.MachineRegion     `json:"region"`
	StartupTime time.Time                `json:"startup_time"` // time the service started
	Service     serviceNamespace.Service `json:"service"`      // the currently running service
}

func (hcs HealthCheckStatus) getHTTPCodeOption() jsonapi.Option {
	if !hcs.Cache.Ok {
		return jsonapi.Code(http.StatusServiceUnavailable)
	}
	return jsonapi.Code(http.StatusOK)
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
