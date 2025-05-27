package main

import (
	"context"
	"flag"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/uclog"
)

func main() {
	ctx := context.Background()
	noDryRun := flag.Bool("no-dry-run", false, "Disable dry run mode, write/overwrite secrets. Default is dry run mode.")
	overwriteSecrets := flag.Bool("overwrite", false, "Overwrite existing secrets, default is to skip if the secrets already exist.")
	sourceAccount := flag.String("source-account", "", "Source account to copy secrets from (root/debug/staging/prod).")
	secretsFilter := flag.String("filter", "", "Filter for secrets to copy.")
	renameSecrets := flag.Bool("rename", false, "Rename secrets to match the target account. i.e. when copying from debug to staging, rename secrets to include 'staging' in the name instead of 'debug'.")

	flag.Parse()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "copyawssecrets")
	defer logtransports.Close()
	targetUniverse := universe.Current()
	if !targetUniverse.IsCloud() {
		uclog.Fatalf(ctx, "This tool can only be run in a cloud universe, not %v", targetUniverse)
	}
	if len(*secretsFilter) == 0 {
		uclog.Fatalf(ctx, "Must specify a secret filter")
	}
	var cfgSourceAccount aws.Config
	var err error
	var sourceUniverse universe.Universe

	if *sourceAccount == "root" {
		cfgSourceAccount, err = ucaws.NewConfigWithDefaultRegion(ctx)
		if *renameSecrets {
			uclog.Fatalf(ctx, "Can't rename secrets when copying from root account")
		}
	} else {
		sourceUniverse = universe.Universe(*sourceAccount)
		if !sourceUniverse.IsCloud() {
			uclog.Fatalf(ctx, "Source account must be a cloud universe (debug,staging,prod) or root, not %v", sourceUniverse)
		}
		if sourceUniverse == targetUniverse {
			uclog.Fatalf(ctx, "Source account (%v) must be different from target account (%v)", sourceUniverse, targetUniverse)
		}
		cfgSourceAccount, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(*sourceAccount), config.WithRegion(ucaws.DefaultRegion))
	}

	if err != nil {
		uclog.Fatalf(ctx, "Can't create AWS config for root account: %v", err)
	}
	cfgTargetAccount, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(string(targetUniverse)), config.WithRegion(ucaws.DefaultRegion))
	if err != nil {
		uclog.Fatalf(ctx, "Can't create AWS config for sub-account '%v': %v", targetUniverse, err)
	}

	sc := &secretCopier{
		sourceClient:  secretsmanager.NewFromConfig(cfgSourceAccount),
		targetClient:  secretsmanager.NewFromConfig(cfgTargetAccount),
		dryRun:        !*noDryRun,
		overwrite:     *overwriteSecrets,
		sourceAccount: *sourceAccount,
		targetAccount: targetUniverse,
		renameSecrets: *renameSecrets,
	}

	count, err := sc.copySecrets(ctx, *secretsFilter)
	if err != nil {
		uclog.Fatalf(ctx, "error copying secrets: %v", err)
	}
	uclog.Infof(ctx, "processed %d secrets from '%v'", count, *sourceAccount)
}
