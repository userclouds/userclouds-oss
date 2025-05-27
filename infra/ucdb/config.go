package ucdb

import (
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

// DBProduct type is used to captures the DB product (Postgres, CockroachDB, etc)
type DBProduct string

// Constants for DBProduct/DBDriver so we don't have magic string all over the code base.
const (
	PostgresDriver = "postgres" // DBDriver
	// DB Product names (all of these use the postgres DB Driver)
	Postgres          = DBProduct("postgres")
	AWSRDSPostgres    = DBProduct("aws-rds-postgres")
	AWSAuroraPostgres = DBProduct("aws-aurora-postgres")
)

// AllDBProducts returns all the DB products
func AllDBProducts() []DBProduct {
	return []DBProduct{
		Postgres,
		AWSRDSPostgres,
		AWSAuroraPostgres,
	}
}

// Config holds database connection info
type Config struct {
	User          string            `json:"user" yaml:"user" validate:"notempty"`
	Password      secret.String     `json:"password" yaml:"password"`
	DBName        string            `json:"dbname" yaml:"dbname" validate:"notempty"`
	DBDriver      string            `json:"dbdriver" yaml:"dbdriver"`   // the underlying protocol used to talk to the DB, for postgres and cockroachdb, this is the same (postgres) since cockroachdb is postgres compatible (mostly)
	DBProduct     DBProduct         `json:"dbproduct" yaml:"dbproduct"` // the underlying DB product name: postgres, cockroachdb, aws-rds-postgres, etc
	Host          string            `json:"host" yaml:"host" validate:"notempty"`
	Port          string            `json:"port" yaml:"port" validate:"notempty"`                     // could be an int, but easier to manage as a string
	RegionalHosts map[string]string `json:"regional_hosts,omitempty" yaml:"regional_hosts,omitempty"` // map from region to host, used to connect to regional DBs

	// may be empty, this lets us use default_transaction_use_follower_reads
	// specifically, if not-nil, the value of this var will be run using ExecContext() on connect
	SessionVars *string `json:"session_vars,omitempty" yaml:"session_vars,omitempty" validate:"allownil"`
}

// IsLocal returns true if the DB is local and doesn't require SSL
func (c *Config) IsLocal() bool {
	return c.DBProduct == Postgres
}

// IsProductPostgres returns true if the DBProduct is postgres (including AWS RDS postgres)
func (c *Config) IsProductPostgres() bool {
	return IsProductPostgres(c.DBProduct)
}

// IsProductPostgres returns true if the DBProduct is postgres (including AWS RDS postgres)
func IsProductPostgres(product DBProduct) bool {
	return product == Postgres || product == AWSRDSPostgres || product == AWSAuroraPostgres
}

// GetRegionalHost returns the regional host for the given region if it exists
func (c *Config) GetRegionalHost(region region.MachineRegion) string {
	if c.RegionalHosts != nil {
		if host, ok := c.RegionalHosts[string(region)]; ok {
			return host
		}
	}

	return ""
}

func (c *Config) extraValidate() error {
	if c.DBDriver != PostgresDriver {
		return ucerr.Errorf("Unsupported DB Driver: '%v' for DB: %v", c.DBDriver, c.DBName)
	}
	if !c.IsProductPostgres() {
		if c.DBProduct == "" {
			// We allow this since the DB config for existing tenants don't have this set, we need to remove this once all tenants DB configs are updated.
			return nil
		}
		return ucerr.Errorf("Unsupported DB Product: '%s'", c.DBProduct)
	}
	return nil
}

//go:generate gendbjson Config

//go:generate genvalidate Config
