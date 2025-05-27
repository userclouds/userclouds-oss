package main

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/testhelpers"
)

func TestTenantReprovision(t *testing.T) {
	ctx := context.Background()
	ocfg, lcfg, companyStorage := testhelpers.NewTestStorage(t)
	testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, companyStorage, ocfg, lcfg)
	ps := &types.ProvisionerState{
		Simulate:           false,
		Deep:               false,
		Operation:          "N/A",
		ResourceType:       "N/A",
		Target:             "all",
		OwnerUserID:        uuid.Nil,
		CompanyStorage:     companyStorage,
		CompanyConfigDBCfg: ocfg,
		StatusDBCfg:        lcfg,
	}
	tenantFiles := loadTenants(ctx, ps)
	provisionTenants(ctx, ps, tenantFiles, false)
	// TODO: check that nothing actually changed here, including things like new columns/accessors?
}
