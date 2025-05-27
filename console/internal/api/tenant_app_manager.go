package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

type badAppIDError struct {
	error
}

type badTenantIDError struct {
	error
}

type forbiddenError struct {
	error
}

type internalError struct {
	error
}

type sqlError struct {
	error
}

type validationError struct {
	error
}

type tenantAppManager struct {
	h      *handler
	r      *http.Request
	tp     *tenantplex.TenantPlex
	tenant *tenantplex.TenantConfig
	app    *tenantplex.App

	errors struct {
		badAppID    badAppIDError
		badTenantID badTenantIDError
		forbidden   forbiddenError
		internal    internalError
		sql         sqlError
		validation  validationError
	}
}

func newTenantAppManager(h *handler, r *http.Request) tenantAppManager {
	return tenantAppManager{h: h, r: r}
}

func (tam *tenantAppManager) hasAdminPermissions(tenantID uuid.UUID) error {
	tenant, err := tam.h.storage.GetTenant(tam.r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(badTenantIDError{ucerr.Friendlyf(err, "tenant id '%v' not found", tenantID)})
		}

		return ucerr.Wrap(sqlError{ucerr.Friendlyf(err, "could not load tenant id '%v'", tenantID)})
	}

	if _, err := tam.h.ensureEmployeeAccessToTenant(tam.r, tenant); err != nil {
		return ucerr.Wrap(forbiddenError{ucerr.Friendlyf(err, "user does not have admin privileges for company '%v'", tenant.CompanyID)})
	}

	return nil
}

func (tam *tenantAppManager) loadTenantApp(tenantID uuid.UUID, appID uuid.UUID) error {
	ctx := tam.r.Context()
	if err := tam.hasAdminPermissions(tenantID); err != nil {
		return ucerr.Wrap(err)
	}

	tenantDB, err := tam.h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	mgr := manager.NewFromDB(tenantDB, tam.h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(badTenantIDError{ucerr.Friendlyf(err, "tenant id '%v' not found", tenantID)})
		}

		return ucerr.Wrap(sqlError{ucerr.Friendlyf(err, "could not load tenant id '%v'", tenantID)})
	}
	tam.tp = tp
	tam.tenant = &tp.PlexConfig

	for i, a := range tam.tenant.PlexMap.Apps {
		if appID == a.ID {
			tam.app = &tam.tenant.PlexMap.Apps[i]
			break
		}
	}
	if tam.app == nil {
		return ucerr.Wrap(badAppIDError{ucerr.Friendlyf(nil, "app id '%v' is not associated with tenant id '%v'", appID, tenantID)})
	}

	return nil
}

func (tam *tenantAppManager) saveTenant() error {
	ctx := tam.r.Context()
	if tam.tenant == nil {
		return ucerr.Wrap(internalError{ucerr.Friendlyf(nil, "tenant was not loaded before saving")})
	}

	if err := tam.tp.Validate(); err != nil {
		return ucerr.Wrap(validationError{ucerr.Friendlyf(err, "tenant is invalid")})
	}

	tenantDB, err := tam.h.tenantCache.GetTenantDB(ctx, tam.tp.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	mgr := manager.NewFromDB(tenantDB, tam.h.cacheConfig)
	if err := mgr.SaveTenantPlex(tam.r.Context(), tam.tp); err != nil {
		return ucerr.Wrap(sqlError{ucerr.Friendlyf(err, "could not save tenant")})
	}

	return nil
}
