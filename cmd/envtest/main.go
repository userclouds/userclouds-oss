package main

import (
	"context"
	"sync"

	"github.com/alecthomas/kong"

	"userclouds.com/infra/async"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	defaultThreadCount   int = 20
	defaultThreadOpCount int = 1
)

type envTestCLI struct {
	Threads      int      `help:"number of worker threads to use" default:"20"`
	Iterations   int      `help:"number times to run tests" default:"1"`
	Verbose      bool     `help:"enable verbose output"`
	Parallel     bool     `short:"p" help:"run environment test for different services in parallel (good for creating mixed load on env)"`
	FailFast     bool     `help:"Crash/exit on first error" default:"false"`
	NoLocalRedis bool     `help:"Disable using redis in authz client" default:"false"`
	UseRegion    bool     `help:"Target a specific region (specify using UC_REGION env var)"`
	UseEKS       bool     `help:"Target a specific EKS in region (implies --use-region flag, UC_REGION must be set)"`
	TenantsFile  string   `help:"Yaml File containing tenants records to test against" default:""`
	Logfile      string   `help:"logfile name for debug output" default:""`
	Tenant       string   `help:"Tenants id or name against which the load test will be running, not required if --tenants-file is provided"`
	Services     []string `arg:"" help:"services to test" enum:"all,authz,tokenizer,userstore" required:""`
}

func (cli envTestCLI) Validate() error {
	if cli.Tenant != "" && cli.TenantsFile != "" {
		return ucerr.New("only one of --tenant or --tenants-file can be provided")
	}
	if cli.Tenant == "" && cli.TenantsFile == "" {
		return ucerr.New("one of --tenant or --tenants-file must be provided")
	}
	if cli.Iterations < 1 {
		return ucerr.New("iterations must be greater than 0")
	}
	if cli.useRegionalURLs() && !region.IsValid(region.Current(), universe.Current()) {
		uv := universe.Current()
		return ucerr.Errorf("invalid region '%v'  for '%v'. Must be one of %v", region.Current(), uv, region.MachineRegionsForUniverse(uv))
	}
	return nil
}
func (cli envTestCLI) useRegionalURLs() bool {
	return cli.UseRegion || cli.UseEKS
}

var envTestCfg envTestCLI

var errs = []error{}
var errMutex = sync.Mutex{}

func main() {
	ctx := context.Background()
	_ = kong.Parse(&envTestCfg)

	screenLogLevel := uclog.LogLevelInfo
	if envTestCfg.Verbose {
		screenLogLevel = uclog.LogLevelDebug
	}
	useLocalRedis := !envTestCfg.NoLocalRedis
	options := make([]logtransports.ToolLogOption, 0, 1)
	if envTestCfg.Logfile != "" {
		options = append(options, logtransports.Filename(envTestCfg.Logfile))
	}

	logtransports.InitLoggerAndTransportsForTools(ctx, screenLogLevel, uclog.LogLevelVerbose, "envtest", options...)
	defer logtransports.Close()

	tenantInfos, err := getTenantInfos(ctx, envTestCfg.TenantsFile, envTestCfg.Tenant, envTestCfg.useRegionalURLs(), envTestCfg.UseEKS)
	if err != nil {
		uclog.Fatalf(ctx, "Error loading tenants: %v", err)
	}
	tenantURLs := collectTenantURLs(tenantInfos)
	svcs := envTestCfg.Services
	if len(svcs) == 1 && svcs[0] == "all" {
		svcs = []string{"authz", "tokenizer", "userstore"}
	}
	uclog.Infof(ctx, "Running environment test on [%s] tenant(s) for services %v", tenantURLs, svcs)

	uv := universe.Current()
	if !uv.IsOnPremOrContainer() {
		healthChecks(ctx, uv, uv.IsCloud())
	}

	wg := sync.WaitGroup{}
	for _, t := range tenantInfos {
		lt := t
		wg.Add(1)
		async.Execute(func() {
			ig := sync.WaitGroup{}
			for _, svc := range svcs {
				ig.Add(1)
				if !envTestCfg.Parallel {
					if err := runTest(ctx, svc, lt.TenantURL, lt.TokenSource, &ig, useLocalRedis, envTestCfg.Iterations); err != nil {
						logErrorf(ctx, err, "Error running test for %v", svc)
					}
				} else {
					ls := svc
					async.Execute(func() {
						if err := runTest(ctx, ls, lt.TenantURL, lt.TokenSource, &ig, useLocalRedis, envTestCfg.Iterations); err != nil {
							logErrorf(ctx, err, "Error running test for %v", ls)
						}
					})
				}
			}
			ig.Wait()
			wg.Done()
		})
	}
	wg.Wait()

	// If we are not running in a loop we depend on each test to check value of "continuous" because the tests take very different
	// amounts of time

	// Check if any errors were logged
	if len(errs) == 0 {
		uclog.Infof(ctx, "Environment tests passed")
	} else {
		uclog.Fatalf(ctx, "Environment tests failed - %v", errs)
	}

}

func runTest(ctx context.Context, testName, tenantURL string, tokenSource jsonclient.Option, ig *sync.WaitGroup, useLocalRedis bool, iterations int) error {
	defer ig.Done()
	switch testName {
	case "tokenizer":
		tokenizerTest(ctx, tenantURL, tokenSource, iterations)
	case "authz":
		authzTest(ctx, tenantURL, tokenSource, useLocalRedis, iterations)
	case "userstore":
		userstoreTest(ctx, tenantURL, tokenSource, iterations)
	case "script":
		scriptTest(ctx, tenantURL, tokenSource, iterations)
	default:
		return ucerr.Errorf("unknown service %v", testName)
	}
	return nil
}

func logErrorf(ctx context.Context, err error, format string, args ...any) {
	uclog.Errorf(ctx, format, args...)
	if envTestCfg.FailFast {
		uclog.Fatalf(ctx, "Environment tests failed - %v", err)
	}
	errMutex.Lock()
	errs = append(errs, err)
	errMutex.Unlock()
}
