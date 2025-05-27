package rootdbstatus

import (
	"userclouds.com/infra/ucdb"
)

// Config holds the config for the "root" database setup
type Config struct {
	DB ucdb.Config `yaml:"db" json:"db"`
}

//go:generate genvalidate Config
