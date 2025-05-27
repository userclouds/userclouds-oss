package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/provisioning/types"
	provisioningLogServer "userclouds.com/logserver/provisioning"
	"userclouds.com/plex/manager"
)

func runProvisionTool(ctx context.Context, simulate, deep bool, op, targetStr, resourceType string, ownerUserID uuid.UUID) error {
	types.ProvisionMode = types.OfflineProvisionMode
	types.DeepProvisioning = deep

	if op == "nuke" && !universe.Current().IsDev() {
		return ucerr.Errorf("cannot nuke resources in '%s' universe, only dev", universe.Current())
	}

	if (ownerUserID != uuid.Nil) == (op != "setowner") {
		return ucerr.Errorf("cannot use --owner flag with operations other than setowner")
	}

	companyConfigSD, err := companyconfig.GetServiceData(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't get service data: %v", err)
	}
	logServerSD, err := logdb.GetServiceData(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "couldn't get service data: %v", err)
	}
	companyConfigDBCfg := companyConfigSD.DBCfg
	logDBCfg := logServerSD.DBCfg

	var companyStorage *companyconfig.Storage
	if simulate {
		companyStorage = nil
	} else {
		companyStorage = cmdline.GetCompanyStorage(ctx)
	}
	ps := &types.ProvisionerState{
		Simulate:           simulate,
		Deep:               deep,
		Operation:          op,
		ResourceType:       resourceType,
		Target:             targetStr,
		OwnerUserID:        ownerUserID,
		CompanyStorage:     companyStorage,
		CompanyConfigDBCfg: companyConfigDBCfg,
		StatusDBCfg:        logDBCfg,
	}
	switch op {
	case "provision":
		return ucerr.Wrap(runProvision(ctx, ps))
	case "validate":
		return ucerr.Wrap(runValidate(ctx, ps))

	case "deprovision":
		switch resourceType {
		case "company":
			companyFiles := loadCompanies(ctx, ps)
			deprovisionCompanies(ctx, companyStorage, simulate, companyFiles)
		case "tenant":
			tenantFiles := loadTenants(ctx, ps)
			deprovisionTenants(ctx, ps, tenantFiles)
		case "events":
			return ucerr.Wrap(tenantProvisioning.ExecuteProvisioningForEvents(ctx, ps.CompanyConfigDBCfg, ps.CompanyStorage, ps.GetUUIDTarget(), []types.ProvisionOperation{types.Cleanup}))
		default:
			return ucerr.Errorf("error: expected company | tenant | events to be specified as resource type, got %s", resourceType)
		}
	case "nuke":
		switch resourceType {
		case "tenant":
			tenantFiles := loadTenants(ctx, ps)
			nukeTenants(ctx, ps, tenantFiles)
		case "events":
			if err := provisioningLogServer.NukeStaticEvents(ctx, companyConfigDBCfg); err != nil {
				uclog.Fatalf(ctx, "Failed to nuke static events: %v", err)
			}
		default:
			uclog.Fatalf(ctx, "cannot nuke resource of type %s, only tenant", resourceType)
		}
	case "setowner":
		// TODO: we should separate the owner op from the provisioning step, probably?
		if resourceType != "company" {
			return ucerr.Errorf("cannot set owner of resourceType %s, only company", resourceType)
		}
		companyFiles := loadCompanies(ctx, ps)
		provisionCompanies(ctx, ps, companyFiles, false)
	case "genfile":
		return ucerr.Wrap(runGenFile(ctx, companyStorage, targetStr, resourceType))
	}
	return nil
}

func runProvision(ctx context.Context, ps *types.ProvisionerState) error {
	switch ps.ResourceType {
	case "company":
		companyFiles := loadCompanies(ctx, ps)
		provisionCompanies(ctx, ps, companyFiles, false)
	case "tenant":
		tenantFiles := loadTenants(ctx, ps)
		provisionTenants(ctx, ps, tenantFiles, false)
	case "events":
		return ucerr.Wrap(tenantProvisioning.ExecuteProvisioningForEvents(ctx, ps.CompanyConfigDBCfg, ps.CompanyStorage, ps.GetUUIDTarget(), []types.ProvisionOperation{types.Provision, types.Validate}))
	default:
		return ucerr.Errorf("error: expected company | tenant | events to be specified as resource type, got %s", ps.ResourceType)
	}
	return nil
}

func runValidate(ctx context.Context, ps *types.ProvisionerState) error {
	// TODO when we clean up old provisioning code make sure we don't show any prompt or try to make changes in the validate pass
	switch ps.ResourceType {
	case "company":
		companyFiles := loadCompanies(ctx, ps)
		provisionCompanies(ctx, ps, companyFiles, true)
	case "tenant":
		tenantFiles := loadTenants(ctx, ps)
		provisionTenants(ctx, ps, tenantFiles, true)
	case "events":
		return ucerr.Wrap(tenantProvisioning.ExecuteProvisioningForEvents(ctx, ps.CompanyConfigDBCfg, ps.CompanyStorage, ps.GetUUIDTarget(), []types.ProvisionOperation{types.Validate}))
	}
	return nil
}

func runGenFile(ctx context.Context, companyStorage *companyconfig.Storage, targetStr, resourceType string) error {
	targetID := uuid.Must(uuid.FromString(targetStr))
	var data any
	switch resourceType {
	case "company":
		company, err := companyStorage.GetCompany(ctx, targetID)
		if err != nil {
			return ucerr.Errorf("can't load company: %v", err)
		}
		data = companyFile{*company}
	case "tenant":
		tenant, err := companyStorage.GetTenant(ctx, targetID)
		if err != nil {
			return ucerr.Errorf("can't load tenant: %v", err)
		}
		mgr, err := manager.NewFromCompanyConfig(ctx, companyStorage, targetID, nil)
		if err != nil {
			return ucerr.Errorf("can't load tenant manager: %v", err)
		}
		tenantPlex, err := mgr.GetTenantPlex(ctx, targetID)
		if err != nil {
			return ucerr.Errorf("can't load tenant plexconfig: %v", err)
		}
		u, err := url.Parse(tenant.TenantURL)
		if err != nil {
			return ucerr.Errorf("couldn't parse tenant URL: %v", err)
		}
		data = types.TenantFile{
			Tenant:     *tenant,
			PlexConfig: tenantPlex.PlexConfig,
			Protocol:   u.Scheme,
			SubDomain:  strings.SplitAfterN(u.Host, ".", 2)[1],
			// we don't write this out by default because the point of provisioning files is to be
			// DB independent (eg. across many dev boxes), but this will make it easier to generate
			// a file that can be used to provision a tenant to a different cluster
			// TenantDBCfg:     companyConfigDBCfg, // TODO: uncomment me to save out DB override info
		}
	default:
		return ucerr.Errorf("unknown resourceType %s for genfile", resourceType)
	}

	bs, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ucerr.Errorf("failed to marshal data to json: %v", err)
	}
	fmt.Println(string(bs))
	return nil

}

// a cmp.Option to allow us to compare objects while ignoring BaseModel.Created / .Updated
func ignoreTimes() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		return strings.Contains(p.GoString(), "BaseModel.Created") ||
			strings.Contains(p.GoString(), "BaseModel.Updated")
	}, cmp.Ignore())
}
