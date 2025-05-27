package main

import (
	"context"
	"os"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
)

func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "setcompanytype")
	defer logtransports.Close()

	if len(os.Args) < 2 {
		uclog.Infof(ctx, "Usage: setcompanytype <company type> <tenant ID or name> ")
		uclog.Infof(ctx, "company type is on of the following: %v", companyconfig.AllCompanyTypes)
		uclog.Infof(ctx, "tenant ID or name is the ID or name of the tenant in the company to set the company type")
		uclog.Infof(ctx, "UC_UNIVERSE and UC_REGION environment variables must be set")
		uclog.Fatalf(ctx, "unknown usage")
	}

	companyType := companyconfig.CompanyType(os.Args[1])
	if err := companyType.Validate(); err != nil {
		uclog.Fatalf(ctx, "Invalid company type: %v", err)
	}
	companyStorage := cmdline.GetCompanyStorage(ctx)
	tenant, err := cmdline.GetTenantByIDOrName(ctx, companyStorage, os.Args[2])
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get tenant '%v': %v", os.Args[2], err)
	}
	company, err := companyStorage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get company for tenant '%v': %v", os.Args[2], err)
	}
	if err := companyStorage.SetCompanyType(ctx, company, companyType); err != nil {
		uclog.Fatalf(ctx, "Failed to set company type for company '%v': %v", company.ID, err)
	}
}
