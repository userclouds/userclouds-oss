package rootdb

import "userclouds.com/infra/migrate"

// migrations specifies all the migrations necessary to get our cockroachDB infra
// up and running, eg. creating users and the root authz/etc databases
// TODO: this is where we should create per-service users and control access better,
// although it means we might be checking passwords in here? Can we use secrets here?
var migrations = migrate.Migrations{
	{
		Version:      0,
		Table:        "users",
		Desc:         "",
		Up:           `CREATE USER userclouds;`,
		DeprecatedUp: []string{`CREATE USER IF NOT EXISTS userclouds;`},
		Down:         `DROP USER userclouds;`,
	},
	// TODO: fix typo in Descs, it's not 'companyconfig table' or 'IDP table',
	// they're actually databases
	{
		Version:      1,
		Table:        "databases",
		Desc:         "create orgconfig table",
		Up:           `CREATE DATABASE orgconfig;`,
		DeprecatedUp: []string{`CREATE DATABASE IF NOT EXISTS orgconfig; GRANT ALL PRIVILEGES ON DATABASE orgconfig TO userclouds;`},
		Down:         `DROP DATABASE orgconfig;`,
	},
	{
		Version:      2,
		Table:        "databases",
		Desc:         "add authz table -- this likely becomes per-tenant in the future?",
		Up:           `CREATE DATABASE authz;`,
		DeprecatedUp: []string{`CREATE DATABASE IF NOT EXISTS authz; GRANT ALL PRIVILEGES ON DATABASE authz TO userclouds;`},
		Down:         `DROP DATABASE authz;`,
	},
	{
		// TODO: this should get removed in favor of using root account (or something else) for per-tenant provisioning
		Version:      3,
		Table:        "databases",
		Desc:         "add IDP table for now, we should change this later since no DB is really needed",
		Up:           `CREATE DATABASE idp;`,
		DeprecatedUp: []string{`CREATE DATABASE IF NOT EXISTS idp; GRANT ALL PRIVILEGES ON DATABASE idp TO userclouds;`},
		Down:         `DROP DATABASE idp`,
	},
	{
		// TODO: this should be different users for different roles, upcoming change
		Version: 4,
		Table:   "users",
		Desc:    "let the userclouds user create users & databases for IDP provisioning",
		Up:      `ALTER USER userclouds CREATEDB CREATEROLE LOGIN;`,
		Down:    `ALTER USER userclouds NOCREATEDB NOCREATEROLE NOLOGIN;`,
	},
	{
		Version:      5,
		Table:        "databases",
		Desc:         "add status_0 table for now",
		Up:           `CREATE DATABASE status_0;`,
		DeprecatedUp: []string{`CREATE DATABASE IF NOT EXISTS status_0; GRANT ALL PRIVILEGES ON DATABASE status_0 TO userclouds;`},
		Down:         `DROP DATABASE status_0`,
	},
	{
		Version:        6,
		Table:          "databases",
		Desc:           "remove authz table",
		Up:             `DROP DATABASE authz;`,
		Down:           `CREATE DATABASE authz;`,
		DeprecatedDown: []string{`CREATE DATABASE IF NOT EXISTS authz; GRANT ALL PRIVILEGES ON DATABASE authz TO userclouds;`},
	},
	{
		Version:        7,
		Table:          "databases",
		Desc:           "remove old idp database",
		Up:             `DROP DATABASE idp;`,
		Down:           `CREATE DATABASE idp;`,
		DeprecatedDown: []string{`CREATE DATABASE IF NOT EXISTS idp; GRANT ALL PRIVILEGES ON DATABASE idp TO userclouds;`},
	},
	{
		Version: 8,
		Table:   "databases",
		Desc:    "rename status_0 to status_{uuid.Nil}",
		Up:      `ALTER DATABASE status_0 RENAME TO status_00000000000000000000000000000000;`,
		Down:    `ALTER DATABASE status_00000000000000000000000000000000 RENAME TO status_0;`,
	},
	{
		// if you have trouble with this migration, make sure to run `make devsetup` to
		// make sure that userclouds_dev_root has perms
		// TODO: should we drop all the old ones?
		Version: 9,
		Table:   "databases",
		Desc:    "get rid of cockroach status tables as they're confusing",
		Up:      `DROP DATABASE status_00000000000000000000000000000000;`,
		Down:    `CREATE DATABASE status_00000000000000000000000000000000;`,
	},
	{
		Version: 10,
		Table:   "databases",
		Desc:    "move orgconfig to companyconfig",
		Up:      `ALTER DATABASE orgconfig RENAME TO companyconfig;`,
		Down:    `ALTER DATABASE companyconfig RENAME TO orgconfig;`,
	},
	{
		Version: 11,
		Table:   "databases",
		Desc:    "grant privileges to usercloud user to the companyconfig DB",
		Up:      `GRANT ALL PRIVILEGES ON DATABASE companyconfig TO userclouds;`,
		Down:    `/* noop */`,
	},
	{
		Version: 12,
		Table:   "databases",
		Desc:    "restrict expensive updates in cockroach DB",
		Up:      `/* noop */`,
		Down:    `/* noop */`,
	},
}
