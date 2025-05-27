// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t InviteKeyType) MarshalText() ([]byte, error) {
	switch t {
	case InviteKeyTypeExistingCompany:
		return []byte("existingcompany"), nil
	case InviteKeyTypeUnknown:
		return []byte("unknown"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown InviteKeyType value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *InviteKeyType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "existingcompany":
		*t = InviteKeyTypeExistingCompany
	case "unknown":
		*t = InviteKeyTypeUnknown
	default:
		return ucerr.Friendlyf(nil, "unknown InviteKeyType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *InviteKeyType) Validate() error {
	switch *t {
	case InviteKeyTypeExistingCompany:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown InviteKeyType value '%d'", *t)
	}
}

// Enum implements Enum
func (t InviteKeyType) Enum() []any {
	return []any{
		"existingcompany",
	}
}

// AllInviteKeyTypes is a slice of all InviteKeyType values
var AllInviteKeyTypes = []InviteKeyType{
	InviteKeyTypeExistingCompany,
}

// just here for easier debugging
func (t InviteKeyType) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
