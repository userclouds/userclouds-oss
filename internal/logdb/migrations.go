package logdb

import "userclouds.com/infra/migrate"

// Migrations specifies all the migrations necessary to get status DB up to today
var Migrations = migrate.Migrations{
	{
		Version: 0,
		Table:   "metrics_plex",
		Desc:    "create initial metrics_plex table",
		Up: `CREATE TABLE metrics_plex (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_plex;`,
	},
	{
		Version: 1,
		Table:   "metrics_idp",
		Desc:    "create initial metrics_idp table",
		Up: `CREATE TABLE metrics_idp (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_idp;`,
	},
	{
		Version: 2,
		Table:   "metrics_console",
		Desc:    "create initial metrics_console table",
		Up: `CREATE TABLE metrics_console (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_console;`,
	},
	{
		Version: 3,
		Table:   "metrics_authz",
		Desc:    "create initial metrics_authz table",
		Up: `CREATE TABLE metrics_authz (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_authz;`,
	},
	{
		Version: 4,
		Table:   "user_events",
		Desc:    "create initial user_events table",
		Up: `CREATE TABLE user_events (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			type VARCHAR NOT NULL,
			user_alias VARCHAR NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}'::JSONB);
			CREATE INDEX ON user_events (user_alias);`,
		Down: `DROP TABLE user_events;`,
	},
	{
		Version: 5,
		Table:   "user_events",
		// NB: we have to drop and recreate here because the postgres vs cdb syntax
		// differs, and we use cdb for tests / postgres elsewhere
		// This is safe because the user_events UPSERT statement never worked in prod
		// so we aren't losing non-test data
		Desc: "recreate user_events table with correct primary key",
		Up: `DROP TABLE user_events;
			CREATE TABLE user_events (
				id UUID NOT NULL,
				created TIMESTAMP NOT NULL DEFAULT NOW(),
				updated TIMESTAMP NOT NULL,
				deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
				type VARCHAR NOT NULL,
				user_alias VARCHAR NOT NULL,
				payload JSONB NOT NULL DEFAULT '{}'::JSONB,
				PRIMARY KEY (id, deleted));
				CREATE INDEX ON user_events (user_alias)`,
		Down: `DROP TABLE user_events;
			CREATE TABLE user_events (
				id UUID NOT NULL PRIMARY KEY,
				created TIMESTAMP NOT NULL DEFAULT NOW(),
				updated TIMESTAMP NOT NULL,
				deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
				type VARCHAR NOT NULL,
				user_alias VARCHAR NOT NULL,
				payload JSONB NOT NULL DEFAULT '{}'::JSONB);
				CREATE INDEX ON user_events (user_alias);`,
	},
	{
		Version: 6,
		Table:   "metrics_tokenizer",
		Desc:    "create initial metrics_tokenizer table",
		Up: `CREATE TABLE metrics_tokenizer (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_tokenizer;`,
	},
	{
		Version: 7,
		Table:   "event_metadata",
		Desc:    "create initial metadata table",

		Up: `CREATE TABLE event_metadata (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			service VARCHAR NOT NULL,
			category VARCHAR NOT NULL,
			string_id VARCHAR NOT NULL,
			code BIGINT,
			url VARCHAR,
			name VARCHAR,
			description VARCHAR,
			PRIMARY KEY (id, deleted));
			CREATE UNIQUE INDEX event_metadata_string_id_deleted ON event_metadata (string_id, deleted);
			CREATE UNIQUE INDEX event_metadata_code_deleted ON event_metadata (code, deleted);`,
		Down: `DROP TABLE event_metadata;`,
	},
	{
		Version: 8,
		Table:   "event_metadata",
		Desc:    "add attributes column for event metadata",
		Up:      `ALTER TABLE event_metadata ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE event_metadata DROP COLUMN attributes;`,
	},
	{
		Version: 9,
		Table:   "metrics_worker",
		Desc:    "create initial metrics_worker table",
		Up: `CREATE TABLE metrics_worker (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);`,
		Down: `DROP TABLE metrics_worker;`,
	},
	{
		Version: 10,
		Table:   "metrics_idp",
		Desc:    "add index to metrics_idp table",
		Up:      `CREATE INDEX metrics_idp_type_timestamp ON metrics_idp (type, timestamp);`,
		Down:    `DROP INDEX metrics_idp_type_timestamp;`,
	},
	{
		Version: 11,
		Table:   "metrics_authz",
		Desc:    "add index to metrics_authz table",
		Up:      `CREATE INDEX metrics_authz_type_timestamp ON metrics_authz (type, timestamp);`,
		Down:    `DROP INDEX metrics_authz_type_timestamp;`,
	},
	{
		Version: 12,
		Table:   "metrics_plex",
		Desc:    "add index to metrics_plex table",
		Up:      `CREATE INDEX metrics_plex_type_timestamp ON metrics_plex (type, timestamp);`,
		Down:    `DROP INDEX metrics_plex_type_timestamp;`,
	},
	{
		Version: 13,
		Table:   "metrics_worker",
		Desc:    "add index to metrics_worker table",
		Up:      `CREATE INDEX metrics_worker_type_timestamp ON metrics_worker (type, timestamp);`,
		Down:    `DROP INDEX metrics_worker_type_timestamp;`,
	},
	{
		Version: 14,
		Table:   "metrics_console",
		Desc:    "add index to metrics_console table",
		Up:      `CREATE INDEX metrics_console_type_timestamp ON metrics_console (type, timestamp);`,
		Down:    `DROP INDEX metrics_console_type_timestamp;`,
	},
	{
		Version: 15,
		Table:   "metrics_checkattribute",
		Desc:    "create initial metrics_checkattribute table",
		Up: `CREATE TABLE metrics_checkattribute (
			id BIGINT,
			type INT,
			timestamp BIGINT,
			count INT
			);
			CREATE INDEX metrics_checkattribute_type_timestamp ON metrics_checkattribute (type, timestamp);`,
		Down: `DROP TABLE metrics_checkattribute;`,
	},
}
