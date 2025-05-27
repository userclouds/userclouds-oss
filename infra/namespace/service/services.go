package service

import (
	"slices"
	"strings"
)

// Service defines a UserClouds service
// For now it's just a string to use as a common identifier
// and give us things like type checking, but over time you can imagine
// this becomes a deeper struct (maybe)
type Service string

// Service names
const (
	Tokenizer      Service = "tokenizer"
	Plex           Service = "plex"
	IDP            Service = "idp" // UserStore?
	LogServer      Service = "logserver"
	AuthZ          Service = "authz"
	CheckAttribute Service = "checkattribute"
	Console        Service = "console"
	Worker         Service = "worker"

	SDK Service = "sdk"

	idpServiceName = "userstore"

	Undefined Service = "undefined"
)

// IsValid validates a service name
func IsValid(svc Service) bool {
	if svc.IsUndefined() {
		return false
	}
	return slices.Contains(AllServices, svc)
}

// IsUndefined checks if the service is undefined
func (s Service) IsUndefined() bool {
	return s == Undefined
}

// ToServiceName returns the service name we use in the kubernetes service name
func (s Service) ToServiceName() string {
	if s == IDP {
		return idpServiceName
	}
	return string(s)
}

// ToCodeName returns the service name as it defined in the code (for codegen)
func (s Service) ToCodeName() string {
	if s == LogServer {
		return "LogServer"
	}
	if s == AuthZ {
		return "AuthZ"
	}
	if s == IDP {
		return "IDP"
	}
	svcStr := string(s)
	return strings.ToUpper(string(svcStr[0])) + svcStr[1:]
}

//go:generate genstringconstenum Service

// AllServices is a list of all valid services
var AllServices = allServiceValues

// HeadlessConsoleServices is a list of services that run in a headless container
var HeadlessConsoleServices = []Service{Plex, IDP, AuthZ}

// AllWebServices is a list of services that run on a Web node (EB)
var AllWebServices = []Service{AuthZ, IDP, LogServer, Plex, Console}
