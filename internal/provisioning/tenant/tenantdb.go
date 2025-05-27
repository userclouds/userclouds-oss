package tenant

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/tenantdb"
)

// TODO: this needs to be wrapped in a constructor rather than randomly partially init'd by callers
type tenantDB struct {
	tenantID uuid.UUID
	company  *companyconfig.Company

	companyConfigStorage *companyconfig.Storage

	// bootstrapDBCfg is any valid DB connection which can be used to create
	// new databases in the same cluster.
	bootstrapDBCfg *ucdb.Config

	// this is normally nil but if present, forces provisioning to this cluster
	// it's just passed through to provisionableDB in initProvisionableDB(), but
	// we need to store it here "in transit" to that struct because the place we know
	// about overrides is in `cmd/provision` and that doesn't reach past tenantDB
	overrideBootstrapDBCfg *ucdb.Config

	// managed centrally at from tenant so that we don't save intermediate/invalid states
	tenantInternal *companyconfig.TenantInternal

	// save this so we can reuse it
	TenantDB *ucdb.DB
}

func (t *tenantDB) initProvisionableDB(ctx context.Context, currTI *companyconfig.TenantInternal) (*provisioning.ProvisionableDB, error) {
	name := fmt.Sprintf("Tenant DB (tenant id: %s)", t.tenantID)
	var pdb *provisioning.ProvisionableDB
	var err error
	// TODO: unify tenantdb.go and logdb.go, this logic needs extracted
	if currTI.TenantDBConfig.Validate() != nil {
		// Unprovisioned (or incompletely provisioned tenant).
		// NOTE: If the config fails to validate, we assume it must be totally re-created.
		resourceNames := provisioning.NewTenantBaseResourceNames(t.tenantID)
		bootstrapDBCfg := t.bootstrapDBCfg
		if t.overrideBootstrapDBCfg != nil {
			bootstrapDBCfg = t.overrideBootstrapDBCfg
		}
		pdb, err = provisioning.NewProvisionableDB(ctx,
			name,
			bootstrapDBCfg,
			resourceNames.IDPDBUserName,
			resourceNames.IDPDBName,
			tenantdb.Schema,
			&tenantdb.SchemaBaseline)
	} else {
		// Already provisioned (at least partially)
		pdb = provisioning.NewProvisionableDBFromExistingConfigs(name,
			t.bootstrapDBCfg,
			&currTI.TenantDBConfig,
			t.overrideBootstrapDBCfg,
			tenantdb.Schema,
			&tenantdb.SchemaBaseline)
	}
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return pdb, nil
}

func (t *tenantDB) provision(ctx context.Context) error {
	return uctrace.Wrap0(ctx, tracer, "ProvisionTenantDB", true, func(ctx context.Context) error {
		uclog.Infof(ctx, "Start provisioning Tenant DB (tenant id: %s)...", t.tenantID)

		if t.tenantInternal == nil {
			return ucerr.Errorf("tenantDB.Provision can't be called without tenantInternal set")
		}

		pdb, err := t.initProvisionableDB(ctx, t.tenantInternal)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if err := pdb.Provision(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to provision Tenant DB (tenant id: %s): %v", t.tenantID, err)
			return ucerr.Wrap(err)
		}
		t.TenantDB = pdb.DB

		t.tenantInternal.TenantDBConfig = *pdb.ProvisionedDBCfg

		uclog.Infof(ctx, "Successfully provisioned Tenant DB (tenant id: %s)!", t.tenantID)
		return nil
	})
}

// Validate ensures the object is valid
func (t *tenantDB) Validate(ctx context.Context) error {
	currTI, err := t.companyConfigStorage.GetTenantInternal(ctx, t.tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	pdb, err := t.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := pdb.Validate(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (t *tenantDB) cleanup(ctx context.Context) error {
	uclog.Debugf(ctx, "Cleaning up Tenant DB (tenant id: %s)", t.tenantID)
	currTI, err := t.companyConfigStorage.GetTenantInternal(ctx, t.tenantID)
	if err != nil {
		uclog.Debugf(ctx, "Failed to load TenantInternal (id: %s) for Tenant DB cleanup: %v", t.tenantID, err)
		return ucerr.Wrap(err)
	}

	pdb, err := t.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = pdb.Cleanup(ctx)
	if err != nil {
		uclog.Debugf(ctx, "Failed to cleanup Tenant DB (tenant id: %s): %v", t.tenantID, err)
		return ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "Successfully cleaned up Tenant DB (tenant id: %s)", t.tenantID)
	return nil
}

// nuke will hard-delete all resources.
func (t *tenantDB) nuke(ctx context.Context) error {
	uclog.Debugf(ctx, "Nuking Tenant DB (tenant id: %s)", t.tenantID)
	currTI, err := t.companyConfigStorage.GetTenantInternal(ctx, t.tenantID)
	if err != nil {
		uclog.Debugf(ctx, "Failed to load TenantInternal (id: %s) for Tenant DB cleanup: %v", t.tenantID, err)
		return ucerr.Wrap(err)
	}

	pdb, err := t.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = pdb.Nuke(ctx)
	if err != nil {
		uclog.Debugf(ctx, "Failed to nuke Tenant DB (tenant id: %s): %v", t.tenantID, err)
		return ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "Successfully nuked Tenant DB (tenant id: %s)", t.tenantID)
	return nil
}
