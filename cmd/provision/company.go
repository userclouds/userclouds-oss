package main

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/authz"
	authzProvisioning "userclouds.com/authz/provisioning"
	"userclouds.com/console/tenantinfo"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantdb"
)

type companyFile struct {
	Company companyconfig.Company `json:"company"`
}

//go:generate genvalidate companyFile

func loadCompanies(ctx context.Context, ps *types.ProvisionerState) []companyFile {
	if ps.IsTargetAll() {
		// Load all companies from DB
		companyFiles := []companyFile{}

		pager, err := companyconfig.NewCompanyPaginatorFromOptions(
			pagination.Limit(pagination.MaxLimit))
		if err != nil {
			uclog.Fatalf(ctx, "error applying pagination options: %v", err)
		}

		for {
			companies, respFields, err := ps.CompanyStorage.ListCompaniesPaginated(ctx, *pager)
			if err != nil {
				uclog.Fatalf(ctx, "error listing companies: %v", err)
			}

			for _, company := range companies {
				companyFiles = append(companyFiles, companyFile{Company: company})
			}

			if !pager.AdvanceCursor(*respFields) {
				break
			}
		}
		return companyFiles
	}

	if id := ps.GetUUIDTarget(); !id.IsNil() {
		// Load single company from DB
		existing, err := ps.CompanyStorage.GetCompany(ctx, id)
		if err != nil {
			uclog.Fatalf(ctx, "failed to get existing company: %v", err)
		}
		return []companyFile{{Company: *existing}}
	}

	// Load single company from file
	var provData companyFile
	if err := loadFile(ps.Target, &provData); err != nil {
		uclog.Fatalf(ctx, "failed to load company provisioning file: %v", err)
	}
	return []companyFile{provData}
}

func provisionCompanies(ctx context.Context, ps *types.ProvisionerState, companyFiles []companyFile, validateOnly bool) {
	for _, provData := range companyFiles {
		if err := provisionCompany(ctx, ps, provData, validateOnly); err != nil {
			uclog.Errorf(ctx, "failed to provision company '%s' (id: %s): %v", provData.Company.Name, provData.Company.ID, err)
			promptToDeleteCompany(ctx, ps.CompanyStorage, ps.Simulate, provData.Company)
		}
	}
}

func deprovisionCompanies(ctx context.Context, storage *companyconfig.Storage, simulate bool, companyFiles []companyFile) {
	for _, provData := range companyFiles {
		promptToDeleteCompany(ctx, storage, simulate, provData.Company)
	}
}

func promptToDeleteCompany(ctx context.Context, storage *companyconfig.Storage, simulate bool, company companyconfig.Company) {
	if simulate {
		return
	}

	if types.ConfirmOperationForProd("delete company (NOTE: this doesn't yet de-provision its tenants)") {
		// TODO: Not de-provisioning AuthZ or tenants yet
		pi := types.ProvisionInfo{CompanyStorage: storage}
		po, err := provisioning.NewProvisionableCompany(ctx, "PromptToDelete", pi, &company, uuid.Nil)
		if err != nil {
			uclog.Errorf(ctx, "failed to prepare company '%s' (id: %s) for cleanup: %v", company.Name, company.ID, err)
		}

		if err := po.Cleanup(ctx); err != nil {
			uclog.Errorf(ctx, "tried but failed to cleanup company '%s' (id: %s): %v", company.Name, company.ID, err)
		}
	}
}

