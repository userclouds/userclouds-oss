package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

// NewTenantAppMessageElementsFromTenantConfig populates a TenantAppMessageElements struct from
// an existing TenantConfig
func NewTenantAppMessageElementsFromTenantConfig(tenantID uuid.UUID, tc tenantplex.TenantConfig, messageTypes []message.MessageType) TenantAppMessageElements {
	tame := TenantAppMessageElements{TenantID: tenantID}
	for _, a := range tc.PlexMap.Apps {
		ame := AppMessageElements{
			AppID:                      a.ID,
			MessageTypeMessageElements: map[message.MessageType]MessageTypeMessageElements{}}
		for _, mt := range messageTypes {
			defaultElementGetter := message.MakeElementGetter(mt)
			appElementGetter := a.MakeElementGetter(mt)
			mtme := MessageTypeMessageElements{
				Type:              mt,
				MessageElements:   map[message.ElementType]MessageElement{},
				MessageParameters: messageParametersForType[mt]}
			for _, elt := range message.ElementTypes(mt) {
				me := MessageElement{Type: elt}
				me.DefaultValue = defaultElementGetter(elt)
				if appValue := appElementGetter(elt); appValue != me.DefaultValue {
					me.CustomValue = appValue
				}
				mtme.MessageElements[elt] = me
			}
			ame.MessageTypeMessageElements[mt] = mtme
		}
		tame.AppMessageElements = append(tame.AppMessageElements, ame)
	}
	return tame
}

// GetTenantAppMessageElementsResponse is the response struct for retrieving the message elements for a tenant
type GetTenantAppMessageElementsResponse struct {
	TenantAppMessageElements TenantAppMessageElements `json:"tenant_app_message_elements"`
}

func (h *handler) getTenantAppEmailElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.getTenantAppMessageElements(w, r, tenantID, message.EmailMessageTypes())
}

func (h *handler) getTenantAppSMSElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.getTenantAppMessageElements(w, r, tenantID, message.SMSMessageTypes())
}

func (h *handler) getTenantAppMessageElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, messageTypes []message.MessageType) {
	ctx := r.Context()

	// load the associated tenant config
	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		} else {
			jsonapi.MarshalError(ctx, w, err)
		}
		return
	}

	tame := NewTenantAppMessageElementsFromTenantConfig(tenantID, tp.PlexConfig, messageTypes)
	if err := tame.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, GetTenantAppMessageElementsResponse{TenantAppMessageElements: tame})
}

// SaveTenantAppMessageElementsRequest is the request struct for saving the message elements for a tenant, app, and email type
type SaveTenantAppMessageElementsRequest struct {
	ModifiedMessageTypeMessageElements ModifiedMessageTypeMessageElements `json:"modified_message_type_message_elements"`
}

//go:generate genvalidate SaveTenantAppMessageElementsRequest

// SaveTenantAppMessageElementsResponse is the response struct for saving the message elements for a tenant
type SaveTenantAppMessageElementsResponse struct {
	TenantAppMessageElements TenantAppMessageElements `json:"tenant_app_message_elements"`
}

func (h *handler) saveTenantAppEmailElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.saveTenantAppMessageElements(w, r, tenantID, message.EmailMessageTypes())
}

func (h *handler) saveTenantAppSMSElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.saveTenantAppMessageElements(w, r, tenantID, message.SMSMessageTypes())
}

func (h *handler) saveTenantAppMessageElements(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, messageTypes []message.MessageType) {
	ctx := r.Context()

	var stamer SaveTenantAppMessageElementsRequest
	if err := jsonapi.Unmarshal(r, &stamer); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// validate request, confirm permissions, and load tenant
	mmtme := stamer.ModifiedMessageTypeMessageElements

	if tenantID != mmtme.TenantID {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("tenant id '%v' does not match request tenant id '%v'", tenantID, mmtme.TenantID), jsonapi.Code(http.StatusBadRequest))
		return
	}

	tam := newTenantAppManager(h, r)
	if err := tam.loadTenantApp(tenantID, mmtme.AppID); err != nil {
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

	// apply all elements, removing if equal to default
	defaultGetter := message.MakeElementGetter(mmtme.MessageType)
	for elt, customValue := range mmtme.MessageElements {
		if defaultValue := defaultGetter(elt); customValue == defaultValue {
			customValue = ""
		}
		tam.app.CustomizeMessageElement(mmtme.MessageType, elt, customValue)
	}

	// save changes
	if err := tam.saveTenant(); err != nil {
		switch {
		case errors.As(err, &tam.errors.internal):
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("internal server error: '%w'", err), "TenantNotLoaded", jsonapi.Code(http.StatusInternalServerError))
		case errors.As(err, &tam.errors.sql):
			jsonapi.MarshalErrorL(ctx, w, err, "SQLError", jsonapi.Code(http.StatusInternalServerError))
		case errors.As(err, &tam.errors.validation):
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("requested changes invalid '%w'", err), "TenantAppEmailElementsChangesInvalid", jsonapi.Code(http.StatusBadRequest))
		default:
			jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("unexpected error: '%w'", err), "UnexpectedSaveError", jsonapi.Code(http.StatusInternalServerError))
		}
		return
	}

	// generate and validate new state
	tame := NewTenantAppMessageElementsFromTenantConfig(tenantID, tam.tp.PlexConfig, messageTypes)
	if err := tame.Validate(); err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "UnexpectedResponseError", jsonapi.Code(http.StatusInternalServerError))
		return
	}

	jsonapi.Marshal(w, SaveTenantAppMessageElementsResponse{TenantAppMessageElements: tame})
}
