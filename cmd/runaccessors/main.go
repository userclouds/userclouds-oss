package main

import (
	"context"

	"github.com/alecthomas/kong"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
)

type runAccessorCLI struct {
	Iterations      int      `help:"number times to run tests" default:"1"`
	Verbose         bool     `help:"enable verbose output"`
	Tenant          string   `help:"Tenant id or name against which the load test will be running" required:""`
	Logfile         string   `help:"logfile name for debug output" default:""`
	AccessorID      string   `help:"accessor id to run" required:""`
	Search          []string `arg:"" help:"search string to run" required:""`
	ContinueOnError bool     `help:"continue on error"`
}

func (cli runAccessorCLI) Validate() error {
	if cli.Iterations < 1 {
		return ucerr.New("iterations must be greater than 0")
	}
	if _, err := uuid.FromString(cli.AccessorID); err != nil {
		return ucerr.Errorf("invalid accessor id '%v'", cli.AccessorID)
	}
	return nil
}

func main() {
	ctx := context.Background()
	var cli runAccessorCLI
	kong.Parse(&cli, kong.UsageOnError())

	screenLogLevel := uclog.LogLevelInfo
	if cli.Verbose {
		screenLogLevel = uclog.LogLevelDebug
	}

	options := make([]logtransports.ToolLogOption, 0, 1)
	if cli.Logfile != "" {
		options = append(options, logtransports.Filename(cli.Logfile))
	}

	logtransports.InitLoggerAndTransportsForTools(ctx, screenLogLevel, uclog.LogLevelVerbose, "runaccessors", options...)
	defer logtransports.Close()
	storage := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, storage, cli.Tenant)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't load tenant by ID or name: %v: %v", cli.Tenant, err)
	}
	tenantURL, err := cmdline.GetTenantURL(ctx, tenant.TenantURL, false, false)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't get tenant URL: %v", err)
	}
	uclog.Infof(ctx, "Running accessor %s on [%s]", cli.AccessorID, tenantURL)
	tokenSource, err := cmdline.GetTokenSourceForTenant(ctx, storage, tenant, tenantURL)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't get token source: %v", err)
	}
	accessorID := uuid.Must(uuid.FromString(cli.AccessorID))
	if err := runAccessor(ctx, tenantURL, tokenSource, cli.Iterations, accessorID, !cli.ContinueOnError, cli.Search); err != nil {
		uclog.Fatalf(ctx, "Error running accessor: %v", err)
	}
}
