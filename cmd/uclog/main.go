package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

var timeWindowArg *string
var regionArg *string
var serviceArg *string
var streamNameArg *string
var fileNameArg *string
var interactiveArg *bool
var verboseArg *bool
var filterServicesArg *string
var callSummaryArg *bool
var summaryArg *bool
var slowCallsArg *int
var liveArg *bool
var perfSummaryPrefixArg *string
var tenantURLArg *string
var clientIDArg *string
var clientSecretArg *string
var outpuPrefixArg *string
var ignoreHTTPErrorsArg *string
var writeRawLogsArg *bool
var outputLogDataToScreenArg *bool

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "usage: bin/uclog [flags] [command] <tenantid>")
		uclog.Infof(ctx, "example: bin/uclog --time 60 --service console --region aws-us-west-2 listlog 804718b8-5788-4c31-9f5e-1ac55782728c")
		uclog.Infof(ctx, "example: bin/uclog --time 60 --streamname staging --interactive --filterserv logserver listlog")
		uclog.Infof(ctx, "command: listlog - dumps contents of the kinesis stream from given time in the past (default 30 min)")
		uclog.Infof(ctx, "command: listmetrics - dumps metrics stream from given time in the past (default 30 min)")
		uclog.Infof(ctx, "command: deregister - cleans up any subscriptions to the stream that are older then 24 hours")

		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage)
		})
	}
	timeWindowArg = flag.String("time", "30", "number of minutes of log file to retrieve, can expressed as an interval 30,10 which will retrieve 20 minutes of logs")
	regionArg = flag.String("region", "", "UC region (like aws-us-west-2)")
	serviceArg = flag.String("service", "", "userclouds service")
	streamNameArg = flag.String("streamname", "", "overrides the stream name for stream operations like listlog")
	fileNameArg = flag.String("filename", "", "uses the file as input instead of the stream for operations like listlog")
	interactiveArg = flag.Bool("interactive", false, "colors output for the console")
	verboseArg = flag.Bool("verbose", false, "include verbose output")
	filterServicesArg = flag.String("filterserv", "", "comma separated list of service names to filter out")
	callSummaryArg = flag.Bool("callsummary", false, "display list of all unique HTTP calls in the log")
	summaryArg = flag.Bool("summary", false, "display lists of login ids, failed HTTP calls, tenants in the log")
	liveArg = flag.Bool("live", false, "causes listlog to continue to output the logs coming until Ctrl+C is pressed")
	perfSummaryPrefixArg = flag.String("perfsum", "", "creates a call time summary for all calls to that url")
	outpuPrefixArg = flag.String("outputpref", "s", "define a prefix for log lines s - service, t - tenant, r - region, h - host")
	ignoreHTTPErrorsArg = flag.String("ignorehttpcode", "", "comma separated list of http codes >= 400 to not count as errors")
	slowCallsArg = flag.Int("slowcalls", 0, "display calls that took longer then this many milliseconds in the call summary")
	writeRawLogsArg = flag.Bool("writerawlogs", false, "write raw logs to a file so they can be replayed later")
	outputLogDataToScreenArg = flag.Bool("outputtoscreen", false, "output log data to screen in addition to the logfile")

	tenantURLArg = flag.String("tenanturl", "", "url for the tenant")
	clientIDArg = flag.String("clientid", "", "client id")
	clientSecretArg = flag.String("clientsecret", "", "client secret")

	flag.Parse()
}

