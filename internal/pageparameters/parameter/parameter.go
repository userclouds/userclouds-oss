package parameter

import (
	"bytes"
	"fmt"
	texttemplate "text/template"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/pageparameters/parametertype"
)

var defaultClientData = ClientData{
	AllowCreation:                 "false",
	AppName:                       "app name",
	DisabledAuthenticationMethods: "facebook,google,linkedin",
	DisabledMFAMethods:            "sms",
	EnabledAuthenticationMethods:  "password,passwordless",
	EnabledMFAMethods:             "email,authenticator,recovery_code",
	OIDCAuthenticationSettings:    "facebook:Facebook:Sign in with Facebook:Sign in to existing account with Facebook,google:Google:Sign in with Google:Sign in to existing account with Google,linkedin:LinkedIn:Sign in with LinkedIn:Sign in to existing account with LinkedIn,microsoft:Microsoft:Sign in with Microsoft:Sign in to existing account with Microsoft",
	PasswordResetEnabled:          "false",
}

var parameterNames = []Name{}

var parametersByName = map[Name]Parameter{}
var parametersWhitelistByName = map[Name]bool{}

// IsParameterWhitelistOnly returns whether a parameter is on the whitelist
func IsParameterWhitelistOnly(n Name) bool {
	return parametersWhitelistByName[n]
}

func register(n Name, t parametertype.Type) {
	// TODO: refactor so that we only panic in the init method that actually calls this
	if _, present := parametersByName[n]; present {
		panic(fmt.Sprintf("duplicate registration for parameter name '%s'", n))
	}

	p := Parameter{Name: n, Type: t}
	parametersByName[n] = p
	parameterNames = append(parameterNames, n)
}

func registerWhitelisted(n Name, t parametertype.Type) {
	register(n, t)
	parametersWhitelistByName[n] = true
}

// Names returns a copy of the slice of all register parameter names
func Names() (names []Name) {
	return append(names, parameterNames...)
}

// ApplyClientData applies the specified client data against the parameter,
// replacing any instances of the client data variables with the associated value.
// If the parameter refers to any client data variables not present in the client
// data, an error will be returned.
func (p Parameter) ApplyClientData(clientData ClientData) (Parameter, error) {
	parameterTemplate, err := texttemplate.New("parameter").Parse(p.Value)
	if err != nil {
		return p, ucerr.Wrap(err)
	}

	buf := &bytes.Buffer{}
	if err := parameterTemplate.Execute(buf, clientData); err != nil {
		return p, ucerr.Wrap(err)
	}

	p.Value = buf.String()
	return p, nil
}

// ApplyDefaultClientData calls ApplyClientData with defaultClientData
func (p Parameter) ApplyDefaultClientData() (Parameter, error) {
	return p.ApplyClientData(defaultClientData)
}

// MakeParameter verifies that the specified parameter name is supported and creates
// a new parameter instance for the name, associated parameter type, and specified
// value
func MakeParameter(n Name, value string) (p Parameter) {
	// If the parameter name is not found, that means it is invalid, and
	// we don't override the default properties of the return parameter.
	// This will result in the return parameter failing the call to
	// Validate because the name, type and value will all be invalid.
	if v, present := parametersByName[n]; present {
		p.Name = v.Name
		p.Type = v.Type
		p.Value = value
	}
	return p
}

// String returns a string representation of a parameter
func (p Parameter) String() string {
	return fmt.Sprintf("{Type: '%s' Name: '%s' Value: '%s'}", p.Type, p.Name, p.Value)
}

// Validate implements the Validatable interface and verifies the parameter name is valid
func (n Name) Validate() error {
	if _, found := parametersByName[n]; !found {
		return ucerr.Errorf("name '%s' is unrecognized", n)
	}

	return nil
}

func (p *Parameter) extraValidate() error {
	if !p.Type.IsValid(p.Value) {
		return ucerr.Errorf("value is invalid for parameter '%s'", p)
	}

	return nil
}

// ValidateDefault applies default client data and calls Validate
func (p Parameter) ValidateDefault() error {
	p, err := p.ApplyDefaultClientData()
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := p.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
