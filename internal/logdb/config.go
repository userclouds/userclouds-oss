package logdb

import (
	"userclouds.com/infra/ucdb"
)

// Config holds the config for the companyconfig" database setup
type Config struct {
	DB ucdb.Config `yaml:"log_db" json:"log_db"`
}

//go:generate genvalidate Config
