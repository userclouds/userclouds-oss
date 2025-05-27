package tenantplex

import (
	"userclouds.com/infra/ucdb"
)

// TenantPlex is the DB model for the Plex config for a tenant.
type TenantPlex struct {
	ucdb.VersionBaseModel

	PlexConfig TenantConfig `db:"plex_config" json:"plex_config" yaml:"plex_config"`
}

//go:generate genvalidate TenantPlex
