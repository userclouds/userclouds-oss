package main

import (
	"os"

	startAuthz "userclouds.com/authz/start"
	startConsole "userclouds.com/console/start"
	startUserStore "userclouds.com/idp/start"
	"userclouds.com/infra/namespace/service"
	startLogServer "userclouds.com/logserver/start"
	startPlex "userclouds.com/plex/start"
	startWorker "userclouds.com/worker/start"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("No service name provided")

	} else if len(args) > 1 {
		panic("Too many arguments provided")
	}
	svc := service.Service(args[0])

	switch svc {
	case service.Plex:
		startPlex.RunPlex()
	case service.IDP:
		startUserStore.RunUserStore()
	case service.AuthZ:
		startAuthz.RunAuthZ()
	case service.CheckAttribute:
		startAuthz.RunCheckAttribute()
	case service.Worker:
		startWorker.RunWorker()
	case service.LogServer:
		startLogServer.RunLogServer()
	case service.Console:
		startConsole.RunConsole()
	default:
		panic("Unknown command: " + string(svc))

	}
}
