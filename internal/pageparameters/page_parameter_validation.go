package pageparameters

import (
	"fmt"
	"strings"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/pageparameters/parametertype"
)

type compositeParameterValidator func(ParameterGetter, param.ClientData) (explanation string, err error)

func getAppliedParameter(pg ParameterGetter, cd param.ClientData, pt pagetype.Type, pn param.Name) (p param.Parameter, explanation string, err error) {
	p, found := pg(pt, pn)
	if !found {
		return p, fmt.Sprintf("parameter '%s' not found for page type '%s'", pn, pt), ucerr.Errorf("parameter '%s' not found for page type '%s'", pn, pt)
	}
	if p, err = p.ApplyClientData(cd); err != nil {
		return p, fmt.Sprintf("could not apply client data to parameter '%s' for page type '%s'", pn, pt), ucerr.Wrap(err)
	}
	return p, "", nil
}

var authenticationMethodValidator = func(pg ParameterGetter, cd param.ClientData) (explanation string, err error) {
	// make sure that any specified authentication methods are enabled for a tenant and app

	authMethods, explanation, err := getAppliedParameter(pg, cd, pagetype.EveryPage, param.AuthenticationMethods)
	if err != nil {
		return explanation, ucerr.Wrap(err)
	}

	enabledAuthMethods, explanation, err := getAppliedParameter(pg, cd, pagetype.EveryPage, param.EnabledAuthenticationMethods)
	if err != nil {
		return explanation, ucerr.Wrap(err)
	}

	for _, authMethod := range parametertype.GetOptions(authMethods.Value) {
		if !strings.Contains(enabledAuthMethods.Value, authMethod) {
			explanation := fmt.Sprintf("enabled authentication method '%s' is not configured", authMethod)
			return explanation, ucerr.New(explanation)
		}
	}

	return "", nil
}

var mfaMethodValidator = func(pg ParameterGetter, cd param.ClientData) (explanation string, err error) {
	// make sure that all specified mfa methods are enabled for a tenant and app

	mfaMethods, explanation, err := getAppliedParameter(pg, cd, pagetype.EveryPage, param.MFAMethods)
	if err != nil {
		return explanation, ucerr.Wrap(err)
	}

	enabledMFAMethods, explanation, err := getAppliedParameter(pg, cd, pagetype.EveryPage, param.EnabledMFAMethods)
	if err != nil {
		return explanation, ucerr.Wrap(err)
	}

	selectedMFAMethods := parametertype.GetOptions(mfaMethods.Value)
	for _, mfaMethod := range selectedMFAMethods {
		if !strings.Contains(enabledMFAMethods.Value, mfaMethod) {
			explanation := fmt.Sprintf("enabled MFA method '%s' is not configured", mfaMethod)
			return explanation, ucerr.New(explanation)
		}
	}

	// if mfa is required, make sure that there is at least one specified mfa method

	mfaRequired, explanation, err := getAppliedParameter(pg, cd, pagetype.EveryPage, param.MFARequired)
	if err != nil {
		return explanation, ucerr.Wrap(err)
	}

	if mfaRequired.Value == "true" && len(selectedMFAMethods) == 0 {
		explanation := "cannot require MFA without at least one enabled MFA method"
		return explanation, ucerr.New(explanation)
	}

	// make sure recovery code is not the only specified mfa method

	if len(selectedMFAMethods) == 1 && oidc.MFAChannelType(selectedMFAMethods[0]) == oidc.MFARecoveryCodeChannel {
		explanation := "cannot select MFA recovery codes without at least one other MFA method"
		return explanation, ucerr.New(explanation)
	}

	return "", nil
}

var compositeParameterValidators = []compositeParameterValidator{authenticationMethodValidator, mfaMethodValidator}

// ValidateCompositeParameters will execute a series of composite parameter validators, returning
// an error if any do not successfully pass. Composite validators are used in cases where the validity
// of a parameter is dependent on the values of one or more other parameters.
func ValidateCompositeParameters(pg ParameterGetter, cd param.ClientData) (explanation string, err error) {
	for _, cv := range compositeParameterValidators {
		if explanation, err := cv(pg, cd); err != nil {
			return explanation, ucerr.Wrap(err)
		}
	}

	return "", nil
}
