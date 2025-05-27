package tenant

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/provisioning"
	logserver_config "userclouds.com/logserver/config"
)

type logDB struct {
	tenantID             uuid.UUID
	companyConfigStorage *companyconfig.Storage
	// bootstrapDBCfg is any valid DB connection which can be used to create
	// new databases in the same cluster.
	bootstrapDBCfg *ucdb.Config

	// managed centrally at from tenant so that we don't save intermediate/invalid states
	tenantInternal *companyconfig.TenantInternal
}

func (l *logDB) initProvisionableDB(ctx context.Context, currTI *companyconfig.TenantInternal) (*provisioning.ProvisionableDB, error) {
	name := fmt.Sprintf("Log DB (tenant id: %s)", l.tenantID)
	var pdb *provisioning.ProvisionableDB
	var err error
	if currTI.LogConfig.Validate() != nil {
		// Unprovisioned (or incompletely provisioned tenant).
		// NOTE: If the config fails to validate, we assume it must be totally re-created.
		resourceNames := logserver_config.NewTenantBaseResourceNames(l.tenantID)
		pdb, err = provisioning.NewProvisionableDB(
			ctx,
			name,
			l.bootstrapDBCfg,
			resourceNames.StatusDBUserName,
			resourceNames.StatusDBName,
			migrate.Schema{Migrations: logdb.GetMigrations(), CreateStatements: nil}, // TODO
			nil) // TODO
	} else {
		// Already provisioned (at least partially)
		pdb = provisioning.NewProvisionableDBFromExistingConfigs(name,
			l.bootstrapDBCfg,
			&currTI.LogConfig.LogDB,
			nil, // no overridden DBs in logdb
			migrate.Schema{Migrations: logdb.GetMigrations(), CreateStatements: nil}, // TODO
			nil) // TODO
	}
	// technically pdb should be nil if err isn't, but three extra lines (+comment) seem safer
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return pdb, nil
}

func (l *logDB) provision(ctx context.Context) error {
	return uctrace.Wrap0(ctx, tracer, "ProvisionLogDB", true, func(ctx context.Context) error {
		uclog.Debugf(ctx, "Start provisioning Log DB (tenant id: %s)...", l.tenantID)

		if l.tenantInternal == nil {
			return ucerr.Errorf("logDB.Provision can't be called without tenantInternal set")
		}

		pdb, err := l.initProvisionableDB(ctx, l.tenantInternal)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if err := pdb.Provision(ctx); err != nil {
			uclog.Debugf(ctx, "Failed to provision Log DB (tenant id: %s): %v", l.tenantID, err)
			return ucerr.Wrap(err)
		}

		l.tenantInternal.LogConfig.LogDB = *pdb.ProvisionedDBCfg

		uclog.Debugf(ctx, "Successfully provisioned Log DB (tenant id: %s)!", l.tenantID)
		return nil
	})
}

// Validate makes sure the object is valid
func (l *logDB) Validate(ctx context.Context) error {
	currTI, err := l.companyConfigStorage.GetTenantInternal(ctx, l.tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	/* TODO - this currently busted

	pdb, err := l.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := pdb.Validate(ctx); err != nil {
		return ucerr.Wrap(err)
	}*/

	ldb, err := ucdb.New(ctx, &currTI.LogConfig.LogDB, migrate.SchemaValidator(logdb.Schema))
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err = ldb.Close(ctx); err != nil {
		uclog.Errorf(ctx, "Failed to close the connection to logdb %v after validation", err)
	}

	return nil
}

func (l *logDB) cleanup(ctx context.Context) error {
	uclog.Debugf(ctx, "Cleaning up Log DB (tenant id: %s)", l.tenantID)
	currTI, err := l.companyConfigStorage.GetTenantInternal(ctx, l.tenantID)
	if err != nil {
		uclog.Debugf(ctx, "Failed to load TenantInternal (id: %s) for Log DB cleanup: %v", l.tenantID, err)
		return ucerr.Wrap(err)
	}

	pdb, err := l.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = pdb.Cleanup(ctx)
	if err != nil {
		uclog.Debugf(ctx, "Failed to cleanup Log DB (tenant id: %s): %v", l.tenantID, err)
		return ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "Successfully cleaned up Log DB (tenant id: %s)", l.tenantID)
	return nil
}

// nuke will hard-delete all resources.
func (l *logDB) nuke(ctx context.Context) error {
	uclog.Debugf(ctx, "Nuking Log DB (tenant id: %s)", l.tenantID)
	currTI, err := l.companyConfigStorage.GetTenantInternal(ctx, l.tenantID)
	if err != nil {
		uclog.Debugf(ctx, "Failed to load TenantInternal (id: %s) for Log DB cleanup: %v", l.tenantID, err)
		return ucerr.Wrap(err)
	}

	pdb, err := l.initProvisionableDB(ctx, currTI)
	if err != nil {
		return ucerr.Wrap(err)
	}

	err = pdb.Nuke(ctx)
	if err != nil {
		uclog.Debugf(ctx, "Failed to nuke Log DB (tenant id: %s): %v", l.tenantID, err)
		return ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "Successfully nuked Log DB (tenant id: %s)", l.tenantID)
	return nil
}
