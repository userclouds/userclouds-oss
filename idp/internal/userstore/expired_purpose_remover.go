package userstore

import (
	"context"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
)

// RemoveExpiredPurposes removes any expired purposes for a user
func RemoveExpiredPurposes(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	userID uuid.UUID,
) (int, error) {
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)
	umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}
	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	referenceTime := time.Now().UTC()
	baseUser, liveValues, softDeletedValues, reg, code, err := umrs.GetAllUserValues(ctx, cm, dtm, userID, false)
	if err != nil {
		return code, ucerr.Wrap(err)
	}

	regionDB, ok := ts.UserRegionDbMap[reg]
	if !ok {
		return http.StatusInternalServerError, ucerr.Errorf("db for region %s not found", reg)
	}
	us := storage.NewUserStorage(ctx, regionDB, reg, ts.ID)
	uvu, err := newUserValueUpdater(ctx, s, us, searchUpdateConfig, referenceTime, ts.ID, uuid.Nil)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}
	uvu.setUser(baseUser)
	if err := uvu.removeExpiredPurposes(ctx, liveValues, softDeletedValues); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if uvu.hasChanges() {
		if err := uvu.saveChanges(ctx); err != nil {
			if ucdb.IsUniqueViolation(err) {
				return http.StatusConflict, ucerr.Wrap(err)
			}
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}
	}

	return http.StatusOK, nil
}
