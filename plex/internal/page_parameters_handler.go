package internal

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/custompages"
	"userclouds.com/internal/gatekeeper"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/tenantconfig"
)

// PageParametersRequest is the request struct used by the front end to
// retrieve the specified set of parameters for a given session and page type
type PageParametersRequest struct {
	SessionID      uuid.UUID        `json:"session_id" validate:"notnil"`
	PageType       pagetype.Type    `json:"page_type"`
	ParameterNames []parameter.Name `json:"parameter_names" validate:"skip"`
}

//go:generate genvalidate PageParametersRequest

func (r *PageParametersRequest) extraValidate() error {
	names := map[parameter.Name]bool{}
	for _, n := range r.ParameterNames {
		if _, found := names[n]; found {
			return ucerr.Errorf("ParameterNames has duplicate name '%s'", n)
		}
		if err := n.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		names[n] = true
	}

	return nil
}

// PageParametersResponse is the response struct returned in response to a request
// for the relevant page parameters for a given session and page type
type PageParametersResponse struct {
	ClientID           string                `json:"client_id" validate:"notempty"`
	PageParameters     []parameter.Parameter `json:"page_parameters,omitempty" validate:"skip"`
	PageSourceOverride string                `json:"page_source_override,omitempty" validate:"skip"`
}

//go:generate genvalidate PageParametersResponse

func getSessionInfo(ctx context.Context, w http.ResponseWriter, sessionID uuid.UUID) (
	clientID string,
	hasValidInvite bool,
	tenant *tenantplex.TenantConfig,
	app *tenantplex.App,
	ok bool) {
	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		ok = false
		uchttp.ErrorL(ctx, w, ucerr.Errorf("can't find login session for id '%s': %w", sessionID, err), http.StatusBadRequest, "BadSession")
		return
	}
	clientID = session.ClientID

	hasValidInvite = false
	if _, err := otp.HasUnusedInvite(ctx, s, session); err != nil {
		if errors.Is(err, otp.ErrNoInviteAssociatedWithSession) {
			// No invite, ignore
		} else {
			// Invite already used or failed to get invite; don't fail the call for this
			uclog.Debugf(ctx, "error checking session for valid invite: %v", err)
		}
	} else {
		hasValidInvite = true
	}

	tc := tenantconfig.MustGet(ctx)
	tenant = &tc
	app, _, err = tenant.PlexMap.FindAppForClientID(clientID)
	if err != nil {
		ok = false
		uchttp.ErrorL(ctx, w, ucerr.Errorf("can't find plex app for client id '%s': %w", clientID, err), http.StatusBadRequest, "BadClientID")
		return
	}

	ok = true
	return
}

func (h *handler) pageParametersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var request PageParametersRequest
	if err := jsonapi.Unmarshal(r, &request); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	clientID, hasValidInvite, tc, app, ok := getSessionInfo(ctx, w, request.SessionID)
	if !ok {
		return
	}

	var parameterNames []parameter.Name

	if gatekeeper.IsAppOnWhitelist(app.ID, string(request.PageType)) {
		tenant := multitenant.MustGetTenantState(ctx)
		s := custompages.NewStorage(tenant.TenantDB)
		customPage, err := s.GetCustomPageForAppPage(ctx, app.ID, string(request.PageType))
		if err == nil {
			// we have a custom page for this app
			response := PageParametersResponse{ClientID: clientID, PageSourceOverride: customPage.PageSource}
			jsonapi.Marshal(w, response)
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			uchttp.ErrorL(ctx, w, ucerr.Errorf("could not retrieve custom page for app '%s': %w", app.ID, err), http.StatusBadRequest, "GetCustomPageFailed")
			return
		}
		parameterNames = request.ParameterNames
	} else {

		// filter out whitelist-only parameters
		parameterNames = []parameter.Name{}
		for _, name := range request.ParameterNames {
			if parameter.IsParameterWhitelistOnly(name) {
				continue
			}
			parameterNames = append(parameterNames, name)
		}
	}

	pageParameters, err :=
		pageparameters.GetRenderParameters(request.PageType,
			parameterNames,
			tenantplex.MakeRenderParameterGetter(tc, app),
			tenantplex.MakeParameterClientData(tc, app))
	if err != nil {
		uchttp.ErrorL(ctx, w, ucerr.Errorf("could not retrieve requested parameter names: %w", err), http.StatusBadRequest, "GetRenderParametersFailed")
		return
	}

	if hasValidInvite {
		// we always allow creation if there is a valid invite
		for i, p := range pageParameters {
			if p.Name == parameter.AllowCreation {
				p.Value = "true"
				pageParameters[i] = p
				break
			}
		}
	}

	response := PageParametersResponse{ClientID: clientID, PageParameters: pageParameters}
	if err := response.Validate(); err != nil {
		uchttp.ErrorL(ctx, w, ucerr.Errorf("response is invalid: %w", err), http.StatusBadRequest, "ResponseInvalid")
		return
	}

	jsonapi.Marshal(w, response)
}
