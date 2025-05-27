package parametertype

// OIDCAuthenticationSettings is a parameter type representing a set of settings for OIDC
// authentication methods.  The parameter value must be a comma-delimited list of settings,
// and each setting must be of the form:
//
// string:string:string:string
//
// Zero or more entries can be specified, with no duplicates.
const OIDCAuthenticationSettings Type = "oidc_authentication_settings"

const oidcAuthenticationSettingsRegexp = "regexp:^[^:]+:[^:]+:[^:]+:[^:]+$"

func init() {
	validator, err := makeOptionValidator(requireAny, oidcAuthenticationSettingsRegexp)
	if err != nil {
		panic(err)
	}
	if err := registerParameterType(OIDCAuthenticationSettings, validator); err != nil {
		panic(err)
	}
}
