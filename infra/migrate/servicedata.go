package migrate

import "userclouds.com/infra/ucdb"

// ServiceData is the set of information needed by `bin/migrate`
// to migrate the database for a service.
// TODO: service isn't the right abstraction here; not all databases currently
// have a 1:1 mapping to a service (`companyconfig` DB kinda maps to `console`,
// `rootdb` has no corresponding service, and `tenantdb` doesn't have a service
// but we use IDP to deal with it).
type ServiceData struct {
	DBCfg                    *ucdb.Config
	Migrations               Migrations
	BaselineVersion          int
	BaselineCreateStatements []string
	PostgresOnlyExtensions   []string
}
