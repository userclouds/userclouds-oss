package api

import (
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/gatekeeper"
	pageparams "userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
)

func makeAppPageParametersResponse(tenantID uuid.UUID, tam tenantAppManager) (AppPageParametersResponse, error) {
	appr := AppPageParametersResponse{TenantID: tenantID, AppID: tam.app.ID, PageTypeParameters: ParameterByNameByPageType{}}

	getters, cd := tenantplex.MakeParameterRetrievalTools(tam.tenant, tam.app)
	for _, pt := range pagetype.Types() {
		paramsByName := ParameterByName{}
		for _, pn := range pt.ParameterNames() {
			if parameter.IsParameterWhitelistOnly(pn) && !gatekeeper.IsAppOnWhitelist(tam.app.ID, string(pt)) {
				continue
			}
			currentValue, defaultValue, err := pageparams.GetParameterValues(pt, pn, getters, cd)
			if err != nil {
				return appr, ucerr.Wrap(validationError{err})
			}
			paramsByName[pn] = Parameter{Name: pn, CurrentValue: currentValue, DefaultValue: defaultValue}
		}
		appr.PageTypeParameters[pt] = paramsByName
	}

	if err := appr.Validate(); err != nil {
		return appr, ucerr.Wrap(validationError{err})
	}

	return appr, nil
}

func (h *handler) getAppPageParameters(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, appID uuid.UUID) {
	ctx := r.Context()

	tam := newTenantAppManager(h, r)
	if err := tam.loadTenantApp(tenantID, appID); err != nil {
		switch {
		case errors.As(err, &tam.errors.badAppID):
			jsonapi.MarshalErrorL(ctx, w, err, "AppIDInvalid", jsonapi.Code(http.StatusBadRequest))
		case errors.As(err, &tam.errors.badTenantID):
			jsonapi.MarshalErrorL(ctx, w, err, "TenantIDInvalid", jsonapi.Code(http.StatusNotFound))
		case errors.As(err, &tam.errors.forbidden):
			jsonapi.MarshalErrorL(ctx, w, err, "OperationForbidden", jsonapi.Code(http.StatusForbidden))
		case errors.As(err, &tam.errors.sql):
			jsonapi.MarshalErrorL(ctx, w, err, "SQLError", jsonapi.Code(http.StatusInternalServerError))
		default:
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected error: '%w'", err), "UnexpectedLoadError", jsonapi.Code(http.StatusInternalServerError))
		}
		return
	}

	appr, err := makeAppPageParametersResponse(tenantID, tam)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("response is invalid: '%w'", err), "GetResponseInvalid", jsonapi.Code(http.StatusInternalServerError))
		return
	}

	jsonapi.Marshal(w, appr)
}

func (h *handler) saveAppPageParameters(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, appID uuid.UUID) {
	ctx := r.Context()

	var sapr SaveAppPageParametersRequest
	if err := jsonapi.Unmarshal(r, &sapr); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "BadRequest")
		return
	}

	tam := newTenantAppManager(h, r)
	if err := tam.loadTenantApp(tenantID, appID); err != nil {
		switch {
		case errors.As(err, &tam.errors.badAppID):
			jsonapi.MarshalErrorL(ctx, w, err, "AppIDInvalid", jsonapi.Code(http.StatusBadRequest))
		case errors.As(err, &tam.errors.badTenantID):
			jsonapi.MarshalErrorL(ctx, w, err, "TenantIDInvalid", jsonapi.Code(http.StatusNotFound))
		case errors.As(err, &tam.errors.forbidden):
			jsonapi.MarshalErrorL(ctx, w, err, "OperationForbidden", jsonapi.Code(http.StatusForbidden))
		case errors.As(err, &tam.errors.sql):
			jsonapi.MarshalErrorL(ctx, w, err, "SQLError", jsonapi.Code(http.StatusInternalServerError))
		default:
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected error: '%w'", err), "UnexpectedLoadError", jsonapi.Code(http.StatusInternalServerError))
		}
		return
	}

	getters, cd := tenantplex.MakeParameterRetrievalTools(tam.tenant, tam.app)
	for pt, changesByName := range sapr.PageTypeParameterChanges {
		for pn, pc := range changesByName {
			if parameter.IsParameterWhitelistOnly(pn) && !gatekeeper.IsAppOnWhitelist(tam.app.ID, string(pt)) {
				continue
			}
			currentValue, defaultValue, err := pageparams.GetParameterValues(pt, pn, getters, cd)
			if err != nil {
				jsonapi.MarshalErrorL(ctx, w, err, "GetParameterValuesError", jsonapi.Code(http.StatusInternalServerError))
				return
			}

			if pc.NewValue != currentValue {
				if pc.NewValue == defaultValue {
					tam.app.DeletePageParameter(pt, pn)
				} else {
					tam.app.SetPageParameter(pt, pn, pc.NewValue)
				}
			}
		}
	}

	if err := tam.saveTenant(); err != nil {
		uclog.Errorf(ctx, "error saving tenant: %v", err)
		switch {
		case errors.As(err, &tam.errors.internal):
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("internal server error: '%w'", err), "TenantNotLoaded", jsonapi.Code(http.StatusInternalServerError))
		case errors.As(err, &tam.errors.sql):
			jsonapi.MarshalErrorL(ctx, w, err, "SQLError", jsonapi.Code(http.StatusInternalServerError))
		case errors.As(err, &tam.errors.validation):
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("requested changes invalid '%w'", err), "PageParameterChangesInvalid", jsonapi.Code(http.StatusBadRequest))
		default:
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected error: '%w'", err), "UnexpectedSaveError", jsonapi.Code(http.StatusInternalServerError))
		}
		return
	}

	appr, err := makeAppPageParametersResponse(tenantID, tam)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("response is invalid: '%w'", err), "PutResponseInvalid", jsonapi.Code(http.StatusInternalServerError))
		return
	}

	jsonapi.Marshal(w, appr)
}
