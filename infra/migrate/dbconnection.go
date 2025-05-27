package migrate

import "userclouds.com/infra/ucdb"

// HasDBConfig defines an interface to allow tooling to access /internal DB config
type HasDBConfig interface {
	GetDBConfig() ucdb.Config
}
