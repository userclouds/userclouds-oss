package internal

import (
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/ucdb"
)

// DPConfig is a temp conversion struct
type DPConfig struct {
	DataProc Config `yaml:"dataprocessor" json:"dataprocessor"`
}

// Validate ...
func (dp *DPConfig) Validate() error {
	return nil
}

// Config defines configuration for data processor
type Config struct {
	Logger logtransports.Config `yaml:"logger" json:"logger"`
	DB     ucdb.Config          `yaml:"db" json:"db"`

	Enabled bool `yaml:"enabled" json:"enabled"`
}

//go:generate genvalidate Config
