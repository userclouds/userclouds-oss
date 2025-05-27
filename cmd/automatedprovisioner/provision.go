package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantplex"
)

type provisionArgs struct {
	tenantFile         *types.TenantFile
	company            *companyconfig.Company
	companyConfigDBCfg *ucdb.Config
	logDBCfg           *ucdb.Config
	cacheCfg           *cache.Config
}

func provisionOrValidateConsole(ctx context.Context, pa provisionArgs, companyStorage *companyconfig.Storage) error {
	if pa.company.ID != pa.tenantFile.Tenant.CompanyID {
		return ucerr.Errorf("company ID  mismatch between company file (%v) and tenant file (%v)", pa.company.ID, pa.tenantFile.Tenant.CompanyID)
	}
	if err := provisionOrValidateCompany(ctx, companyStorage, pa); err != nil {
		return ucerr.Wrap(err)
	}
	if err := provisionOrValidateTenant(ctx, companyStorage, pa); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func provisionOrValidateCompany(ctx context.Context, companyStorage *companyconfig.Storage, pa provisionArgs) error {
	pi := types.ProvisionInfo{CompanyStorage: companyStorage, TenantDB: nil, TenantID: pa.tenantFile.Tenant.ID, CacheCfg: pa.cacheCfg}
	po, err := provisioning.NewProvisionableCompany(ctx, "AutoProvConsoleCompany", pi, pa.company, pa.company.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if lc, err := companyStorage.GetCompany(ctx, pa.company.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	} else if err == nil {
		uclog.Infof(ctx, "Company %v/%v already exists, validating", lc.Name, lc.ID)
		return ucerr.Wrap(po.Validate(ctx))
	} else if err := po.Provision(ctx); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func provisionOrValidateTenant(ctx context.Context, companyStorage *companyconfig.Storage, pa provisionArgs) error {
	tenantID := pa.tenantFile.Tenant.ID
	if _, err := companyStorage.GetTenant(ctx, tenantID); err == nil {
		uclog.Infof(ctx, "Tenant %v already exists, Validating", tenantID)
		pt, err := tenantProvisioning.NewProvisionableTenantFromExisting(ctx,
			"AutoProvConsoleTenant",
			tenantID,
			companyStorage,
			pa.companyConfigDBCfg,
			pa.logDBCfg,
			pa.cacheCfg,
		)
		if err != nil {
			return ucerr.Wrap(err)
		}
		return ucerr.Wrap(pt.Validate(ctx))
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Errorf("error checking for existing tenant ID %v: %w", tenantID, err)
	}
	return ucerr.Wrap(provisionConsoleTenant(ctx, companyStorage, pa))
}
func provisionConsoleTenant(ctx context.Context, companyStorage *companyconfig.Storage, pa provisionArgs) error {
	tenantFile := pa.tenantFile
	tenant := tenantFile.Tenant
	tpKeys, err := provisioning.GeneratePlexKeys(ctx, tenant.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	tenantFile.PlexConfig.Keys = *tpKeys
	if err := pa.tenantFile.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	pt, err := tenantProvisioning.NewProvisionableTenant(ctx,
		"AutoProvConsoleTenant",
		&tenant,
		&tenantplex.TenantPlex{VersionBaseModel: ucdb.NewVersionBaseWithID(tenant.ID), PlexConfig: tenantFile.PlexConfig},
		companyStorage,
		pa.companyConfigDBCfg,
		pa.tenantFile.TenantDBCfg,
		pa.logDBCfg,
		pa.cacheCfg,
		[]uuid.UUID{},
	)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := pt.Provision(ctx); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func loadTenantFile(ctx context.Context, basePath string, customerDomain, companyName, googleClientID, adminUserEmail string) (*types.TenantFile, error) {
	var tf types.TenantFile
	tmpl, err := template.ParseFiles(filepath.Join(basePath, "tenant_console.json.tmpl"))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	var buf bytes.Buffer
	data := struct {
		CustomerDomain, CompanyName, GoogleClientID, AdminUserEmail string
	}{
		CompanyName:    companyName,
		CustomerDomain: customerDomain,
		GoogleClientID: googleClientID,
		AdminUserEmail: adminUserEmail,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, ucerr.Wrap(err)
	}
	jsonData := buf.Bytes()
	if err := json.Unmarshal(jsonData, &tf); err != nil {
		uclog.Errorf(ctx, "tenant file content: %v", string(jsonData))
		return nil, ucerr.Wrap(err)
	}
	for i := range tf.PlexConfig.PlexMap.Apps {
		tf.PlexConfig.PlexMap.Apps[i].ClientID = crypto.GenerateClientID()
	}
	tf.PlexConfig.PlexMap.EmployeeApp.ClientID = crypto.GenerateClientID()
	return &tf, nil
}

func loadProvisionData(ctx context.Context, basePath string) (*companyconfig.Company, *types.TenantFile, error) {
	companyName, err := lookupEnvVariable("COMPANY_NAME")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	customerDomain, err := lookupEnvVariable("CUSTOMER_DOMAIN")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	googleClientID, err := lookupEnvVariable("GOOGLE_CLIENT_ID")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	adminUserEmail, err := lookupEnvVariable("ADMIN_USER_EMAIL")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "Loading provisioning files for company '%s' and customer domain '%s' with admin user '%s'", companyName, customerDomain, adminUserEmail)
	tf, err := loadTenantFile(ctx, basePath, customerDomain, companyName, googleClientID, adminUserEmail)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	company := companyconfig.Company{BaseModel: ucdb.NewBaseWithID(tf.Tenant.CompanyID), Name: companyName, Type: companyconfig.CompanyTypeCustomer}
	return &company, tf, nil
}

func lookupEnvVariable(envName string) (string, error) {
	env, ok := os.LookupEnv(envName)
	if !ok || env == "" {
		return "", ucerr.Errorf("%s env var not set", envName)
	}
	return env, nil
}
