package pageparameters

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
)

// GetParameterValues will return the current and default values for a given page type and parameter name, given the provided
// parameter getters and applying the passed in client data. If there is no customized value, the current value and default
// value will be the same.
func GetParameterValues(pt pagetype.Type,
	pn param.Name,
	getters []ParameterGetter,
	cd param.ClientData) (currentValue string, defaultValue string, err error) {
	for _, getter := range getters {
		if p, found := getter(pt, pn); found {
			p, err = p.ApplyClientData(cd)
			if err != nil {
				return "", "", ucerr.Wrap(err)
			}
			if err = p.Validate(); err != nil {
				return "", "", ucerr.Wrap(err)
			}

			defaultValue = currentValue
			currentValue = p.Value
		}
	}
	if len(defaultValue) == 0 {
		defaultValue = currentValue
	}

	return currentValue, defaultValue, nil
}

// GetRenderParameters will return the appropriate rendering parameters for a specified page type and set of parameter
// names, using the passed in ParameterGetter function and applying the passed in client data to the retrieved parameters.
// These parameters will include both page type specific parameters and parameters that apply for all page types.
func GetRenderParameters(pt pagetype.Type,
	requestedNames []param.Name,
	getter ParameterGetter,
	cd param.ClientData) (params []param.Parameter, err error) {
	nameSet := map[param.Name]bool{}
	params = []param.Parameter{}
	for _, name := range requestedNames {
		// check for duplicate requests
		if _, found := nameSet[name]; found {
			return params, ucerr.Errorf("duplicate parameter name '%s' in request", name)
		}
		nameSet[name] = true

		// find the appropriate parameter
		p, found := getter(pt, name)
		if !found {
			return params, ucerr.Errorf("could not find requested parameter name '%s' for page type '%s'", name, pt)
		}

		// apply the client data to the parameter
		p, err = p.ApplyClientData(cd)
		if err != nil {
			return params, ucerr.Wrap(err)
		}

		// make sure the resulting parameter value is valid
		if err := p.Validate(); err != nil {
			return params, ucerr.Wrap(err)
		}

		params = append(params, p)
	}

	return params, nil
}
