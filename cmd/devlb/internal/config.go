package internal

import (
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/service"
)

// Config holds config info for the dev universe load balancer
type Config struct {
	service.Endpoint `yaml:"svc_listener" json:"svc_listener"`
	Log              logtransports.Config `yaml:"logger" json:"logger"`
}

//go:generate genvalidate Config
