package companyconfig

import (
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// TenantLogConfig contains per tenant logging configuration
type TenantLogConfig struct {
	LogDB ucdb.Config `json:"logdb" yaml:"logdb,omitempty"`
}

//go:generate gendbjson TenantLogConfig

//go:generate genvalidate TenantLogConfig

func (c *TenantLogConfig) extraValidate() error {
	if c.LogDB.DBDriver != ucdb.PostgresDriver && c.LogDB.DBDriver != "" {
		return ucerr.New("LogDB configuration error: only 'postgres' driver is supported")
	}
	return nil
}

// Config holds the config for the companyconfig" database setup
type Config struct {
	CompanyDB ucdb.Config `yaml:"company_db" json:"company_db"`
}

//go:generate genvalidate Config
