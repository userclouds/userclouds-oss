package main

import (
	"context"
	"flag"
	"net/url"
	"strings"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
)

var verbose *bool
var screenLogLevel = uclog.LogLevelInfo

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "usage: bin/tenantcopy [flags] [services] <tenanturl_src> <clientid_src> <clientsecret_src> <tenanturl_dest> <clientid_dest> <clientsecret_dest>")
		uclog.Infof(ctx, "service: all - copies data for all services")
		uclog.Infof(ctx, "service: authz - copies data for authz service")
		uclog.Infof(ctx, "service: userstore - runs a set of userstore tests")
		uclog.Infof(ctx, "<tenanturl_src> <clientid_src> <clientsecret_src> <tenanturl_dest> <clientid_dest> <clientsecret_dest>: src and dest tenants")
		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage)
		})
	}

	verbose = flag.Bool("verbose", false, "enable verbose output")

	flag.Parse()
}

type tenantRecord struct {
	tenantURL    string
	clientID     string
	clientSecret string
	authzURL     *url.URL
	tokenSource  jsonclient.Option
}

func main() {
	ctx := context.Background()

	initFlags(ctx)

	if flag.NArg() != 7 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected 1 service name and src/dst tenant connection info to be specified, got %d: %v", flag.NArg(), flag.Args())
	}

	if *verbose {
		screenLogLevel = uclog.LogLevelDebug
	}

	logtransports.InitLoggerAndTransportsForTools(ctx, screenLogLevel, uclog.LogLevelVerbose, "tenantcopy")
	defer logtransports.Close()

	service := flag.Arg(0)

	tenantSrc := tenantRecord{tenantURL: flag.Arg(1), clientID: flag.Arg(2), clientSecret: flag.Arg(3)}
	tenantDst := tenantRecord{tenantURL: flag.Arg(4), clientID: flag.Arg(5), clientSecret: flag.Arg(6)}

	services := []string{}

	if service == "all" {
		services = []string{"authz"} // TODO  "tokenizer", "userstore, "authn", "logserver"
	} else {
		services = append(services, strings.Split(service, ",")...)
	}

	uclog.Infof(ctx, "Copying data for services %v from [%s] to [%s]", services, tenantSrc.tenantURL, tenantDst.tenantURL)

	initAuth(ctx, &tenantSrc)
	initAuth(ctx, &tenantDst)

	for _, s := range services {
		switch s {

		case "authz":
			if err := copyAuthz(ctx, &tenantSrc, &tenantDst); err != nil {
				uclog.Fatalf(ctx, "Authz data copy failed with %v", err)
			}

		default:
			uclog.Errorf(ctx, "unknown service %v", s)
		}
	}

}

func initAuth(ctx context.Context, tr *tenantRecord) {
	authzURL, err := url.Parse(tr.tenantURL)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse tenant URL %s: %v", tr.tenantURL, err)
	}
	ts, err := jsonclient.ClientCredentialsForURL(tr.tenantURL, tr.clientID, tr.clientSecret, nil)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create token source for %s: %v", tr.tenantURL, err)
	}
	tr.authzURL = authzURL
	tr.tokenSource = ts
}
