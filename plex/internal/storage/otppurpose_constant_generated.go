// NOTE: automatically generated file -- DO NOT EDIT

package storage

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t OTPPurpose) MarshalText() ([]byte, error) {
	switch t {
	case OTPPurposeAccountVerify:
		return []byte("accountverify"), nil
	case OTPPurposeInvalid:
		return []byte("invalid"), nil
	case OTPPurposeInvite:
		return []byte("invite"), nil
	case OTPPurposeLogin:
		return []byte("login"), nil
	case OTPPurposeResetPassword:
		return []byte("resetpassword"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown OTPPurpose value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *OTPPurpose) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "accountverify":
		*t = OTPPurposeAccountVerify
	case "invalid":
		*t = OTPPurposeInvalid
	case "invite":
		*t = OTPPurposeInvite
	case "login":
		*t = OTPPurposeLogin
	case "resetpassword":
		*t = OTPPurposeResetPassword
	default:
		return ucerr.Friendlyf(nil, "unknown OTPPurpose value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *OTPPurpose) Validate() error {
	switch *t {
	case OTPPurposeAccountVerify:
		return nil
	case OTPPurposeInvite:
		return nil
	case OTPPurposeLogin:
		return nil
	case OTPPurposeResetPassword:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown OTPPurpose value '%d'", *t)
	}
}

// Enum implements Enum
func (t OTPPurpose) Enum() []any {
	return []any{
		"accountverify",
		"invite",
		"login",
		"resetpassword",
	}
}

// AllOTPPurposes is a slice of all OTPPurpose values
var AllOTPPurposes = []OTPPurpose{
	OTPPurposeAccountVerify,
	OTPPurposeInvite,
	OTPPurposeLogin,
	OTPPurposeResetPassword,
}

// just here for easier debugging
func (t OTPPurpose) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
