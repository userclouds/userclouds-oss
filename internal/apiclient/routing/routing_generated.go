// NOTE: automatically generated file -- DO NOT EDIT

package routing

import (
	"userclouds.com/infra/namespace/service"
	"userclouds.com/internal/routinghelper"
)

// Routing rules that are not host specific
var nonHostRules = []routinghelper.Rule{
	{
		PathPrefixes: []string{"/authz/", "/auditlog/"},
		HostHeaders:  nil,
		Service:      service.AuthZ,
	},
	{
		PathPrefixes: []string{"/authn/", "/userevent/", "/userstore/", "/tokenizer/", "/s3shim/"},
		HostHeaders:  nil,
		Service:      service.IDP,
	},
	{
		PathPrefixes: []string{"/logserver/"},
		HostHeaders:  nil,
		Service:      service.LogServer,
	},
	{
		PathPrefixes: []string{"/"},
		HostHeaders:  nil,
		Service:      service.Plex,
	},
}

// serviceToPort maps a service name to a port
var serviceToPort = map[service.Service]int{
	service.AuthZ:     5200,
	service.Console:   5300,
	service.IDP:       5100,
	service.LogServer: 5500,
	service.Plex:      5000,
}
