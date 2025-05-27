package servicecolors

import (
	"context"

	"userclouds.com/infra/namespace/color"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
)

// Colors is a map of services to console colors
var Colors = map[service.Service]color.Color{
	service.Plex:           color.BrightBlue,
	service.AuthZ:          color.BrightCyan,
	service.CheckAttribute: color.Purple,
	service.IDP:            color.BrightPurple,
	service.Console:        color.Green,
	service.LogServer:      color.Yellow,
	service.Worker:         color.Blue,
}

// MustGetColor returns the color for a given service name
func MustGetColor(ctx context.Context, name string) color.Color {
	svc := service.Service(name)
	if !service.IsValid(svc) {
		uclog.Fatalf(ctx, "Invalid service: '%v'", svc)
	}
	svcColor, ok := Colors[svc]
	if !ok {
		uclog.Fatalf(ctx, "No color defined for service: '%v'", svc)
	}
	return svcColor
}