// takes universe from tne env var, and the service name on the command line
func main() {
	ctx := context.Background()

	// Initialize the log file
	logfile := fmt.Sprintf("/tmp/uclog.%d", time.Now().UTC().UnixNano())
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "uclog", logtransports.NoPrefix(), logtransports.Filename(logfile))
	defer logtransports.Close()

	initFlags(ctx)

	if flag.NArg() == 0 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected command to be specified, got %d: %v", flag.NArg(), flag.Args())
	}

	commandName := flag.Arg(0)

	if commandName == "listlog" && flag.NArg() > 2 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected at most 2 non-flag args, got %d: %v", flag.NArg(), flag.Args())
	}

	tenantID := uuid.Nil
	var err error
	if flag.NArg() == 2 {
		tenantID, err = uuid.FromString(flag.Arg(1))
		if err != nil {
			uclog.Fatalf(ctx, "error: couldn't parse specified tenant_id, got %s", flag.Arg(1))
		}
	}

	// a little more param validation
	// TODO: this should be centralized and complete, I've just added cases that bit me so far
	if regionArg != nil && *regionArg != "" {
		reg := region.MachineRegion(*regionArg)
		if err := reg.Validate(); err != nil {
			uclog.Fatalf(ctx, "invalid region %v -- must be one of %v", *regionArg, region.MachineRegionsForUniverse(universe.Current()))
		}
	}

	if serviceArg != nil && *serviceArg != "" && !service.IsValid(service.Service(*serviceArg)) {
		uclog.Fatalf(ctx, "invalid service %v -- must be one of %v", *serviceArg, service.AllServices)
	}
	svc := service.Service(*serviceArg)

	if commandName == "listmetrics" &&
		((clientIDArg == nil || *clientIDArg == "") ||
			(clientSecretArg == nil || *clientSecretArg == "") ||
			(tenantURLArg == nil || *tenantURLArg == "")) {
		uclog.Fatalf(ctx, "listmetrics requires tenant URL, client ID, and client secret")
	}

	if commandName == "listmetrics" &&
		(ignoreHTTPErrorsArg != nil && *ignoreHTTPErrorsArg != "") {
		uclog.Fatalf(ctx, "ignoreHttpErrorsArg doesn't apply tp listmetrics command")
	}

	if commandName == "listlog" && *fileNameArg != "" {
		if *streamNameArg != "" || *liveArg {
			uclog.Fatalf(ctx, "If input filename is specificed streamname and live can't be used")
		}
		if *writeRawLogsArg {
			uclog.Fatalf(ctx, "If input filename is specificed writerawlogs can't be used")
		}
	}
	uclog.Debugf(ctx, "Dataprocessor started with command %s on %v", commandName, tenantID)

	switch commandName {
	case "listlog":
		err = listLog(ctx, tenantID, svc)
	case "listmetrics":
		err = ListMetricsForService(ctx, tenantID, *tenantURLArg, *clientIDArg, *clientSecretArg, *serviceArg, *timeWindowArg,
			*filterServicesArg, *callSummaryArg, *summaryArg)
	case "deregister":
		err = deregister(ctx, tenantID, svc)
	default:
		uclog.Fatalf(ctx, "error: unknown command - %s", commandName)
	}

	if err != nil {
		uclog.Fatalf(ctx, "error: command %s on %v failed %v", commandName, tenantID, err)
	}

	uclog.Debugf(ctx, "Log file - %s ", logfile)
}

func deregister(ctx context.Context, tenantID uuid.UUID, svc service.Service) error {
	sc, err := newStreamClient(ctx, *streamNameArg, tenantID, *regionArg, svc)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(sc.deregisterConsumer(ctx))
}

func listLog(ctx context.Context, tenantID uuid.UUID, svc service.Service) error {
	sc, err := newStreamClient(ctx, *streamNameArg, tenantID, *regionArg, svc)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if *streamNameArg != "" {
		err = sc.deregisterConsumer(ctx)
		if err != nil {
			uclog.Errorf(ctx, "error: failed to deregister expired consumers - %v", err)
		}
	}
	return ucerr.Wrap(ListRegionLogsForService(ctx, sc, tenantID, *regionArg, svc, *timeWindowArg, *streamNameArg, *interactiveArg,
		*filterServicesArg, *verboseArg, *callSummaryArg, *summaryArg, *slowCallsArg, *liveArg, *perfSummaryPrefixArg, *outpuPrefixArg, *ignoreHTTPErrorsArg,
		*fileNameArg, *writeRawLogsArg, *outputLogDataToScreenArg))
}