func provisionCompany(ctx context.Context, ps *types.ProvisionerState, provData companyFile, validateOnly bool) error {
	if err := provData.Validate(); err != nil {
		return ucerr.Errorf("company failed basic validation: %v", err)
	}

	if ps.Simulate {
		uclog.Infof(ctx, "Successfully simulated provisioning company '%s' (id: %s)", provData.Company.Name, provData.Company.ID)
		return nil
	}
	existing, err := ps.CompanyStorage.GetCompany(ctx, provData.Company.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Errorf("failed to get existing company from DB (and not ErrNoRows): %v", err)
	}

	// if we actually loaded one, ask the user what to do
	var expectedToBeEqual bool
	if err == nil {
		expectedToBeEqual = true
		uclog.Infof(ctx, "Matching company exists: %+v", existing)

		if !cmp.Equal(existing, &provData.Company, ignoreTimes()) {
			uclog.Infof(ctx, "new data differs (- existing, + new):\n%s", cmp.Diff(existing, &provData.Company, ignoreTimes()))
			if !types.ConfirmOperation("overwrite existing company with new data") {
				return nil
			}
			expectedToBeEqual = false
		}
	}

	options := []provisioning.CompanyOption{}
	if !ps.OwnerUserID.IsNil() {
		options = append(options, provisioning.Owner(ps.OwnerUserID))
	}

	consoleInfo, err := tenantinfo.GetConsoleTenantInfo(ctx, ps.CompanyStorage)
	noConsoleTenantInDB := errors.Is(err, sql.ErrNoRows)
	if err != nil && !noConsoleTenantInDB {
		return ucerr.Errorf("failed to get console tenant info from DB: %v", err)
	}
	// If we are provisioning the "root" company (which owns Console), the Console tenant
	// may not yet exist, so we can't set up Company ACLs. This is only a problem
	// for initial bootstrapping. Re-provisioning the company should be fine, however.
	var pi types.ProvisionInfo
	var consoleCompanyID uuid.UUID
	if noConsoleTenantInDB {
		if hasAnyTenants, err := ps.CompanyStorage.HasAnyTenants(ctx); err != nil {
			return ucerr.Wrap(err)
		} else if hasAnyTenants {
			return ucerr.Errorf("there are existing tenants, but none of them are the console (console should be created first)")
		}
		uclog.Infof(ctx, "Creating console tenant company: %s %s", provData.Company.Name, provData.Company.ID)
		pi = types.ProvisionInfo{CompanyStorage: ps.CompanyStorage, TenantDB: nil, TenantID: uuid.Nil}
		consoleCompanyID = provData.Company.ID
	} else {
		uclog.Debugf(ctx, "Console tenant info loaded from DB: %+v", consoleInfo)
		consoleTenantDB, _, _, err := tenantdb.Connect(ctx, ps.CompanyStorage, consoleInfo.TenantID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if consoleInfo.CompanyID == provData.Company.ID {
			uclog.Infof(ctx, "Reprovision Console tenant company: %s %s", provData.Company.Name, provData.Company.ID)
		}
		pi = types.ProvisionInfo{CompanyStorage: ps.CompanyStorage, TenantDB: consoleTenantDB, TenantID: consoleInfo.TenantID}
		consoleCompanyID = consoleInfo.CompanyID
	}

	po, err := provisioning.NewProvisionableCompany(ctx, "ProvCompCMD", pi, &provData.Company, consoleCompanyID, options...)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if !validateOnly {
		if err := po.Provision(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		if err := po.Validate(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	// safety: check what's there now and display a diff
	got, err := ps.CompanyStorage.GetCompany(ctx, provData.Company.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if cmp.Equal(existing, got, ignoreTimes()) != expectedToBeEqual {
		uclog.Infof(ctx, "AFTER SAVE: new company was changed (- existing, + new):\n%s", cmp.Diff(existing, got, ignoreTimes()))
		uclog.Infof(ctx, "If you didn't expect this, you should probably debug now.")
	}

	// if we are provisioning with "setowner", we need to add an object for the owner to each tenant
	if !validateOnly && !ps.OwnerUserID.IsNil() {
		tenants, err := ps.CompanyStorage.ListTenantsForCompany(ctx, provData.Company.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		for _, tenant := range tenants {
			uclog.Infof(ctx, "Provisioning owner object for tenant %s", tenant.ID)
			tenantDB, _, _, err := tenantdb.Connect(ctx, ps.CompanyStorage, tenant.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}
			p := authzProvisioning.NewEntityAuthZ(
				"ProvCompanyTenantOwnerObject",
				types.ProvisionInfo{CompanyStorage: ps.CompanyStorage, TenantDB: tenantDB, TenantID: tenant.ID},
				nil,
				nil,
				[]authz.Object{{BaseModel: ucdb.NewBaseWithID(ps.OwnerUserID), TypeID: authz.UserObjectTypeID, OrganizationID: provData.Company.ID}},
				nil)
			if err := p.Provision(ctx); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}
