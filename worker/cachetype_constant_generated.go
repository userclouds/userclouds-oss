// NOTE: automatically generated file -- DO NOT EDIT

package worker

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t CacheType) MarshalText() ([]byte, error) {
	switch t {
	case CacheTypeAll:
		return []byte("all"), nil
	case CacheTypeAuthZ:
		return []byte("authz"), nil
	case CacheTypeCompanyConfig:
		return []byte("company_config"), nil
	case CacheTypePlex:
		return []byte("plex"), nil
	case CacheTypeUserStore:
		return []byte("userstore"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown CacheType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *CacheType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "all":
		*t = CacheTypeAll
	case "authz":
		*t = CacheTypeAuthZ
	case "company_config":
		*t = CacheTypeCompanyConfig
	case "plex":
		*t = CacheTypePlex
	case "userstore":
		*t = CacheTypeUserStore
	default:
		return ucerr.Friendlyf(nil, "unknown CacheType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *CacheType) Validate() error {
	switch *t {
	case CacheTypeAll:
		return nil
	case CacheTypeAuthZ:
		return nil
	case CacheTypeCompanyConfig:
		return nil
	case CacheTypePlex:
		return nil
	case CacheTypeUserStore:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown CacheType value '%s'", *t)
	}
}

// Enum implements Enum
func (t CacheType) Enum() []any {
	return []any{
		"all",
		"authz",
		"company_config",
		"plex",
		"userstore",
	}
}

// AllCacheTypes is a slice of all CacheType values
var AllCacheTypes = []CacheType{
	CacheTypeAll,
	CacheTypeAuthZ,
	CacheTypeCompanyConfig,
	CacheTypePlex,
	CacheTypeUserStore,
}
