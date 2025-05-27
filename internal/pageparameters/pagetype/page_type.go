package pagetype

import (
	"userclouds.com/infra/ucerr"
	param "userclouds.com/internal/pageparameters/parameter"
)

var pageTypeParameterNames = map[Type][]param.Name{}
var pageTypeParametersByName = map[Type]map[param.Name]param.Parameter{}
var pageTypes = []Type{}
var pageTypeTestParameters = map[Type][]param.Parameter{}

func registerPageParameters(pt Type, params []param.Parameter) error {
	if _, present := pageTypeParametersByName[pt]; present {
		return ucerr.Errorf("duplicate registration for page type '%s'", pt)
	}

	paramsByName := map[param.Name]param.Parameter{}
	paramNames := []param.Name{}
	testParams := []param.Parameter{}
	for _, p := range params {
		if _, found := paramsByName[p.Name]; found {
			return ucerr.Errorf("duplicate parameter '%s' for page type '%s'", p.Name, pt)
		}
		testParam, err := p.ApplyDefaultClientData()
		if err != nil {
			return ucerr.Errorf("could not apply default client data for parameter '%s'", p)
		}
		if err := testParam.Validate(); err != nil {
			return ucerr.Errorf("invalid parameter '%s'", p.String())
		}

		paramsByName[p.Name] = p
		paramNames = append(paramNames, p.Name)
		testParams = append(testParams, testParam)
	}
	pageTypeParameterNames[pt] = paramNames
	pageTypeParametersByName[pt] = paramsByName
	pageTypes = append(pageTypes, pt)
	pageTypeTestParameters[pt] = testParams

	return nil
}

// DefaultParameterGetter is a ParameterGetter that return default parameters that apply for
// the specified page
var DefaultParameterGetter = func(pt Type, pn param.Name) (param.Parameter, bool) {
	p, found := pageTypeParametersByName[pt][pn]
	return p, found
}

// DefaultRenderParameterGetter is a ParameterGetter that return default parameters that apply
// for the specified page or for all pages
var DefaultRenderParameterGetter = func(pt Type, pn param.Name) (param.Parameter, bool) {
	if p, found := DefaultParameterGetter(pt, pn); found {
		return p, true
	}
	return DefaultParameterGetter(EveryPage, pn)
}

// Types returns a copy of the slice of all registered page types
func Types() (types []Type) {
	return append(types, pageTypes...)
}

// ParameterNames returns a copy of the slice of all registered parameter names for the page type
func (pt Type) ParameterNames() (names []param.Name) {
	if foundNames, found := pageTypeParameterNames[pt]; found {
		names = append(names, foundNames...)
	}
	return
}

// RenderParameterNames returns a copy of the slice of all registered parameter names
// for a page type and for every page
func (pt Type) RenderParameterNames() (names []param.Name) {
	names = pt.ParameterNames()
	if pt != EveryPage {
		names = append(names, EveryPage.ParameterNames()...)
	}
	return
}

// SupportsParameterName returns true if the parameter name is associated with the page type
func (pt Type) SupportsParameterName(pn param.Name) bool {
	if parametersByName, found := pageTypeParametersByName[pt]; found {
		if _, found := parametersByName[pn]; found {
			return true
		}
	}

	return false
}

// TestParameters returns a map of parameter name to test parameter for the page type
func (pt Type) TestParameters() (paramsByName map[param.Name]param.Parameter) {
	paramsByName = map[param.Name]param.Parameter{}
	if params, found := pageTypeTestParameters[pt]; found {
		for _, p := range params {
			paramsByName[p.Name] = p
		}
	}
	return
}

// TestRenderParameters returns a map of parameter name to test render parameter for the page
// type and for every page
func (pt Type) TestRenderParameters() (paramsByName map[param.Name]param.Parameter) {
	paramsByName = map[param.Name]param.Parameter{}
	for _, t := range []Type{pt, EveryPage} {
		if params, found := pageTypeTestParameters[t]; found {
			for _, p := range params {
				paramsByName[p.Name] = p
			}
		}
	}
	return
}

// Validate implements the Validatable interface and verifies the page type is valid
func (pt Type) Validate() error {
	if _, found := pageTypeParametersByName[pt]; found {
		return nil
	}

	return ucerr.Errorf("invalid page type: %s", pt)
}
