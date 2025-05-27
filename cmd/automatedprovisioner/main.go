package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/provisioning/types"
)

const (
	defaultBaseProvisionFilesPath = "config/provisioning/onprem"
	// see: helm/userclouds-on-prem/templates/provision-job.yaml
	skipEnsureAWSSecretsAccessEnvVar = "SKIP_ENSURE_AWS_SECRETS_ACCESS"
)

func main() {
	ctx := context.Background()
	startTime := time.Now().UTC()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelVerbose, uclog.LogLevelNonMessage, "automatedprovisioner", logtransports.UseJSONLog())
	uv := universe.Current()
	if !uv.IsOnPremOrContainer() {
		uclog.Fatalf(ctx, "automated provisioner not supported for '%v'", uv)
	}
	if value, ok := os.LookupEnv(skipEnsureAWSSecretsAccessEnvVar); ok && value == "true" {
		uclog.Infof(ctx, "Skipping AWS Secrets Manager access ensured")
	} else if err := ensureAWSSecretsAccess(ctx); err != nil {
		uclog.Fatalf(ctx, "Failed to ensure AWS Secrets Manager access: %v", err)
	}
	// load early so we bail out instead of failing later
	baseProvisionFilesPath, ok := os.LookupEnv("UC_BASE_PROVISION_FILES_PATH")
	if !ok {
		baseProvisionFilesPath = defaultBaseProvisionFilesPath
	}
	company, tf, err := loadProvisionData(ctx, baseProvisionFilesPath)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to load provisioning files: '%v'", err)
	}
	tenantDBDownMigrate := -1
	downMigrateRequest, ok := os.LookupEnv("TENANT_DB_DOWN_MIGRATE_DB_VERSION")
	if ok {
		if tenantDBDownMigrate, err = strconv.Atoi(downMigrateRequest); err != nil {
			uclog.Fatalf(ctx, "Failed to parse TENANT_DB_DOWN_MIGRATE_DB_VERSION: '%s' %v", downMigrateRequest, err)
		}
		uclog.Infof(ctx, "Down migrating tenantdb to version %d", tenantDBDownMigrate)
	}

	serviceData, err := migrateDatabases(ctx, uv, tenantDBDownMigrate)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to migrate databases: %v", err)
	}
	provisionArgs := provisionArgs{
		tenantFile:         tf,
		company:            company,
		companyConfigDBCfg: serviceData["companyconfig"].DBCfg,
		logDBCfg:           serviceData["status"].DBCfg,
		cacheCfg:           nil,
	}
	companyStorage := cmdline.GetCompanyStorage(ctx)
	if err := provisionOrValidateConsole(ctx, provisionArgs, companyStorage); err != nil {
		uclog.Fatalf(ctx, "Failed to provision console tenant: %v", err)
	}
	if err := provisionEvents(ctx, provisionArgs.companyConfigDBCfg, companyStorage); err != nil {
		uclog.Fatalf(ctx, "Failed to provision or validate events: %v", err)
	}
	uclog.Infof(ctx, "Automated Provisioning complete. took %v", time.Now().UTC().Sub(startTime))
}

func provisionEvents(ctx context.Context, companyConfigDBCfg *ucdb.Config, companyStorage *companyconfig.Storage) error {
	return ucerr.Wrap(tenantProvisioning.ExecuteProvisioningForEvents(ctx, companyConfigDBCfg, companyStorage, uuid.Nil, []types.ProvisionOperation{types.Provision, types.Validate}))
}

func ensureAWSSecretsAccess(ctx context.Context) error {
	secretName := uuid.Must(uuid.NewV1()).String()
	uclog.Infof(ctx, "Ensuring AWS Secrets Manager access: %s", secretName)
	fakeSecret, err := secret.NewString(ctx, "ensureaccess", secretName, "test-access")
	if err != nil {
		return ucerr.Wrap(err)
	}
	if value, err := fakeSecret.Resolve(ctx); err != nil {
		return ucerr.Wrap(err)
	} else if value != "test-access" {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "AWS Secrets Manager access ensured")
	return nil
}
