package pageparameters

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/pagetype"
)

func validatePageTypeParameters() error {
	// ensure that there is no overlap between parameters associated with all pages
	// and with each page type

	for _, pt := range pagetype.Types() {
		if pt == pagetype.EveryPage {
			continue
		}

		for _, pn := range pt.ParameterNames() {
			if pagetype.EveryPage.SupportsParameterName(pn) {
				return ucerr.Errorf("param '%s' associated with '%s' and '%s'", pn, pagetype.EveryPage, pt)
			}
		}
	}

	return nil
}

func init() {
	if err := validatePageTypeParameters(); err != nil {
		panic(err)
	}
}
