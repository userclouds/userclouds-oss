package parametertype

// AuthenticationMethods is a parameter type representing a set of authentication method
// options.  The parameter value must be a comma-delimited list of authentication methods.
// Zero or more methods can be specified, with no duplicates.
const AuthenticationMethods Type = "authentication_methods"

func init() {
	validator, err := makeOptionValidator(requireAny, matchAllOptionTypes)
	if err != nil {
		panic(err)
	}
	if err := registerParameterType(AuthenticationMethods, validator); err != nil {
		panic(err)
	}
}
