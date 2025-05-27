package api

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
)

// Parameter represents the current and default values of a parameter as returned to the editor.
// If the parameter has not been customized, the current value will be equal to the default value.
type Parameter struct {
	Name         param.Name `json:"parameter_name"`
	CurrentValue string     `json:"current_value"`
	DefaultValue string     `json:"default_value"`
}

//go:generate genvalidate Parameter

// ParameterByName is a map from parameter name to parameter
type ParameterByName map[param.Name]Parameter

// ParameterByNameByPageType is a map from page type to ParameterByName
type ParameterByNameByPageType map[pagetype.Type]ParameterByName

// AppPageParametersResponse represents the set of current and default parameter values, by page type,
// for a given tenant and app id. This is returned to the editor on a GET request or after a successful
// PUT request that has applied changes requested by the editor.
type AppPageParametersResponse struct {
	TenantID           uuid.UUID                 `json:"tenant_id" validate:"notnil"`
	AppID              uuid.UUID                 `json:"app_id" validate:"notnil"`
	PageTypeParameters ParameterByNameByPageType `json:"page_type_parameters" validate:"skip"`
}

//go:generate genvalidate AppPageParametersResponse

func (r *AppPageParametersResponse) extraValidate() error {
	if numPageTypes := len(pagetype.Types()); len(r.PageTypeParameters) != numPageTypes {
		return ucerr.Friendlyf(nil, "number of page types (%d) does not match expected (%d)", len(r.PageTypeParameters), numPageTypes)
	}

	for pt, paramsByName := range r.PageTypeParameters {
		if err := pt.Validate(); err != nil {
			return ucerr.Friendlyf(err, "page type '%s' is invalid", pt)
		}

		for pn, p := range paramsByName {
			if err := p.Validate(); err != nil {
				return ucerr.Friendlyf(err, "invalid parameter '%s' for page type '%s'", p, pt)
			}
			if p.Name != pn {
				return ucerr.Friendlyf(nil, "parameter '%s' does not match name '%s' for page type '%s'", p, pn, pt)
			}
			if !pt.SupportsParameterName(p.Name) {
				return ucerr.Friendlyf(nil, "parameter '%s' is not supported by page type '%s'", p, pt)
			}
		}
	}

	return nil
}

// ParameterChange represents a change to a parameter as requested by the editor. If the editor intends to remove the
// customization, the new value can be set to the empty string. If the new value is equal to the default value for the
// parameter, the request will also be treated as a deletion request.
type ParameterChange struct {
	Name     param.Name `json:"parameter_name"`
	NewValue string     `json:"new_value"`
}

//go:generate genvalidate ParameterChange

// ParameterChangeByName is a map from parameter name to parameter change
type ParameterChangeByName map[param.Name]ParameterChange

// ParameterChangeByNameByPageType is a map from page type to ParameterChangeByName
type ParameterChangeByNameByPageType map[pagetype.Type]ParameterChangeByName

// SaveAppPageParametersRequest represents the set of requested parameter changes, by page type. The editor will
// pass this data structure as the request object for a specified tenant and app.
type SaveAppPageParametersRequest struct {
	PageTypeParameterChanges ParameterChangeByNameByPageType `json:"page_type_parameter_changes"`
}

// Validate implements the Validatable interface and verifies the request is valid
func (r *SaveAppPageParametersRequest) Validate() error {
	if len(r.PageTypeParameterChanges) == 0 {
		return ucerr.Friendlyf(nil, "request has no changes")
	}

	for pt, changesByName := range r.PageTypeParameterChanges {
		if err := pt.Validate(); err != nil {
			return ucerr.Friendlyf(err, "page type '%s' is invalid", pt)
		}
		if len(changesByName) == 0 {
			return ucerr.Friendlyf(nil, "request has no changes for specified page type '%s'", pt)
		}

		for pn, pc := range changesByName {
			if err := pc.Validate(); err != nil {
				return ucerr.Friendlyf(err, "invalid parameter '%s' for page type '%s'", pc, pt)
			}
			if pc.Name != pn {
				return ucerr.Friendlyf(nil, "parameter '%s' does not match name '%s' for page type '%s'", pc, pn, pt)
			}
			if !pt.SupportsParameterName(pc.Name) {
				return ucerr.Friendlyf(nil, "parameter '%s' is not supported by page type '%s'", pc, pt)
			}
		}
	}

	return nil
}
