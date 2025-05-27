package rootdbstatus

import "userclouds.com/infra/migrate"

// Migrations specifies all the migrations necessary to get our postgres/RDS infra
// up and running, eg. creating users and the root authz/etc databases
var Migrations = migrate.Migrations{
	{
		Version: 0,
		Table:   "users",
		Desc:    "create our default service user",
		// When using the same DB Server for both root DB and root status DB, then the 'userclouds' user already exists. so we want to have a no-op in this case.
		Up: `DO $do$
BEGIN
	IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'userclouds') THEN
		RAISE NOTICE 'Role/user userclouds already exists. Skipping.';
	ELSE
		CREATE USER userclouds CREATEROLE CREATEDB;
	END IF;
END
$do$;`,

		Down: `DROP USER userclouds;`,
	},
	{
		Version: 1,
		Table:   "databases",
		Desc:    "add status_00000000000000000000000000000000 table",
		Up:      `CREATE DATABASE status_00000000000000000000000000000000;`,
		Down:    `DROP DATABASE status_00000000000000000000000000000000;`,
	},
	{
		// TODO: RDS doesn't allow "GRANT ALL" so we enumerate, but we should ensure these are minimal :)
		Version: 2,
		Table:   "users",
		Desc:    "user permissions",
		Up:      `GRANT ALL PRIVILEGES ON DATABASE status_00000000000000000000000000000000 TO userclouds;`,
		Down:    `REVOKE ALL ON *.* FROM userclouds`,
	},
}
