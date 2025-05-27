package main

import (
	"context"
	"flag"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/provisioning/types"
)

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "Usage: bin/provision [flags] <operation> <object> <target>")
		uclog.Infof(ctx, "       operation: ")
		uclog.Infof(ctx, "          provision: run provisioning against the given target")
		uclog.Infof(ctx, "          validate: validate provisioning against the given target")
		uclog.Infof(ctx, "          setowner: [company only] set the UUID given in --owner flag to an admin of this company")
		uclog.Infof(ctx, "          deprovision: soft-delete resource")
		uclog.Infof(ctx, "          nuke: [tenant only, dev only] hard-delete underlying resources")
		uclog.Infof(ctx, "          genfile: generate a provisioning file for the given resource")
		uclog.Infof(ctx, "       object: company|tenant|events")
		uclog.Infof(ctx, "       target: <uuid>|<filename>|all")
		uclog.Infof(ctx, "  Flags:")

		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage)
		})
	}

	flag.Parse()
}

func main() {
	ctx := context.Background()
	flagSimulate := flag.Bool("simulate", false, "validate target object(s) without mutating state in the DB")
	flagVerbose := flag.Bool("verbose", false, "enable verbose output")
	flagOwner := flag.String("owner", "", "[company only] UUID of a user to be marked as admin/owner of the resource")
	flagLogfile := flag.String("logfile", "", "logfile name for debug output")
	flagDeep := flag.Bool("deep", false, "deep provisioning validates and corrects relationships between system objects")
	flagUseBaselineSchema := flag.Bool("useBaselineSchema", false, "force migration-by-migration provisioning from baseline schema (if available) instead of using create statements from final schema")
	initFlags(ctx)

	screenLogLevel := uclog.LogLevelInfo
	if *flagVerbose {
		screenLogLevel = uclog.LogLevelVerbose
	}
	logtransports.InitLoggerAndTransportsForTools(ctx, screenLogLevel, uclog.LogLevelVerbose, "provision", logtransports.Filename(*flagLogfile))
	defer logtransports.Close()
	if flag.NArg() != 3 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected command, resource type, and resource id to be specified, got %d: %v", flag.NArg(), flag.Args())
	}

	op := flag.Arg(0)
	targetStr := flag.Arg(2)
	resourceType := flag.Arg(1)
	ownerUserID := uuid.Nil
	if *flagOwner != "" {
		var err error
		ownerUserID, err = uuid.FromString(*flagOwner)
		if err != nil {
			uclog.Fatalf(ctx, "failed to parse 'owner' flag '%s': %v", *flagOwner, err)
		}
	}
	types.ConfirmOperation = func(p string) bool {
		uclog.Debugf(ctx, "%s ? [yN] ", p)
		r := cmdline.Confirm(ctx, "%s ? [yN] ", p)
		uclog.Debugf(ctx, "Response: %v ", r)
		return r
	}

	types.ConfirmOperationForProd = func(prompt string) bool {
		if !types.ConfirmOperation(prompt) {
			return false
		}
		return cmdline.ProductionConfirm(ctx, "I know", "You're in prod. Type 'I know' to confirm: ")
	}

	types.UseBaselineSchema = *flagUseBaselineSchema

	if err := runProvisionTool(ctx, *flagSimulate, *flagDeep, op, targetStr, resourceType, ownerUserID); err != nil {
		flag.Usage()
		uclog.Fatalf(ctx, "error: %v", err)
	}
}
