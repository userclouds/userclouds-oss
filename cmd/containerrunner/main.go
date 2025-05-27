package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"sigs.k8s.io/yaml"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/multirun"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/localrouting"
	"userclouds.com/internal/routinghelper"
	"userclouds.com/internal/servicecolors"
)

const (
	routingConfigFile = "userclouds/routing.yaml"
	port              = 3040
	// see: docker/userclouds-headless/docker-compose.yaml (we mount /userclouds/logs to the host)
	ucconfigLogFileLocation = "/userclouds/logs/ucconfig.log"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelNonMessage, "containerrunner")
	defer logtransports.Close()

	headless := flag.Bool("headless", false, "headless mode (plex, authz and userstore/idp only)")
	routingFile := flag.String("routing", routingConfigFile, "routing config file")

	flag.Parse()
	var manifestFilePath string
	if flag.NArg() > 0 {
		manifestFilePath = flag.Arg(0)
	} else {
		manifestFilePath = ""
	}
	svcs := make([]service.Service, 0)
	if *headless {
		svcs = append(svcs, service.HeadlessConsoleServices...)
	} else {
		svcs = append(svcs, service.AllWebServices...)
		svcs = append(svcs, service.Worker)
	}
	routesCfg := loadRouteCfg(ctx, *routingFile)
	mux, err := createRouterHTTPMux(ctx, routesCfg, svcs)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create router: %v", err)
	}
	cmds := getCommands(svcs)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		runProcesses(ctx, cmds)
		wg.Done()
	}()
	go func() {
		runRouter(ctx, fmt.Sprintf(":%d", port), mux)
		wg.Done()
	}()

	runUserCloudConfig(ctx, manifestFilePath, routesCfg.GetPorts())
	wg.Wait()
}

func createRouterHTTPMux(ctx context.Context, routesCfg routinghelper.RouteConfig, usedServices []service.Service) (*http.ServeMux, error) {
	proxies, err := localrouting.NewProxiesMap(ctx, routesCfg, usedServices)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	// NOTE: we intentionally do not use uchttp.NewServeMux here, since the uchttp-wrapped
	// handler code strips the initial prefix from the route
	mux := http.NewServeMux()
	if err := proxies.AddRulesToServer(ctx, "container", routesCfg.Rules, mux); err != nil {
		return nil, ucerr.Wrap(err)
	}
	http.Handle("/", mux)
	return mux, nil
}

func loadRouteCfg(ctx context.Context, cfgFile string) routinghelper.RouteConfig {
	cfgBytes, err := os.ReadFile(cfgFile)
	if err != nil {
		uclog.Fatalf(ctx, "failed to open routing config file: %v: %v", cfgFile, err)
	}
	var routeCfg routinghelper.RouteConfig
	if err := yaml.Unmarshal(cfgBytes, &routeCfg); err != nil {
		uclog.Fatalf(ctx, "failed to decode routing config file: %v: %v", cfgFile, err)
	}
	uclog.Infof(ctx, "Loaded %v rules from: %v", len(routeCfg.Rules), cfgFile)
	return routeCfg
}

func runRouter(ctx context.Context, address string, mux *http.ServeMux) {
	uclog.Debugf(ctx, "Router listening on %s", address)
	uclog.Fatalf(ctx, "%v", http.ListenAndServe(address, mux))
}

func runProcesses(ctx context.Context, cmds []multirun.Command) {
	for i, cmd := range cmds {
		if cmds[i].Color == "" {
			cmds[i].Color = servicecolors.MustGetColor(ctx, cmd.GetName())
		}
	}

	if err := multirun.SetupCommands(ctx, cmds); err != nil {
		uclog.Fatalf(ctx, "failed to setup commands: %v", err)
	}
	env := multirun.NewEnv(ctx, cmds)
	for _, c := range cmds {
		multirun.WrapOutputs(c, env)
	}
	if err := multirun.Run(ctx, cmds, env); err != nil {
		uclog.Fatalf(ctx, "failed to run: %v", err)
	}
}

func runUserCloudConfig(ctx context.Context, manifestFilePath string, ports []int) {
	if manifestFilePath == "" {
		return
	}
	if fi, err := os.Stat(manifestFilePath); err != nil || fi.IsDir() {
		uclog.Infof(ctx, "No manifest file found at %v, skipping config load (%v)", manifestFilePath, err)
		return
	}

	if !waitForServicesReady(ctx, time.Second*10, ports) {
		uclog.Fatalf(ctx, "Failed to wait for services to be ready")
	}

	// Using multirun to capture output.
	if err := multirun.RunSingleCommand(ctx, "ucconfig", "--logfile", ucconfigLogFileLocation, "apply", "--auto-approve", manifestFilePath); err != nil {
		uclog.Errorf(ctx, "ucconfig failed: %v", err)
	}
}

func waitForServicesReady(ctx context.Context, timeout time.Duration, ports []int) bool {
	time.Sleep(3 * time.Second) // Give services a chance to start
	wg := sync.WaitGroup{}
	servicesReady := 0
	wg.Add(len(ports))
	for _, svcPort := range ports {
		port := svcPort
		go func() {
			if waitForServiceReadiness(ctx, port, timeout) {
				servicesReady++
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return servicesReady == len(ports)
}

func waitForServiceReadiness(ctx context.Context, port int, timeout time.Duration) bool {
	url := fmt.Sprintf("http://localhost:%d", port)
	uclog.Infof(ctx, "Waiting for service on port %v to be ready", port)
	client := jsonclient.New(url)
	startTime := time.Now().UTC()
	for time.Now().UTC().Sub(startTime) < timeout {
		if err := client.Get(ctx, "/healthcheck", nil); err != nil {
			uclog.Debugf(ctx, "Waiting for service on port %v to be ready: %v", port, err)
			time.Sleep(1 * time.Second)
			continue
		}
		return true
	}
	return false
}

func getCommands(svcs []service.Service) []multirun.Command {
	cmds := make([]multirun.Command, 0, len(svcs))
	for _, svc := range svcs {
		cmds = append(cmds, multirun.Command{Bin: string(svc)})
	}
	return cmds
}
