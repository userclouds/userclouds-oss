// NOTE: automatically generated file -- DO NOT EDIT

package storage

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t MFAPurpose) MarshalText() ([]byte, error) {
	switch t {
	case MFAPurposeConfigure:
		return []byte("configure"), nil
	case MFAPurposeInvalid:
		return []byte("invalid"), nil
	case MFAPurposeLogin:
		return []byte("login"), nil
	case MFAPurposeLoginSetup:
		return []byte("loginsetup"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown MFAPurpose value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *MFAPurpose) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "configure":
		*t = MFAPurposeConfigure
	case "invalid":
		*t = MFAPurposeInvalid
	case "login":
		*t = MFAPurposeLogin
	case "loginsetup":
		*t = MFAPurposeLoginSetup
	default:
		return ucerr.Friendlyf(nil, "unknown MFAPurpose value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *MFAPurpose) Validate() error {
	switch *t {
	case MFAPurposeConfigure:
		return nil
	case MFAPurposeLogin:
		return nil
	case MFAPurposeLoginSetup:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown MFAPurpose value '%d'", *t)
	}
}

// Enum implements Enum
func (t MFAPurpose) Enum() []any {
	return []any{
		"configure",
		"login",
		"loginsetup",
	}
}

// AllMFAPurposes is a slice of all MFAPurpose values
var AllMFAPurposes = []MFAPurpose{
	MFAPurposeConfigure,
	MFAPurposeLogin,
	MFAPurposeLoginSetup,
}

// just here for easier debugging
func (t MFAPurpose) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
