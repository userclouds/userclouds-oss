package companyconfig

import "userclouds.com/infra/migrate"

// UsedColumns is a list of columns by table used by the orm in this build
// this is used by the db connection schema validator, and "created" at load time
// by a series of _generated.go init() functions to allow parallel data generation
// by genorm during `make codegen`
var UsedColumns = map[string][]string{}

// we put the migrations schema in here so we can validate those queries (rather than just exempting them)
// and because we don't support duplicate table names across databases, having it in one place will
// actually handle it everywhere (since the schema is identical)
func init() {
	UsedColumns["migrations"] = []string{
		"down",
		"dsc",
		"tbl",
		"up",
		"version",
	}
}

// BaselineSchemaVersion is the version used for SchemaBaseline
var BaselineSchemaVersion = 99

// SchemaBaseline is the baseline companyconfig schema to use for migration tests and schema generation
var SchemaBaseline = migrate.Schema{
	Migrations:       Migrations[:BaselineSchemaVersion],
	CreateStatements: schema99,
	Columns:          UsedColumns,
}

var schema99 = []string{
	`CREATE TABLE public.companies (
	id UUID NOT NULL,
	name VARCHAR NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL DEFAULT now(),
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	CONSTRAINT companies_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE TABLE public.event_metadata (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	service VARCHAR NOT NULL,
	category VARCHAR NOT NULL,
	string_id VARCHAR NOT NULL,
	attributes JSONB NOT NULL DEFAULT '{}',
	code INT8 NULL,
	url VARCHAR NULL,
	name VARCHAR NULL,
	description VARCHAR NULL,
	CONSTRAINT event_metadata_pk PRIMARY KEY (deleted, id),
	UNIQUE (string_id, deleted),
	UNIQUE (code, deleted)
)`,
	`CREATE TABLE public.invite_keys (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	type INT8 NOT NULL,
	key VARCHAR NOT NULL,
	expires TIMESTAMP NOT NULL,
	used BOOL NOT NULL,
	company_id UUID NULL,
	"role" VARCHAR NULL,
	invitee_user_id UUID NULL,
	invitee_email VARCHAR NOT NULL DEFAULT '',
	tenant_roles JSONB NOT NULL DEFAULT '{}',
	CONSTRAINT invite_keys_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE INDEX invite_keys_invitee_user_id_idx ON invite_keys (invitee_user_id)`,
	`CREATE INDEX invite_keys_key_idx ON invite_keys (key)`,
	`CREATE TABLE public.sessions (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	id_token VARCHAR NOT NULL DEFAULT '',
	state VARCHAR NOT NULL DEFAULT '',
	access_token VARCHAR NOT NULL DEFAULT '',
	refresh_token VARCHAR NOT NULL DEFAULT '',
	impersonator_id_token VARCHAR NOT NULL DEFAULT '',
	impersonator_access_token VARCHAR NOT NULL DEFAULT '',
	impersonator_refresh_token VARCHAR NOT NULL DEFAULT '',
	CONSTRAINT sessions_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE TABLE public.tenant_user_column_cleanup (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	run_id VARCHAR NOT NULL,
	CONSTRAINT tenant_user_column_cleanup_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE TABLE public.tenants (
	id UUID NOT NULL,
	name VARCHAR NOT NULL,
	company_id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL DEFAULT now(),
	tenant_url VARCHAR NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	use_organizations BOOL NOT NULL DEFAULT false,
	state VARCHAR NOT NULL DEFAULT 'creating',
	CONSTRAINT tenants_pk PRIMARY KEY (deleted, id),
	UNIQUE (tenant_url, deleted)
)`,
	`CREATE TABLE public.tenants_internal (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	tenant_db_config JSONB NOT NULL,
	log_config JSONB NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	cache_config JSONB NOT NULL DEFAULT '{}',
	CONSTRAINT tenants_internal_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE TABLE public.tenants_plex (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	plex_config JSONB NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	_version INT8 NOT NULL DEFAULT 0,
	CONSTRAINT tenants_plex_pk PRIMARY KEY (deleted, id)
)`,
	`CREATE TABLE public.tenants_urls (
	id UUID NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT now(),
	updated TIMESTAMP NOT NULL,
	deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	tenant_id UUID NOT NULL,
	tenant_url VARCHAR NOT NULL,
	validated BOOL NOT NULL DEFAULT false,
	system BOOL NOT NULL DEFAULT false,
	active BOOL NOT NULL DEFAULT false,
	dns_verifier VARCHAR NOT NULL DEFAULT '',
	certificate_valid_until TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
	CONSTRAINT tenants_urls_pk PRIMARY KEY (deleted, id),
	UNIQUE (tenant_url, deleted)
)`,
}
