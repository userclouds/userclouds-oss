package parametertype

import (
	"strings"

	"userclouds.com/infra/oidc"
)

// MFAMethodTypes is a comma-delimited list of the supported mfa method options
const MFAMethodTypes = "email,sms,authenticator,recovery_code"

// MFAMethods is a parameter type representing a set of mfa methods.  The
// parameter value must be a comma-delimited list of mfa methods, each of which
// must be present in MFAMethodTypes. Zero or more methods can be specified,
// with no duplicates.
const MFAMethods Type = "mfa_methods"

func init() {
	// ensure that all specified types are valid oidc MFA channels
	for mt := range strings.SplitSeq(MFAMethodTypes, ",") {
		ct := oidc.MFAChannelType(mt)
		if err := ct.Validate(); err != nil {
			panic(err)
		}
	}

	validator, err := makeOptionValidator(requireAny, MFAMethodTypes)
	if err != nil {
		panic(err)
	}
	if err := registerParameterType(MFAMethods, validator); err != nil {
		panic(err)
	}
}
