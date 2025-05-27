package user

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantdb"
)

// ColumnCleaner is used to clean up user columns
type ColumnCleaner struct {
	companyStorage *companyconfig.Storage
	ctx            context.Context
}

// NewColumnCleaner creates a new cleaner
func NewColumnCleaner(ctx context.Context, s *companyconfig.Storage) *ColumnCleaner {
	return &ColumnCleaner{
		ctx:            ctx,
		companyStorage: s,
	}
}

func (c ColumnCleaner) getRunID() string {
	runID := time.Now().UTC().Format(time.RFC3339)
	uclog.Debugf(c.ctx, "starting runID %s", runID)
	return runID
}

// CleanUpAllTenants cleans up the user columns for all tenants
func (c ColumnCleaner) CleanUpAllTenants() {
	if !cmdline.Confirm(c.ctx, "clean up user columns for all tenants for %v environment? [yN] ", universe.Current()) {
		return
	}

	runID := c.getRunID()

	pager, err := companyconfig.NewTenantInternalPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
	)

	if err != nil {
		uclog.Fatalf(c.ctx, "could not apply pagination options: %v", err)
	}

	for {
		tis, respFields, err := c.companyStorage.ListTenantInternalsPaginated(
			c.ctx,
			*pager,
		)
		if err != nil {
			uclog.Fatalf(c.ctx, "could not list tenant internals: %v", err)
		}

		for _, ti := range tis {
			if c.isCleaned(ti.ID) {
				uclog.Debugf(c.ctx, "skipping tenant '%v' because it's already been cleaned", ti.ID)
				continue
			}

			if cmdline.Confirm(c.ctx, "clean up tenant '%v'? [yN]", ti.ID) {
				c.cleanUpTenant(&ti)
				c.setCleaned(ti.ID, runID)
			}

		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
}

func (c ColumnCleaner) cleanUpTenant(ti *companyconfig.TenantInternal) {
	tenantDB, err := tenantdb.ConnectWithConfig(c.ctx, &ti.TenantDBConfig)
	if err != nil {
		uclog.Fatalf(c.ctx, "could not connect to tenant '%v' db: %v", ti.ID, err)
	}
	configStorage := storage.New(c.ctx, tenantDB, ti.ID, nil)
	if err := configStorage.CleanUpUserColumns(c.ctx); err != nil {
		uclog.Fatalf(c.ctx, "could not clean up user columns for tenant '%v': %v", ti.ID, err)
	}
}

func (c ColumnCleaner) isCleaned(tenantID uuid.UUID) bool {
	isCleaned, err := c.companyStorage.AreTenantUserColumnsCleaned(c.ctx, tenantID)
	if err != nil {
		uclog.Fatalf(c.ctx, "could not check whether tenant '%v' is cleaned: %v", tenantID, err)
	}

	return isCleaned
}

func (c ColumnCleaner) setCleaned(tenantID uuid.UUID, runID string) {
	if err := c.companyStorage.SetTenantUserColumnsCleaned(c.ctx, tenantID, runID); err != nil {
		uclog.Fatalf(c.ctx, "could not mark tenant '%v' cleaned: %v", tenantID, err)
	}
}
