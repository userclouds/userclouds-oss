package main

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
)

var flagLogDB = flag.Bool("logdb", false, "connect to the logging DB instead of the tenant DB")
var flagPrompt = flag.Bool("prompt", false, "prompt for tenant ID rather than passing as an arg")
var flagRegion = flag.String("region", "", "connect to a node in a specific region (Cockroach only)") // blank translates to us-west-2 today
var flagUserRegion = flag.String("userregion", "", "connect to a user region database in a specific region")
var flagCompany = flag.Bool("companies", false, "search for a tenant by company name instead of tenant name")
var flagSearch = flag.Bool("search", false, "search for a tenant by name, and potentially return a list to choose from")

// dbshell lets you connect to a tenant DB directly
// TODO: enable listing / searching tenants from here?
// TODO: integrate with make dbshell
func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "dbshell")
	defer logtransports.Close()

	flag.Parse()

	if len(flag.Args()) < 1 && !*flagPrompt {
		uclog.Debugf(ctx, "Usage: tenantdbshell [flags] [tenant ID or name]")
		uclog.Debugf(ctx, "(uses universe from the environment var UC_UNIVERSE, region from env var UC_REGION for config)")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "Interface flags:")
		uclog.Debugf(ctx, "  --companies: only valid with --search, and searches for company name instead of tenant name")
		uclog.Debugf(ctx, "  --search: search for a tenant by name, and potentially return a list to choose from")
		uclog.Debugf(ctx, "  --prompt: prompt for tenant ID rather than passing as an arg")
		uclog.Debugf(ctx, "")
		uclog.Debugf(ctx, "Connection flags:")
		uclog.Debugf(ctx, "  --logdb: connect to the logging DB instead of the tenant DB")
		uclog.Debugf(ctx, "  --region: connect to the tenant DB in a specific region (default: us-west-2)") // default because this is what we store in tenant_db_config
		uclog.Debugf(ctx, "  --userregion: connect to a user region database in a specific region")

		uclog.Fatalf(ctx, "Expected a tenantID, instead got %d: %v", flag.NArg(), flag.Args())
	}

	if *flagCompany && !*flagSearch {
		// if there's a need, could certainly allow company ID to be provided and generate a list of tenants,
		// but we don't use company IDs many places today
		uclog.Fatalf(ctx, "Cannot use --company without --search at this time.")
	}

	var input string
	if *flagPrompt {
		if *flagCompany {
			input = cmdline.ReadConsole(ctx, "Enter company name: ")
		} else if *flagSearch {
			input = cmdline.ReadConsole(ctx, "Enter tenant name: ")
		} else {
			input = cmdline.ReadConsole(ctx, "Enter tenant ID or name: ")
		}
	} else {
		input = flag.Args()[0]
	}

	ccs := cmdline.GetCompanyStorage(ctx)
	var tdbConfig *ucdb.Config
	var err error
	if *flagSearch {
		tdbConfig, err = search(ctx, ccs, input, *flagCompany)
		if err != nil {
			uclog.Fatalf(ctx, "couldn't find company or tenant: %v", err)
		}
	} else {
		var err error
		tdbConfig, err = getTenantDBConfig(ctx, ccs, input)
		if err != nil {
			uclog.Fatalf(ctx, "error getting tenant config: %v", err)
		}
	}

	dbName := tdbConfig.DBName
	if strings.Contains(dbName, ".") {
		parts := strings.Split(dbName, ".")
		dbName = parts[1]
	}

	var args []string
	password, err := tdbConfig.Password.Resolve(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "error resolving password: %v", err)
	}

	// Select path between x86 and apple silicon
	psqlPath := "/opt/homebrew/bin/psql"
	if _, err := os.Lstat(psqlPath); err != nil {
		psqlPath = "/usr/local/bin/psql"
	}

	var cmd *exec.Cmd
	if *flagLogDB {
		if *flagRegion != "" && *flagRegion != "us-west-2" {
			uclog.Fatalf(ctx, "logdb only runs in us-west-2 today")
		}

		args = append(args, tdbConfig.DBName, "--user", tdbConfig.User, "--port", tdbConfig.Port)
		cmd = exec.Command(psqlPath, args...)
		cmd.Env = append(cmd.Env, "PGPASSWORD="+password)
	} else if tdbConfig.IsProductPostgres() {
		host := tdbConfig.Host
		if *flagRegion != "" {
			if h, ok := tdbConfig.RegionalHosts[*flagRegion]; ok {
				host = h
			} else {
				uclog.Fatalf(ctx, "no host found for region %s", *flagRegion)
			}
		}
		args = append(args, "-h", host, "-p", tdbConfig.Port, "-U", tdbConfig.User, "-d", dbName)
		cmd = exec.Command(psqlPath, args...)
		cmd.Env = append(cmd.Env, "PGPASSWORD="+password)
	} else {
		uclog.Fatalf(ctx, "unsupported DB type: %v", tdbConfig.DBProduct)
	}

	// we just map everything to the console we're currently on so it's passthrough
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		uclog.Errorf(ctx, "error running sql: %v", err)
	}
}

func getTenantDBConfig(ctx context.Context, ccs *companyconfig.Storage, IDorName string) (*ucdb.Config, error) {
	tenant, err := cmdline.GetTenantByIDOrName(ctx, ccs, IDorName)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return getTenantDBConfigForID(ctx, ccs, tenant.ID)
}

func getTenantDBConfigForID(ctx context.Context, ccs *companyconfig.Storage, id uuid.UUID) (*ucdb.Config, error) {
	tenConfig, err := ccs.GetTenantInternal(ctx, id)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if *flagLogDB {
		return &tenConfig.LogConfig.LogDB, nil
	}

	if *flagUserRegion != "" && *flagUserRegion != string(tenConfig.PrimaryUserRegion) {
		reg := region.DataRegion(*flagUserRegion)
		if err := reg.Validate(); err != nil {
			return nil, ucerr.Errorf("invalid user region %s: %s", *flagUserRegion, err)
		}
		dbConfig, ok := tenConfig.RemoteUserRegionDBConfigs[reg]
		if !ok {
			return nil, ucerr.Errorf("no user region %s found for tenant with ID %s", *flagUserRegion, id)
		}
		return &dbConfig, nil
	}

	return &tenConfig.TenantDBConfig, nil
}

func search(ctx context.Context, ccs *companyconfig.Storage, query string, isCompany bool) (*ucdb.Config, error) {
	if isCompany {
		return searchCompanies(ctx, ccs, query)
	}
	return searchTenants(ctx, ccs, query)
}
