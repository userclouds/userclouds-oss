// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t CompanyType) MarshalText() ([]byte, error) {
	switch t {
	case CompanyTypeCustomer:
		return []byte("customer"), nil
	case CompanyTypeDemo:
		return []byte("demo"), nil
	case CompanyTypeInternal:
		return []byte("internal"), nil
	case CompanyTypeProspect:
		return []byte("prospect"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown CompanyType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *CompanyType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "customer":
		*t = CompanyTypeCustomer
	case "demo":
		*t = CompanyTypeDemo
	case "internal":
		*t = CompanyTypeInternal
	case "prospect":
		*t = CompanyTypeProspect
	default:
		return ucerr.Friendlyf(nil, "unknown CompanyType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *CompanyType) Validate() error {
	switch *t {
	case CompanyTypeCustomer:
		return nil
	case CompanyTypeDemo:
		return nil
	case CompanyTypeInternal:
		return nil
	case CompanyTypeProspect:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown CompanyType value '%s'", *t)
	}
}

// Enum implements Enum
func (t CompanyType) Enum() []any {
	return []any{
		"customer",
		"demo",
		"internal",
		"prospect",
	}
}

// AllCompanyTypes is a slice of all CompanyType values
var AllCompanyTypes = []CompanyType{
	CompanyTypeCustomer,
	CompanyTypeDemo,
	CompanyTypeInternal,
	CompanyTypeProspect,
}
