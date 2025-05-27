package parameter

import "userclouds.com/internal/pageparameters/parametertype"

// Name is the name of a parameter
type Name string

// Parameter is a struct defining a page parameter
type Parameter struct {
	Name  Name               `json:"parameter_name"`
	Type  parametertype.Type `json:"parameter_type"`
	Value string             `json:"parameter_value"`
}

//go:generate genvalidate Parameter

// ClientData specifies the client-specific data that can be
// referred to within a parameter. The variables are all derivable
// from tenant and app configurations associated with a request
type ClientData struct {
	AllowCreation                 string
	AppName                       string
	DisabledAuthenticationMethods string
	DisabledMFAMethods            string
	EnabledAuthenticationMethods  string
	EnabledMFAMethods             string
	OIDCAuthenticationSettings    string
	PasswordResetEnabled          string
}
