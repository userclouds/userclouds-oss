package pageparameters

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
)

// ParameterGetter is a function that returns a parameter for a given page type and parameter name
type ParameterGetter func(pt pagetype.Type, pn param.Name) (param.Parameter, bool)

// ParameterByName is a mapping of parameter name to parameter
type ParameterByName map[param.Name]param.Parameter

// ParameterByNameByPageType is a mapping of page type to ParameterByName
type ParameterByNameByPageType map[pagetype.Type]ParameterByName

// Validate implements the Validatable interface and verifies the data structure is valid
func (pbnbpt ParameterByNameByPageType) Validate() error {
	for pt, params := range pbnbpt {
		if err := pt.Validate(); err != nil {
			return ucerr.Friendlyf(err, "invalid page type '%s' in  page parameters", pt)
		}
		for pn, p := range params {
			if err := p.ValidateDefault(); err != nil {
				return ucerr.Friendlyf(err, "invalid parameter '%s' for page type '%s' in  page parameters", p, pt)
			}
			if pn != p.Name {
				return ucerr.Friendlyf(nil, "parameter '%s' does not match name '%s' for page type '%s' in  page parameters", p, pn, pt)
			}
			if !pt.SupportsParameterName(pn) {
				return ucerr.Friendlyf(nil, "parameter '%s' is not supported by page type '%s' in  page parameters", p, pt)
			}
		}
	}

	return nil
}
