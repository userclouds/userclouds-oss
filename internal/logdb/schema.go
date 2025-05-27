package logdb

import "userclouds.com/infra/migrate"

// UsedColumns is a list of columns by table used by the orm in this build
// this is used by the db connection schema validator, and "created" at load time
// by a series of _generated.go init() functions to allow parallel data generation
// by genorm during `make codegen`
var UsedColumns = map[string][]string{}

// Schema should be auto-generated soon, but is used for validation
var Schema = migrate.Schema{
	Columns:    UsedColumns,
	Migrations: Migrations,
}
