package parametertype

// SelectedAuthenticationMethods is a parameter type representing the authentication method
// options that a client wants to support.  The parameter value must be a comma-delimited
// list of authentication methods.  At least one method must be specified, with no duplicates.
const SelectedAuthenticationMethods Type = "selected_authentication_methods"

func init() {
	validator, err := makeOptionValidator(requireAtLeastOne, matchAllOptionTypes)
	if err != nil {
		panic(err)
	}
	if err := registerParameterType(SelectedAuthenticationMethods, validator); err != nil {
		panic(err)
	}
}
