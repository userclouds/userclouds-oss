package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/worker"
)

var flagList *bool
var flagAddRegion *string
var flagRemoveRegion *string

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "Usage: remoteuserregionconfig <uuid of existing tenant>")
		uclog.Infof(ctx, "UC_UNIVERSE environment variable must be set")
		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage)
		})
	}

	flagList = flag.Bool("list", false, "list the existing user regions")
	flagAddRegion = flag.String("add", "", "add a new user region")
	flagRemoveRegion = flag.String("remove", "", "remove a user region")

	flag.Parse()
}

// Config holds config info for IDP
type Config struct {
	RemoteUserDBs map[region.DataRegion]ucdb.Config `yaml:"remote_user_region_bootstrap_db_configs"`
}

//go:generate genvalidate Config

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "remoteuserregionconfig")
	defer logtransports.Close()

	var cfg Config
	if err := yamlconfig.LoadToolConfig(ctx, "remoteuserregion", &cfg); err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	initFlags(ctx)

	if flag.NArg() == 0 {
		flag.Usage()
		uclog.Fatalf(ctx, "Expected a UUID, instead got %d args: %v", len(os.Args), os.Args)
	}

	tenantID, err := uuid.FromString(flag.Arg(0))
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse tenant ID: %v", err)
	}

	ccs := cmdline.GetCompanyStorage(ctx)
	tenantInternal, err := ccs.GetTenantInternal(ctx, tenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get tenant: %v", err)
	}

	if *flagList {
		fmt.Printf("\n")
		fmt.Printf("User regions for tenant %s:\n", tenantID)
		if tenantInternal.PrimaryUserRegion == "" {
			fmt.Printf("	primary user region missing\n")
		} else {
			fmt.Printf("	%s (primary)\n", tenantInternal.PrimaryUserRegion)
		}
		for region := range tenantInternal.RemoteUserRegionDBConfigs {
			fmt.Printf("	%s (remote)\n", region)
		}
		fmt.Printf("\n")
		return
	}

	if *flagAddRegion != "" {
		remoteDataRegion := region.DataRegion(*flagAddRegion)
		if err := remoteDataRegion.Validate(); err != nil {
			uclog.Fatalf(ctx, "Invalid remote user region: %v", err)
		}

		if remoteDataRegion == tenantInternal.PrimaryUserRegion {
			uclog.Fatalf(ctx, "Remote user region %s is already the primary user region", remoteDataRegion)
		}
		if _, ok := tenantInternal.RemoteUserRegionDBConfigs[remoteDataRegion]; ok {
			uclog.Fatalf(ctx, "Remote user region %s already exists for tenant", remoteDataRegion)
		}

		bootstrapDBCfg, ok := cfg.RemoteUserDBs[remoteDataRegion]
		if !ok {
			uclog.Fatalf(ctx, "No DB config found for remote user region %s", remoteDataRegion)
		}

		s, err := secret.NewString(ctx, "console", fmt.Sprintf("%s-%s-dbpassword", tenantInternal.TenantDBConfig.DBName, remoteDataRegion), crypto.MustRandomHex(24))
		if err != nil {
			uclog.Fatalf(ctx, "failed to generate secret for remote tenant db password: %v", err)
		}
		dbNameComponents := strings.Split(tenantInternal.TenantDBConfig.DBName, ".")
		remoteTenantDBName := dbNameComponents[len(dbNameComponents)-1]

		newDBConfig := ucdb.Config{
			Host:          bootstrapDBCfg.Host,
			Port:          "5432",
			RegionalHosts: bootstrapDBCfg.RegionalHosts,
			User:          tenantInternal.TenantDBConfig.User,
			Password:      *s,
			DBName:        remoteTenantDBName,
			DBDriver:      ucdb.PostgresDriver,
			DBProduct:     ucdb.AWSAuroraPostgres,
		}

		pdb := provisioning.NewProvisionableDBFromExistingConfigs(tenantInternal.TenantDBConfig.DBName,
			&bootstrapDBCfg,
			&newDBConfig,
			nil, // we're not overriding DB config here, we're operating entirely in the new cluster
			tenantdb.Schema,
			&tenantdb.SchemaBaseline)
		_ = pdb
		if err := pdb.Provision(ctx); err != nil {
			uclog.Fatalf(ctx, "failed to provision new tenantdb: %v", err)
		}

		tenantInternal.RemoteUserRegionDBConfigs[remoteDataRegion] = newDBConfig
		updateTenantInternal(ctx, tenantInternal)
		return
	}

	if *flagRemoveRegion != "" {
		remoteDataRegion := region.DataRegion(*flagRemoveRegion)
		if err := remoteDataRegion.Validate(); err != nil {
			uclog.Fatalf(ctx, "Invalid remote user region: %v", err)
		}

		if _, ok := tenantInternal.RemoteUserRegionDBConfigs[remoteDataRegion]; !ok {
			uclog.Fatalf(ctx, "Remote user region %s does not exist for tenant", remoteDataRegion)
		}

		delete(tenantInternal.RemoteUserRegionDBConfigs, remoteDataRegion)
		updateTenantInternal(ctx, tenantInternal)
		return
	}
}

func updateTenantInternal(ctx context.Context, tenantInternal *companyconfig.TenantInternal) {
	uclog.Infof(ctx, "Sending worker message to save Tenant Internal for ID: %v", tenantInternal.ID)
	wc, err := cmdline.GetWorkerClientForTool(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get worker client: %v", err)
	}
	msg := worker.SaveTenantInternalMessage(tenantInternal)
	if err := wc.Send(ctx, msg); err != nil {
		uclog.Fatalf(ctx, "failed to send message: %v", err)
	}
}
