package tenantdb

import "userclouds.com/infra/migrate"

//go:generate genschemas

// Migrations specifies all the migrations necessary to get IDP's DB up to today
//
//	TODO: we should use sql_safe_updates just to be safe, but introducing it now
//	requires rewriting history on migrations (esp down migrations) so will wait
//	until we do consolidation.
var Migrations = migrate.Migrations{
	{
		Version: 0,
		Table:   "tokens",
		Desc:    "create initial tokens table",
		// NB: explicitly not using IF NOT EXISTS here to ensure we fail loudly on table duplication
		Up: `CREATE TABLE tokens (
			username VARCHAR NOT NULL,
			client_id VARCHAR NOT NULL,
			raw_jwt VARCHAR,
			access_token VARCHAR UNIQUE,
			scopes VARCHAR,
			name VARCHAR,
			nickname VARCHAR,
			email VARCHAR,
			email_verified BOOLEAN,
			PRIMARY KEY (username, client_id)
			);`,
		Down: `DROP TABLE tokens;`,
	},
	{
		Version: 1,
		Table:   "authns",
		Desc:    "create initial authns table",
		Up: `CREATE TABLE authns (
			username VARCHAR NOT NULL PRIMARY KEY,
			salted_hashed_password VARCHAR NOT NULL
			);`,
		Down: `DROP TABLE authns;`,
	},
	{
		Version: 2,
		Table:   "tokens",
		Desc:    "add picture column to token claims",
		Up: `ALTER TABLE tokens
			 ADD picture VARCHAR;`,
		Down: `ALTER TABLE tokens
			 DROP COLUMN picture;`,
	},
	{
		// TODO: coalesce these migrations up front to single creates, but this saves people a DB reset for now
		Version: 3,
		Table:   "authns",
		Desc:    "recreate authns table",
		Up: `DROP TABLE authns;
			CREATE TABLE authns (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			user_id UUID NOT NULL,
			type VARCHAR NOT NULL,
			username VARCHAR NOT NULL,
			password VARCHAR NOT NULL
			);`,
		Down: `DROP TABLE authns;
		CREATE TABLE authns (
			username VARCHAR NOT NULL PRIMARY KEY,
			salted_hashed_password VARCHAR NOT NULL
			);`,
	},
	{
		Version: 4,
		Table:   "users",
		Desc:    "add users table",
		Up: `CREATE TABLE users (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			email VARCHAR NOT NULL,
			require_mfa BOOL NOT NULL
			);`,
		Down: `DROP TABLE users;`,
	},
	{
		Version: 5,
		Table:   "mfa_requests",
		Desc:    "add mfa_requests table",
		Up: `CREATE TABLE mfa_requests (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			user_id UUID NOT NULL,
			issued TIMESTAMP NOT NULL,
			code VARCHAR NOT NULL,
			username VARCHAR NOT NULL,
			client_id VARCHAR NOT NULL,
			scope VARCHAR NOT NULL
			);`,
		Down: `DROP TABLE mfa_requests;`,
	},
	{
		// NB: this isn't really a great production migration since it will break
		// the currently-deployed code, but we're early enough that I didn't want to
		// bother with multiple step migrations, and this will get collapsed before launch
		// Note also that the downward migration is schema-preserving but not truly
		// data-preserving, because why?
		Version: 6,
		Table:   "authns",
		Desc:    "move types to be ints instead of strings",
		Up: `ALTER TABLE authns ADD COLUMN type2 INT;
			UPDATE authns SET type2=1 WHERE type='up';
			UPDATE authns SET type2=2 WHERE type='google';
			ALTER TABLE authns DROP COLUMN type;
			ALTER TABLE authns RENAME COLUMN type2 TO type;
			ALTER TABLE authns ALTER COLUMN type SET NOT NULL;`,
		Down: `ALTER TABLE authns DROP COLUMN type;
			ALTER TABLE authns ADD COLUMN type VARCHAR;
			UPDATE authns SET type='up' WHERE type IS NULL;
			ALTER TABLE authns ALTER COLUMN type SET NOT NULL;`,
	},
	{
		Version: 7,
		Table:   "object_types",
		Desc:    "create object_types table",
		Up: `CREATE TABLE object_types (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			type_name VARCHAR UNIQUE NOT NULL
			);`,
		Down: `DROP TABLE object_types;`,
	},
	{
		// NOTE: in Cockroach & Postgres, UNIQUE columns have auto indices.
		Version: 8,
		Table:   "objects",
		Desc:    "create objects table",
		Up: `CREATE TABLE objects (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			alias VARCHAR UNIQUE NOT NULL,
			type_id UUID NOT NULL
			);`,
		Down: `DROP TABLE objects;`,
	},
	{
		Version: 9,
		Table:   "edge_types",
		Desc:    "create edge_types table",
		Up: `CREATE TABLE edge_types (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			type_name VARCHAR UNIQUE NOT NULL,
			source_object_type_id UUID NOT NULL,
			target_object_type_id UUID NOT NULL
			);`,
		Down: `DROP TABLE edge_types;`,
	},
	{
		// NOTE: in Cockroach, UNIQUE creates an index, so querying by source
		// object OR by target object should be fast.
		// If needed we can create an index on edge_type_id.
		// TODO: should the INDEX on target_object_id be a composite index on
		// target & source? If we have many edges to a target it may be needed.
		Version: 10,
		Table:   "edges",
		Desc:    "create edges table",
		Up: `CREATE TABLE edges (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			edge_type_id UUID NOT NULL,
			source_object_id UUID NOT NULL,
			target_object_id UUID NOT NULL,
			UNIQUE (source_object_id, target_object_id, edge_type_id),
			INDEX (target_object_id)
			);`,
		Down: `DROP TABLE edges;`,
	},
	{
		Version: 11,
		Table:   "authns",
		Desc:    "add alive column to authns table",
		Up:      `ALTER TABLE authns ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE authns DROP COLUMN alive;`,
	},
	{
		Version: 12,
		Table:   "users",
		Desc:    "add alive column to users table",
		Up:      `ALTER TABLE users ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE users DROP COLUMN alive;`,
	},
	{
		Version: 13,
		Table:   "mfa_requests",
		Desc:    "add alive column to mfa_requests table",
		Up:      `ALTER TABLE mfa_requests ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE mfa_requests DROP COLUMN alive;`,
	},
	{
		Version: 14,
		Table:   "object_types",
		Desc:    "add alive column to object_types table",
		Up:      `ALTER TABLE object_types ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE object_types DROP COLUMN alive;`,
	},
	{
		Version: 15,
		Table:   "objects",
		Desc:    "add alive column to objects table",
		Up:      `ALTER TABLE objects ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE objects DROP COLUMN alive;`,
	},
	{
		Version: 16,
		Table:   "edges",
		Desc:    "add alive column to edges table",
		Up:      `ALTER TABLE edges ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE edges DROP COLUMN alive;`,
	},
	{
		Version: 17,
		Table:   "edge_types",
		Desc:    "add alive column to edge_types table",
		Up:      `ALTER TABLE edge_types ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE edge_types DROP COLUMN alive;`,
	},
	{
		// NOTE: when upgrading, this makes a less restrictive constraint and should always succeed.
		// When downgrading, it's possible (though unlikely given current aliases) to hit collisions.
		Version: 18,
		Table:   "objects",
		Desc:    "drop unique constraint on 'alias', make (type_id, alias) uniquely indexed",
		Up: `DROP INDEX objects_alias_key CASCADE;
			ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_key UNIQUE (type_id, alias);`,
		Down: `DROP INDEX objects_type_id_alias_key CASCADE;
			ALTER TABLE objects ADD CONSTRAINT objects_alias_key UNIQUE (alias);`,
	},
	{
		// NB for the next four migrations, ALTER TABLE...ADD CONSTRAINT...UNIQUE automatically creates an index
		Version: 19,
		Table:   "object_types",
		Desc:    "fix unique constraint to support soft-delete",
		Up: `DROP INDEX object_types_type_name_key CASCADE;
			ALTER TABLE object_types ADD CONSTRAINT object_types_type_name_alive_key UNIQUE (type_name, alive);`,
		Down: `DROP INDEX object_types_type_name_alive_key CASCADE;
			ALTER TABLE object_types ADD CONSTRAINT object_types_type_name_key UNIQUE (type_name)`,
	},
	{
		Version: 20,
		Table:   "objects",
		Desc:    "fix unique constraint to support soft-delete",
		Up: `DROP INDEX objects_type_id_alias_key CASCADE;
			ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_alive_key UNIQUE (type_id, alias, alive);`,
		Down: `DROP INDEX objects_type_id_alias_alive_key CASCADE;
			ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_key UNIQUE (type_id, alias)`,
	},
	{
		Version: 21,
		Table:   "edge_types",
		Desc:    "fix unique constraint to support soft-delete",
		Up: `DROP INDEX edge_types_type_name_key CASCADE;
			ALTER TABLE edge_types ADD CONSTRAINT edge_types_type_name_alive_key UNIQUE (type_name, alive);`,
		Down: `DROP INDEX edge_types_type_name_alive_key CASCADE;
			ALTER TABLE edge_types ADD CONSTRAINT edge_types_type_name_key UNIQUE (type_name)`,
	},
	{
		Version: 22,
		Table:   "edges",
		Desc:    "fix unique constraint to support soft-delete",
		Up: `DROP INDEX edges_source_object_id_target_object_id_edge_type_id_key CASCADE;
			ALTER TABLE edges ADD CONSTRAINT edges_source_object_id_target_object_id_edge_type_id_alive_key UNIQUE (source_object_id, target_object_id, edge_type_id, alive);`,
		Down: `DROP INDEX edges_source_object_id_target_object_id_edge_type_id_alive_key CASCADE;
			ALTER TABLE edges ADD CONSTRAINT edges_source_object_id_target_object_id_edge_type_id_key UNIQUE (source_object_id, target_object_id, edge_type_id)`,
	},
	{
		Version: 23,
		Table:   "users",
		Desc:    "add profile column to users table",
		Up:      `ALTER TABLE users ADD COLUMN profile JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE users DROP COLUMN profile;`,
	},
	{
		Version: 24,
		Table:   "authns_social",
		Desc:    "create initial authns_social table",
		Up: `CREATE TABLE authns_social (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			alive BOOL DEFAULT TRUE,
			user_id UUID NOT NULL,
			type INT NOT NULL,
			oidc_sub VARCHAR NOT NULL,
			UNIQUE (type, oidc_sub)
			);`,
		Down: `DROP TABLE authns_social;`,
	},
	{
		Version: 25,
		Table:   "authns",
		Desc:    "drop authn type field",
		Up:      `ALTER TABLE authns DROP COLUMN type;`,
		// We only ever used type=1 (i.e. AuthnTypeUsernamePassword)
		Down: `ALTER TABLE authns ADD COLUMN type INT;
			UPDATE authns SET type=1;
			ALTER TABLE authns ALTER COLUMN type SET NOT NULL;`,
	},
	{
		Version: 26,
		Table:   "authns",
		Desc:    "rename authns to authns_password",
		Up:      `ALTER TABLE authns RENAME TO authns_password;`,
		Down:    `ALTER TABLE authns_password RENAME TO authns;`,
	},
	{
		// TODO: coalesce these migrations up front to single creates, but this saves people a DB reset for now
		Version: 27,
		Table:   "tokens",
		Desc:    "recreate tokens table",
		Up: `DROP TABLE tokens;
			CREATE TABLE tokens (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			alive BOOL DEFAULT TRUE,
			user_id UUID NOT NULL,
			client_id VARCHAR NOT NULL,
			raw_jwt VARCHAR NOT NULL,
			access_token VARCHAR UNIQUE,
			scopes VARCHAR NOT NULL
			);`,
		Down: `DROP TABLE tokens;
			CREATE TABLE tokens (
			username VARCHAR NOT NULL,
			client_id VARCHAR NOT NULL,
			raw_jwt VARCHAR,
			access_token VARCHAR UNIQUE,
			scopes VARCHAR,
			name VARCHAR,
			nickname VARCHAR,
			email VARCHAR,
			email_verified BOOLEAN,
			picture VARCHAR,
			PRIMARY KEY (username, client_id)
			);`,
	},
	{
		Version: 28,
		Table:   "authns_password",
		Desc:    "add unique constraint on username",
		Up:      `ALTER TABLE authns_password ADD CONSTRAINT authns_password_username_alive_key UNIQUE (username, alive);`,
		Down:    `DROP INDEX authns_password_username_alive_key CASCADE;`,
	},
	{
		Version: 29,
		Table:   "authns_password",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE authns_password ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE authns_password SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE authns_password ADD CONSTRAINT authns_password_username_deleted_key UNIQUE (username, deleted);
			ALTER TABLE authns_password DROP COLUMN alive;
			ALTER TABLE authns_password ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX authns_password_id_key CASCADE;`,
		Down: `DELETE FROM authns_password WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM authns_password GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE authns_password ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE authns_password ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE authns_password SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE authns_password ADD CONSTRAINT authns_password_username_alive_key UNIQUE (username, alive);
			ALTER TABLE authns_password DROP COLUMN deleted;`,
	},
	{
		// it's unclear to me why the (oidc_sub, type) constraint gets dropped during this migration, but
		// it only happens when we drop the deleted column in the Down migration, which is why we don't recreate
		// it in the up, and it has to be the last statement
		Version: 30,
		Table:   "authns_social",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE authns_social ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE authns_social SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE authns_social DROP COLUMN alive;
			ALTER TABLE authns_social ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX authns_social_id_key CASCADE;`,
		Down: `DELETE FROM authns_social WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM authns_social GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE authns_social ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE authns_social ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE authns_social SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE authns_social DROP COLUMN deleted;
			ALTER TABLE authns_social ADD CONSTRAINT authns_social_type_oidc_sub_key UNIQUE (oidc_sub, type);`,
	},
	{
		Version: 31,
		Table:   "edge_types",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE edge_types ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE edge_types SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE edge_types ADD CONSTRAINT edge_types_type_name_deleted_key UNIQUE (type_name, deleted);
			ALTER TABLE edge_types DROP COLUMN alive;
			ALTER TABLE edge_types ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX edge_types_id_key CASCADE;`,
		Down: `DELETE FROM edge_types WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM edge_types GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE edge_types ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE edge_types ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE edge_types SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE edge_types ADD CONSTRAINT edge_types_type_name_alive_key UNIQUE (type_name, alive);
			ALTER TABLE edge_types DROP COLUMN deleted;`,
	},
	{
		Version: 32,
		Table:   "edges",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE edges ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE edges SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE edges ADD CONSTRAINT edges_source_object_id_target_object_id_edge_type_id_deleted_key UNIQUE (source_object_id, target_object_id, edge_type_id, deleted);
			ALTER TABLE edges DROP COLUMN alive;
			ALTER TABLE edges ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX edges_id_key CASCADE;`,
		Down: `DELETE FROM edges WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM edges GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE edges ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE edges ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE edges SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE edges ADD CONSTRAINT edges_source_object_id_target_object_id_edge_type_id_alive_key UNIQUE (source_object_id, target_object_id, edge_type_id, alive);
			ALTER TABLE edges DROP COLUMN deleted;`,
	},
	{
		Version: 33,
		Table:   "mfa_requests",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE mfa_requests ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE mfa_requests SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE mfa_requests DROP COLUMN alive;
			ALTER TABLE mfa_requests ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX mfa_requests_id_key CASCADE;`,
		Down: `DELETE FROM mfa_requests WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM mfa_requests GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE mfa_requests ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE mfa_requests ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE mfa_requests SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE mfa_requests DROP COLUMN deleted;`,
	},
	{
		Version: 34,
		Table:   "object_types",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE object_types ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE object_types SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE object_types ADD CONSTRAINT object_types_type_name_deleted_key UNIQUE (type_name, deleted);
			ALTER TABLE object_types DROP COLUMN alive;
			ALTER TABLE object_types ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX object_types_id_key CASCADE;`,
		Down: `DELETE FROM object_types WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM object_types GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE object_types ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE object_types ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE object_types SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE object_types ADD CONSTRAINT object_types_type_name_alive_key UNIQUE (type_name, alive);
			ALTER TABLE object_types DROP COLUMN deleted;`,
	},
	{
		Version: 35,
		Table:   "objects",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE objects ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE objects SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_deleted_key UNIQUE (alias, type_id, deleted);
			ALTER TABLE objects DROP COLUMN alive;
			ALTER TABLE objects ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX objects_id_key CASCADE;`,
		Down: `DELETE FROM objects WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM objects GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE objects ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE objects ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE objects SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_alive_key UNIQUE (alias, type_id, alive);
			ALTER TABLE objects DROP COLUMN deleted;`,
	},
	{
		Version: 36,
		Table:   "tokens",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE tokens ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE tokens SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE tokens DROP COLUMN alive;
			ALTER TABLE tokens ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX tokens_id_key CASCADE;`,
		Down: `DELETE FROM tokens WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM tokens GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE tokens ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE tokens ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE tokens SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE tokens DROP COLUMN deleted;`,
	},
	{
		Version: 37,
		Table:   "users",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE users ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE users SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE users DROP COLUMN alive;
			ALTER TABLE users ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX users_id_key CASCADE;`,
		Down: `DELETE FROM users WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM users GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE users ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE users ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE users SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE users DROP COLUMN deleted;`,
	},
	{
		Version: 38,
		Table:   "users",
		Desc:    "copy (denormalize) email into user profile",
		Up: `UPDATE users SET profile=JSONB_SET(
			profile, ARRAY['email'], CONCAT('"', email, '"')::JSONB
		);`,
		Down: `UPDATE users SET profile=JSON_REMOVE_PATH(profile, ARRAY['email']);`,
	},
	{
		Version: 39,
		Table:   "mfa_requests",
		Desc:    "drop OIDC-specific fields from mfa_requests",
		Up: `ALTER TABLE mfa_requests DROP COLUMN client_id;
			ALTER TABLE mfa_requests DROP COLUMN scope;`,
		Down: `ALTER TABLE mfa_requests ADD COLUMN client_id VARCHAR;
			ALTER TABLE mfa_requests ADD COLUMN scope VARCHAR;
			UPDATE mfa_requests SET client_id='down_migrated_default', scope='down_migrated_default';
			ALTER TABLE mfa_requests ALTER COLUMN client_id SET NOT NULL;
			ALTER TABLE mfa_requests ALTER COLUMN scope SET NOT NULL;`,
	},
	{
		Version: 40,
		Table:   "tokens",
		Desc:    "drop tokens table from IDP since Plex manages all tokens now",
		Up:      `DROP TABLE tokens;`,
		Down: `CREATE TABLE tokens (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			user_id UUID NOT NULL,
			client_id VARCHAR NOT NULL,
			raw_jwt VARCHAR NOT NULL,
			access_token VARCHAR NULL,
			scopes VARCHAR NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			CONSTRAINT "primary" PRIMARY KEY (id ASC, deleted ASC),
			UNIQUE INDEX tokens_access_token_key (access_token ASC),
			FAMILY "primary" (id, created, updated, user_id, client_id, raw_jwt, access_token, scopes, deleted)
		)`,
	},
	{
		Version: 41,
		Table:   "oidc_login_sessions",
		Desc:    "add DB-backed storage to Plex for oidc_login_sessions",
		Up: `CREATE TABLE oidc_login_sessions (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			client_id VARCHAR NOT NULL,
			response_types VARCHAR NOT NULL,
			redirect_uri VARCHAR NOT NULL,
			state VARCHAR NOT NULL,
			scopes VARCHAR NOT NULL,
			nonce VARCHAR NOT NULL,
			social_provider INT8 NOT NULL,
			mfa_state_id UUID NOT NULL,
			otp_state_id UUID NOT NULL,
			pkce_state_id UUID NOT NULL
		)`,
		Down: `DROP TABLE oidc_login_sessions;`,
	},
	{
		Version: 42,
		Table:   "mfa_states",
		Desc:    "add DB-backed storage to Plex for mfa_states",
		Up: `CREATE TABLE mfa_states (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			session_id UUID NOT NULL,
			token VARCHAR NOT NULL,
			provider UUID NOT NULL
		)`,
		Down: `DROP TABLE mfa_states;`,
	},
	{
		Version: 43,
		Table:   "otp_states",
		Desc:    "add DB-backed storage to Plex for otp_states",
		Up: `CREATE TABLE otp_states (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			session_id UUID NOT NULL,
			email VARCHAR NOT NULL,
			code VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOLEAN NOT NULL,
			purpose INT8 NOT NULL
		)`,
		Down: `DROP TABLE otp_states;`,
	},
	{
		Version: 44,
		Table:   "pkce_states",
		Desc:    "add DB-backed storage to Plex for pkce_states",
		Up: `CREATE TABLE pkce_states (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			session_id UUID NOT NULL,
			code_challenge VARCHAR NOT NULL,
			method INT8 NOT NULL,
			used BOOLEAN NOT NULL
		)`,
		Down: `DROP TABLE pkce_states;`,
	},
	{
		Version: 45,
		Table:   "plex_tokens",
		Desc:    "add DB-backed storage to Plex for plex_tokens",
		Up: `CREATE TABLE plex_tokens (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			client_id VARCHAR NOT NULL,
			auth_code VARCHAR NOT NULL,
			access_token VARCHAR NOT NULL,
			id_token VARCHAR NOT NULL,
			idp_subject VARCHAR NOT NULL,
			scopes VARCHAR NOT NULL,
			session_id UUID NOT NULL,
			UNIQUE (auth_code),
			UNIQUE (access_token)
		)`,
		Down: `DROP TABLE plex_tokens;`,
	},
	{
		Version: 46,
		Table:   "reset_tokens",
		Desc:    "add DB-backed storage to Plex for reset_tokens",
		Up: `CREATE TABLE reset_tokens (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			email VARCHAR NOT NULL,
			token VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOLEAN NOT NULL,
			client_id VARCHAR NOT NULL,
			session_id UUID NOT NULL,
			UNIQUE (token)
		)`,
		Down: `DROP TABLE reset_tokens;`,
	},
	{
		Version: 47,
		Table:   "reset_tokens",
		Desc:    "drop reset_tokens since we use otp_states now instead",
		Up:      `DROP TABLE reset_tokens;`,
		Down: `CREATE TABLE reset_tokens (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			email VARCHAR NOT NULL,
			token VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOLEAN NOT NULL,
			client_id VARCHAR NOT NULL,
			session_id UUID NOT NULL,
			UNIQUE (token)
		)`,
	},
	{
		// NOTE: user ID is a string not a UUID because this is a Plex table, which can currently interoperate
		// with different IDPs that use different formats for user ID.
		Version: 48,
		Table:   "otp_states",
		Desc:    "add UserID to otp_states",
		Up:      `ALTER TABLE otp_states ADD COLUMN user_id VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE otp_states DROP COLUMN user_id;`,
	},
	{
		Version: 49,
		Table:   "users",
		Desc:    "drop email column from users to re-normalize it (it's in the user profile)",
		Up:      `ALTER TABLE users DROP COLUMN email;`,
		Down: `ALTER TABLE users ADD COLUMN email VARCHAR;
				UPDATE users SET email=profile->>'email'::VARCHAR;
				ALTER TABLE users ALTER COLUMN email SET NOT NULL;`,
	},
	{
		Version: 50,
		Table:   "authns_social",
		Desc:    "add index on user_id to authns_social",
		Up:      `CREATE INDEX ON authns_social (user_id);`,
		Down:    `DROP INDEX authns_social_user_id_idx CASCADE;`,
	},
	{
		Version: 51,
		Table:   "authns_password",
		Desc:    "add index on user_id to authns_password",
		Up:      `CREATE INDEX ON authns_password (user_id);`,
		Down:    `DROP INDEX authns_password_user_id_idx CASCADE;`,
	},
	{
		Version: 52,
		Table:   "users",
		Desc:    "add extended/custom profile column to users table",
		Up:      `ALTER TABLE users ADD COLUMN profile_ext JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE users DROP COLUMN profile_ext;`,
	},
	{
		Version: 53,
		Table:   "oidc_login_sessions",
		Desc:    "add delegation state reference to oidc sessions",
		Up:      `ALTER TABLE oidc_login_sessions ADD COLUMN delegation_state_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down:    `ALTER TABLE oidc_login_sessions DROP COLUMN delegation_state_id;`,
	},
	{
		Version: 54,
		Table:   "delegation_states",
		Desc:    "add new delegation state table",
		Up: `CREATE TABLE delegation_states (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			authenticated_user_id VARCHAR NOT NULL
		)`,
		Down: `DROP TABLE delegation_states;`,
	},
	{
		Version: 55,
		Table:   "delegation_invites",
		Desc:    "add new delegation invites table",
		Up: `CREATE TABLE delegation_invites (
			id UUID NOT NULL PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			client_id STRING NOT NULL,
			invited_to_account_id STRING NOT NULL
		)`,
		Down: `DROP TABLE delegation_invites;`,
	},
	{
		Version: 56,
		Table:   "edge_types",
		Desc:    "add attributes column to edge_types table",
		Up:      `ALTER TABLE edge_types ADD COLUMN attributes JSONB NOT NULL DEFAULT '[]';`,
		Down:    `ALTER TABLE edge_types DROP COLUMN attributes;`,
	},
	{
		Version: 57,
		Table:   "authns_social",
		Desc:    "fix unique constraint on type & oidc subject to include deleted column",
		Up: `DROP INDEX authns_social_type_oidc_sub_key CASCADE;
			ALTER TABLE authns_social ADD CONSTRAINT authns_social_type_oidc_sub_deleted_key UNIQUE (type, oidc_sub, deleted);`,
		Down: `DROP INDEX authns_social_type_oidc_sub_deleted_key CASCADE;
			ALTER TABLE authns_social ADD CONSTRAINT authns_social_type_oidc_sub_key UNIQUE (type, oidc_sub);`,
	},
	{
		Version: 58,
		Table:   "auditlog",
		Desc:    "create initial auditlog table",
		Up: `CREATE TABLE auditlog (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			actor_id STRING NOT NULL,
			type STRING,
			payload JSONB NOT NULL,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE auditlog;`,
	},
	{
		Version: 59,
		Table:   "generation_policies",
		Desc:    "table for holding GenerationPolicies",
		Up: `CREATE TABLE generation_policies (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			function VARCHAR NOT NULL,
			parameters VARCHAR,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE generation_policies`,
	},
	{
		Version: 60,
		Table:   "access_policies",
		Desc:    "create access policies for tokenizer",
		Up: `CREATE TABLE access_policies (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			version INT NOT NULL,
			function VARCHAR NOT NULL,
			parameters VARCHAR,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE access_policies;`,
	},
	{
		Version: 61,
		Table:   "token_records",
		Desc:    "create table to store tokens",
		Up: `CREATE TABLE token_records (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			data JSONB NOT NULL,
			token JSONB NOT NULL,
			generation_policy_id UUID NOT NULL,
			access_policy_id UUID NOT NULL,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE token_records;`,
	},
	{
		Version: 62,
		Table:   "access_policies",
		Desc:    "unique (function, parameters)",
		Up:      `CREATE UNIQUE INDEX access_policies_function_parameters_version_deleted ON access_policies (function, parameters, version, deleted);`,
		Down:    `DROP INDEX access_policies_function_parameters_version_deleted`,
	},
	{
		Version: 63,
		Table:   "generation_policies",
		Desc:    "unique (function, parameters)",
		Up:      `CREATE UNIQUE INDEX generation_policies_function_parameters_deleted ON generation_policies (function, parameters, deleted);`,
		Down:    `DROP INDEX generation_policies_function_parameters_deleted`,
	},
	{
		// Note: this migration is data-lossy but it's ok since tokenizer isn't in production yet
		Version: 64,
		Table:   "token_records",
		Desc:    "add index on token",
		Up: `ALTER TABLE token_records DROP COLUMN token;
			ALTER TABLE token_records ADD COLUMN token VARCHAR NOT NULL;
			CREATE INDEX token_records_token_deleted ON token_records (token, deleted);`,
		Down: `DROP INDEX token_records_token_deleted;
			ALTER TABLE token_records DROP COLUMN token;
			ALTER TABLE token_records ADD COLUMN token JSONB NOT NULL;`,
	},
	{
		Version: 65,
		Table:   "access_policies",
		Desc:    "fix primary key for APs to include version so we don't overwrite earlier versions ... oops",
		Up: `ALTER TABLE access_policies ALTER PRIMARY KEY USING COLUMNS (id, version, deleted);
			DROP INDEX access_policies_id_deleted_key CASCADE;`,
		Down: `ALTER TABLE access_policies ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX access_policies_id_version_deleted_key CASCADE;`,
	},
	{
		// Note: this migration is data-lossy but it's ok since tokenizer isn't in production yet
		Version: 66,
		Table:   "token_records",
		Desc:    "might as well make data non-JSONB as well",
		Up: `ALTER TABLE token_records DROP COLUMN data;
			ALTER TABLE token_records ADD COLUMN data VARCHAR NOT NULL;`,
		Down: `ALTER TABLE token_records DROP COLUMN data;
			ALTER TABLE token_records ADD COLUMN data JSONB NOT NULL;`,
	},
	{
		// TODO - this is not a great way to do this - instead we should store the policy ver, auth ver and user store ver
		Version: 67,
		Table:   "access_policies",
		Desc:    "force policies to be re-provision",
		Up:      `/* noop */`,
		Down:    `/* noop */`,
	},
	{
		Version: 68,
		Table:   "access_policies",
		Desc:    "add policy name",
		Up:      `ALTER TABLE access_policies ADD COLUMN name VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE access_policies DROP COLUMN name;`,
	},
	{
		Version: 69,
		Table:   "generation_policies",
		Desc:    "add policy name",
		Up:      `ALTER TABLE generation_policies ADD COLUMN name VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE generation_policies DROP COLUMN name;`,
	},
	{
		Version: 70,
		Table:   "token_records",
		Desc:    "fix index on token to be unique :/",
		Up: `DROP INDEX token_records_token_deleted;
			CREATE UNIQUE INDEX token_records_token_deleted ON token_records (token, deleted);`,
		Down: `DROP INDEX token_records_token_deleted;
			CREATE INDEX token_records_token_deleted ON token_records (token, deleted);`,
	},
	{
		Version: 71,
		Table:   "token_records",
		Desc:    "change tokens index to primary for faster resolve perf",
		Up:      `ALTER TABLE token_records ALTER PRIMARY KEY USING COLUMNS (token, deleted);`,
		// NB: we drop two indices in the down migration because ALTER PRIMARY KEY keeps the old PK
		// as an UNIQUE index (https://www.cockroachlabs.com/docs/stable/alter-primary-key.html).
		// On the up path this is useful since we still want indexed access to ID, but on the down
		// path we need to leave things in the same state we found them (and since we are making
		// (id, deleted) the PK again, we preserve indexed access to ID, and we already have
		// a secondary index from migration 70 above on (token, deleted))
		Down: `ALTER TABLE token_records ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX token_records_id_deleted_key CASCADE;
			DROP INDEX token_records_token_deleted_key CASCADE;`,
	},
	{
		Version: 72,
		Table:   "access_policies",
		Desc:    "initialize empty names with the id",
		Up: `UPDATE access_policies SET name=id::text WHERE name='';
			 ALTER TABLE access_policies ADD CONSTRAINT access_policies_name UNIQUE (name, version, deleted);`,
		Down: `DROP INDEX access_policies_name CASCADE;`,
	},
	{
		Version: 73,
		Table:   "generation_policies",
		Desc:    "initialize empty names with the id",
		Up: `UPDATE generation_policies SET name=id::text WHERE name='';
			 ALTER TABLE generation_policies ADD CONSTRAINT generation_policies_name UNIQUE (name, deleted);`,
		Down: `DROP INDEX generation_policies_name CASCADE;`,
	},
	{
		Version: 74,
		Table:   "users",
		Desc:    "add external_alias to users",
		Up: `ALTER TABLE users ADD COLUMN external_alias VARCHAR UNIQUE;
			CREATE INDEX users_external_alias_deleted ON users (external_alias, deleted);`,
		Down: `ALTER TABLE users DROP COLUMN external_alias;`,
	},
	{
		Version: 75,
		Table:   "users",
		Desc:    "fix external_alias unique constraint to include deleted, and we don't need the old index anymore",
		Up: `DROP INDEX users_external_alias_key CASCADE;
			DROP INDEX users_external_alias_deleted CASCADE;
			ALTER TABLE users ADD CONSTRAINT users_external_alias_deleted_key UNIQUE (external_alias, deleted);`,
		Down: `DROP INDEX users_external_alias_deleted_key CASCADE;
			ALTER TABLE users ADD CONSTRAINT users_external_alias_key UNIQUE (external_alias);
			CREATE INDEX users_external_alias_deleted ON users (external_alias, deleted);`,
	},
	{
		Version: 76,
		Table:   "plex_tokens",
		Desc:    "add column to store underlying IDP token for social logins",
		Up:      `ALTER TABLE plex_tokens ADD COLUMN underlying_token VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE plex_tokens DROP COLUMN underlying_token;`,
	},
	{
		Version: 77,
		Table:   "plex_tokens",
		Desc:    "Add refresh token to plex tokens",
		Up:      `ALTER TABLE plex_tokens ADD COLUMN refresh_token VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE plex_tokens DROP COLUMN refresh_token;`,
	},
	{
		Version: 78,
		Table:   "oidc_login_sessions",
		Desc:    "Add column to store temporary provider information when user is adding a new authn method to their account",
		Up:      `ALTER TABLE oidc_login_sessions ADD COLUMN add_authn_provider_data JSONB NOT NULL DEFAULT '{}'::JSONB;`,
		Down:    `ALTER TABLE oidc_login_sessions DROP COLUMN add_authn_provider_data;`,
	},
	{
		Version: 79,
		Table:   "edges",
		Desc:    "Create edges between GBAC-based groups and users",
		Up: `INSERT into edges (id, created, updated, edge_type_id, source_object_id, target_object_id, deleted)
				SELECT gen_random_uuid(), now():::TIMESTAMP, now():::TIMESTAMP, '237aba41-90bd-47de-a6cd-bf75c3c76b74', target_object_id, source_object_id, '0001-01-01 00:00:00':::TIMESTAMP
				FROM edges WHERE edge_type_id = '60b69666-4a8a-4eb3-94dd-621298fb365d' AND deleted = '0001-01-01 00:00:00':::TIMESTAMP
				ON CONFLICT (source_object_id, target_object_id, edge_type_id, deleted) DO UPDATE SET updated = excluded.updated;
			INSERT into edges (id, created, updated, edge_type_id, source_object_id, target_object_id, deleted)
				SELECT gen_random_uuid(), now():::TIMESTAMP, now():::TIMESTAMP, 'e5eb5062-8f08-4cd8-a43e-b4a81ab55f50', target_object_id, source_object_id, '0001-01-01 00:00:00':::TIMESTAMP
				FROM edges WHERE edge_type_id = '1eec16ec-6130-4f9e-a51f-21bc19b20d8f' AND deleted = '0001-01-01 00:00:00':::TIMESTAMP
				ON CONFLICT (source_object_id, target_object_id, edge_type_id, deleted) DO UPDATE SET updated = excluded.updated;
			DELETE from edge_types where id = '60b69666-4a8a-4eb3-94dd-621298fb365d';
			DELETE from edge_types where id = '1eec16ec-6130-4f9e-a51f-21bc19b20d8f';
			/* no-match-cols-vals lint-system-table */`,
		Down: `DELETE FROM edges WHERE edge_type_id = '237aba41-90bd-47de-a6cd-bf75c3c76b74' OR edge_type_id = 'e5eb5062-8f08-4cd8-a43e-b4a81ab55f50';`,
	},
	{
		Version: 80,
		Table:   "organizations",
		Desc:    "Add organizations table for authz",
		Up: `CREATE TABLE organizations (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE organizations;`,
	},
	{
		Version: 81,
		Table:   "objects",
		Desc:    "Add organization_id to objects",
		Up:      `ALTER TABLE objects ADD COLUMN organization_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';`,
		Down:    `ALTER TABLE objects DROP COLUMN organization_id;`,
	},
	{
		Version: 82,
		Table:   "edge_types",
		Desc:    "Add organization_id to edge_types",
		Up:      `ALTER TABLE edge_types ADD COLUMN organization_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';`,
		Down:    `ALTER TABLE edge_types DROP COLUMN organization_id;`,
	},
	{
		Version: 83,
		Table:   "idp_sync_runs",
		Desc:    "Add idp_sync_runs to keep track of sync progress",
		Up: `CREATE TABLE idp_sync_runs (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			active_provider_id UUID NOT NULL,
			follower_provider_ids UUID[] NOT NULL DEFAULT '{}'::UUID[],
			since TIMESTAMP NOT NULL,
			until TIMESTAMP NOT NULL,
			error VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted),
			INDEX (active_provider_id, deleted)
			);`,
		Down: `DROP TABLE idp_sync_runs;`,
	},
	{
		Version: 84,
		Table:   "idp_sync_records",
		Desc:    "Add idp_sync_records to keep track of each sync action",
		Up: `CREATE TABLE idp_sync_records (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			sync_run_id UUID NOT NULL,
			user_id VARCHAR NOT NULL,
			error VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE idp_sync_records;`,
	},
	{
		Version: 85,
		Table:   "objects",
		Desc:    "Drop not NULL constraint from alias column",
		Up:      `ALTER TABLE objects ALTER COLUMN alias DROP NOT NULL;`,
		Down:    `ALTER TABLE objects ALTER COLUMN alias SET NOT NULL;`,
	},
	{
		Version: 86,
		Table:   "users",
		Desc:    "Add organization_id to users",
		Up:      `ALTER TABLE users ADD COLUMN organization_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';`,
		Down:    `ALTER TABLE users DROP COLUMN organization_id;`,
	},
	{
		Version: 87,
		Table:   "users",
		Desc:    "Split up users into appropraite organizations",
		Up: `UPDATE users SET organization_id = id_org.group_id FROM users u JOIN
				(SELECT users.id user_id, edges.target_object_id group_id FROM users JOIN edges ON users.id = edges.source_object_id
					WHERE users.deleted = '0001-01-01 00:00:00':::TIMESTAMP AND
						(edge_type_id = '237aba41-90bd-47de-a6cd-bf75c3c76b74' OR edge_type_id = 'e5eb5062-8f08-4cd8-a43e-b4a81ab55f50') AND
						edges.deleted='0001-01-01 00:00:00':::TIMESTAMP AND
						target_object_id NOT IN ('1ee4497e-c326-4068-94ed-3dcdaaaa53bc' /* dev */, 'c8564de2-6d04-4706-aef0-4e905b7d7196' /* prod */,
						'74f313fc-806c-4ae2-abfd-76a972c29a2d' /* staging */, 'dd2d7aa6-5b9a-4941-baf4-064efe6083f2' /* debug */)) as id_org
				ON (u.id = id_org.user_id AND u.deleted = '0001-01-01 00:00:00':::TIMESTAMP)
				WHERE users.id = id_org.user_id AND users.deleted = '0001-01-01 00:00:00':::TIMESTAMP;
			UPDATE users SET organization_id = id_org.group_id FROM users u JOIN
				(SELECT users.id user_id, edges.target_object_id group_id FROM users JOIN edges ON users.id = edges.source_object_id
					WHERE users.deleted = '0001-01-01 00:00:00':::TIMESTAMP AND
						(edge_type_id = '237aba41-90bd-47de-a6cd-bf75c3c76b74' OR edge_type_id = 'e5eb5062-8f08-4cd8-a43e-b4a81ab55f50') AND
						edges.deleted='0001-01-01 00:00:00':::TIMESTAMP AND
						target_object_id IN ('1ee4497e-c326-4068-94ed-3dcdaaaa53bc' /* dev */, 'c8564de2-6d04-4706-aef0-4e905b7d7196' /* prod */,
						'74f313fc-806c-4ae2-abfd-76a972c29a2d' /* staging */, 'dd2d7aa6-5b9a-4941-baf4-064efe6083f2' /* debug */)) as id_org
				ON (u.id = id_org.user_id AND u.deleted = '0001-01-01 00:00:00':::TIMESTAMP)
				WHERE users.id = id_org.user_id AND users.deleted = '0001-01-01 00:00:00':::TIMESTAMP;`,
		Down: `UPDATE users SET organization_id = '00000000-0000-0000-0000-000000000000';`,
	},
	{
		Version: 88,
		Table:   "idp_sync_runs",
		Desc:    "Add record counts to idp_sync_runs",
		Up: `ALTER TABLE idp_sync_runs ADD COLUMN total_records INTEGER NOT NULL DEFAULT 0,
			ADD COLUMN failed_records INTEGER NOT NULL DEFAULT 0;`,
		Down: `ALTER TABLE idp_sync_runs DROP COLUMN total_records,
			DROP COLUMN failed_records;`,
	},
	{
		Version: 89,
		Table:   "accessors",
		Desc:    "create accessors table for userstore",
		Up: `CREATE TABLE accessors (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			version INT NOT NULL,
			column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[],
			access_policy_id UUID NOT NULL,
			transformation_policy_id UUID NOT NULL,
			PRIMARY KEY (id, version, deleted),
			UNIQUE (name, version, deleted)
			);`,
		Down: `DROP TABLE accessors;`,
	},
	{
		Version: 90,
		Table:   "columns",
		Desc:    "create columns table for userstore",
		Up: `CREATE TABLE columns (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			type INT NOT NULL,
			default_value VARCHAR NOT NULL,
			uniq BOOLEAN NOT NULL,
			PRIMARY KEY (id, deleted),
			UNIQUE (name, deleted)
			);`,
		Down: `DROP TABLE columns;`,
	},
	{
		Version: 91,
		Table:   "mutators",
		Desc:    "create mutators table for userstore",
		Up: `CREATE TABLE mutators (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			version INT NOT NULL,
			column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[],
			access_policy_id UUID NOT NULL,
			validation_policy_id UUID NOT NULL,
			PRIMARY KEY (id, version, deleted),
			UNIQUE (name, version, deleted)
			);`,
		Down: `DROP TABLE mutators;`,
	},
	{
		Version: 92,
		Table:   "users",
		Desc:    "delete extended/custom profile column from users table",
		Up:      `ALTER TABLE users DROP COLUMN profile_ext;`,
		Down:    `ALTER TABLE users ADD COLUMN profile_ext JSONB NOT NULL DEFAULT '{}';`,
	},
	{
		Version: 93,
		Table:   "custom_pages",
		Desc:    "add custom_pages table",
		Up: `CREATE TABLE custom_pages (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			app_id UUID NOT NULL,
			page_name VARCHAR NOT NULL,
			page_source VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted));`,
		Down: `DROP TABLE custom_pages;`,
	},
	{
		Version: 94,
		Table:   "accessors",
		Desc:    "add selector_config to accessors table",
		Up:      `ALTER TABLE accessors ADD COLUMN selector_config JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE accessors DROP COLUMN selector_config;`,
	},
	{
		Version: 95,
		Table:   "mutators",
		Desc:    "add selector_config to mutators table",
		Up:      `ALTER TABLE mutators ADD COLUMN selector_config JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE mutators DROP COLUMN selector_config;`,
	},
	{
		Version: 96,
		Table:   "accessors",
		Desc:    "Change column_ids to column_names in accessors table",
		Up: `ALTER TABLE accessors DROP COLUMN column_ids;
				ALTER TABLE accessors ADD COLUMN column_names VARCHAR[] NOT NULL DEFAULT '{}'::VARCHAR[];
				DELETE FROM accessors WHERE created < '2023-03-12 15:00:00':::TIMESTAMP;`,
		Down: `ALTER TABLE accessors DROP COLUMN column_names;
				ALTER TABLE accessors ADD COLUMN column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];`,
	},
	{
		Version: 97,
		Table:   "mutators",
		Desc:    "Change column_ids to column_names in mutators table",
		Up: `ALTER TABLE mutators DROP COLUMN column_ids;
				ALTER TABLE mutators ADD COLUMN column_names VARCHAR[] NOT NULL DEFAULT '{}'::VARCHAR[];
				DELETE FROM mutators WHERE created < '2023-03-12 15:00:00':::TIMESTAMP;`,
		Down: `ALTER TABLE mutators DROP COLUMN column_names;
				ALTER TABLE mutators ADD COLUMN column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];`,
	},
	{
		Version: 98,
		Table:   "accessors",
		Desc:    "Re-add column_ids to minimize downtime",
		Up:      `ALTER TABLE accessors ADD COLUMN column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];`,
		Down:    `ALTER TABLE accessors DROP COLUMN column_ids;`,
	},
	{
		Version: 99,
		Table:   "mutators",
		Desc:    "Re-add column_ids to minimize downtime",
		Up:      `ALTER TABLE mutators ADD COLUMN column_ids UUID[] NOT NULL DEFAULT '{}'::UUID[];`,
		Down:    `ALTER TABLE mutators DROP COLUMN column_ids;`,
	},
	{
		Version: 100,
		Table:   "acme_orders",
		Desc:    "add acme_orders table",
		Up: `CREATE TABLE acme_orders (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			status VARCHAR NOT NULL,
			url VARCHAR NOT NULL,
			host VARCHAR NOT NULL,
			token VARCHAR NOT NULL,
			challenge_url VARCHAR NOT NULL,
			tenant_url_id UUID NOT NULL,
			PRIMARY KEY (id, deleted)
		);`,
		Down: `DROP TABLE acme_orders;`,
	},
	{
		Version: 101,
		Table:   "acme_certificates",
		Desc:    "add acme_certificates table",
		Up: `CREATE TABLE acme_certificates (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			status VARCHAR NOT NULL,
			order_id UUID NOT NULL,
			private_key VARCHAR NOT NULL,
			certificate VARCHAR NOT NULL,
			certificate_chain VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted)
		);`,
		Down: `DROP TABLE acme_certificates;`,
	},
	{
		Version: 102,
		Table:   "users",
		Desc:    "remove profile column from users table",
		Up:      `ALTER TABLE users DROP COLUMN profile;`,
		Down:    `ALTER TABLE users ADD COLUMN profile JSONB NOT NULL DEFAULT '{}';`,
	},
	{
		Version: 103,
		Table:   "mfa_requests",
		Desc:    "add channel and supported_channels to mfa_requests table",
		Up: `ALTER TABLE mfa_requests ADD COLUMN channel JSONB NOT NULL DEFAULT '{}';
			ALTER TABLE mfa_requests ADD COLUMN supported_channels JSONB NOT NULL DEFAULT '{}';`,
		Down: `ALTER TABLE mfa_requests DROP COLUMN channel;
			ALTER TABLE mfa_requests DROP COLUMN supported_channels;`,
	},
	{
		Version: 104,
		Table:   "mfa_states",
		Desc:    "add supported_channels to mfa_states table",
		Up:      `ALTER TABLE mfa_states ADD COLUMN supported_channels JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE mfa_states DROP COLUMN supported_channels;`,
	},
	{
		// This migration cleans up old versions of the PK name in order for the next migration to work :)
		Version: 105,
		Table:   "edges",
		Desc:    "fix edges PK name",
		Up: `ALTER TABLE edges DROP CONSTRAINT IF EXISTS edges_pkey;
			ALTER TABLE edges ADD CONSTRAINT IF NOT EXISTS "primary" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 106,
		Table:   "edges",
		Desc:    "fix edges PK columns",
		Up: `ALTER TABLE edges DROP CONSTRAINT "primary";
			ALTER TABLE edges ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE edges DROP CONSTRAINT "primary";
			ALTER TABLE edges ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 107,
		Table:   "columns",
		Desc:    "add attributes to columns",
		Up:      `ALTER TABLE columns ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE columns DROP COLUMN attributes;`,
	},
	{
		Version: 108,
		Table:   "columns",
		Desc:    "add indexed to columns",
		Up: `ALTER TABLE columns ADD COLUMN index_type INT NOT NULL DEFAULT 0;
				 ALTER TABLE columns ALTER COLUMN uniq SET DEFAULT false;`,
		Down: `ALTER TABLE columns DROP COLUMN index_type;
				 ALTER TABLE columns ALTER COLUMN uniq DROP DEFAULT;`,
	},
	{
		Version: 109,
		Table:   "organizations",
		Desc:    "add region to organizations",
		Up:      `ALTER TABLE organizations ADD COLUMN region VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE organizations DROP COLUMN region;`,
	},
	{
		Version: 110,
		Table:   "columns",
		Desc:    "remove uniq from columns",
		Up:      `ALTER TABLE columns DROP COLUMN uniq;`,
		Down:    `ALTER TABLE columns ADD COLUMN uniq BOOLEAN NOT NULL DEFAULT false;`,
	},
	{
		Version: 111,
		Table:   "acme_certificates",
		Desc:    "add not_after to acme_certificates",
		Up:      `ALTER TABLE acme_certificates ADD COLUMN not_after TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP;`,
		Down:    `ALTER TABLE acme_certificates DROP COLUMN not_after;`,
	},
	{
		Version: 112,
		Table:   "objects",
		Desc:    "Add organization_id to objects table unique key",
		Up: `ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_organization_id_deleted_key UNIQUE (alias, type_id, organization_id, deleted);
			DROP INDEX IF EXISTS objects_type_id_alias_deleted_key CASCADE;`,
		Down: `ALTER TABLE objects ADD CONSTRAINT objects_type_id_alias_deleted_key UNIQUE (alias, type_id, deleted);
			DROP INDEX IF EXISTS objects_type_id_alias_organization_id_deleted_key CASCADE;`,
	},
	{
		Version: 113,
		Table:   "organizations",
		Desc:    "Add unique constraint to organization names",
		Up:      `ALTER TABLE organizations ADD CONSTRAINT organizations_name_key UNIQUE (deleted, name);`,
		Down:    `DROP INDEX organizations_name_key CASCADE;`,
	},
	{
		Version: 114,
		Table:   "saml_sessions",
		Desc:    "create table to track SAML sessions",
		Up: `CREATE TABLE saml_sessions (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			expire_time TIMESTAMP NOT NULL,
			_index VARCHAR NOT NULL,
			name_id VARCHAR NOT NULL,
			name_id_format VARCHAR NOT NULL,
			subject_id VARCHAR NOT NULL,
			groups JSONB NOT NULL DEFAULT '[]':::JSONB,
			user_name VARCHAR NOT NULL,
			user_email VARCHAR NOT NULL,
			user_common_name VARCHAR NOT NULL,
			user_surname VARCHAR NOT NULL,
			user_given_name VARCHAR NOT NULL,
			user_scoped_affiliation VARCHAR NOT NULL,
			custom_attributes JSONB NOT NULL DEFAULT '[]':::JSONB,
			state VARCHAR NOT NULL,
			relay_state VARCHAR NOT NULL,
			request_buffer BYTEA NOT NULL,
			PRIMARY KEY (deleted, id)
		);`,
		Down: `DROP TABLE saml_sessions;`,
	},
	{
		Version: 115,
		Table:   "mfa_requests",
		Desc:    "drop username, channel, and supported_channels NOT NULL constraints for mfa_requests table",
		Up: `ALTER TABLE mfa_requests ALTER COLUMN username DROP NOT NULL;
			ALTER TABLE mfa_requests ALTER COLUMN channel DROP NOT NULL;
			ALTER TABLE mfa_requests ALTER COLUMN supported_channels DROP NOT NULL;`,
		Down: `UPDATE mfa_requests SET username='unused';
			ALTER TABLE mfa_requests ALTER COLUMN username SET NOT NULL;
			ALTER TABLE mfa_requests ALTER COLUMN channel SET NOT NULL;
			ALTER TABLE mfa_requests ALTER COLUMN supported_channels SET NOT NULL;`,
	},
	{
		Version: 116,
		Table:   "users",
		Desc:    "drop require_mfa NOT NULL constraint for users table",
		Up:      `ALTER TABLE users ALTER COLUMN require_mfa DROP NOT NULL;`,
		Down: `UPDATE users SET require_mfa=false;
			ALTER TABLE users ALTER COLUMN require_mfa SET NOT NULL;`,
	},
	{
		Version: 117,
		Table:   "mfa_requests",
		Desc:    "add channel_id and supported_channel_types to mfa_requests table",
		Up: `ALTER TABLE mfa_requests ADD COLUMN channel_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE mfa_requests ADD COLUMN supported_channel_types JSONB NOT NULL DEFAULT '{}';`,
		Down: `ALTER TABLE mfa_requests DROP COLUMN channel_id;
			ALTER TABLE mfa_requests DROP COLUMN supported_channel_types;`,
	},
	{
		Version: 118,
		Table:   "user_mfa_configuration",
		Desc:    "add user_mfa_configuration table",
		Up: `CREATE TABLE user_mfa_configuration(
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			last_evaluated TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			mfa_channels JSONB NOT NULL DEFAULT '{}':::JSONB,
			CONSTRAINT user_mfa_configuration_pkey PRIMARY KEY (id ASC, deleted ASC)
		     );`,
		Down: `DROP TABLE user_mfa_configuration;`,
	},
	{
		Version: 119,
		Table:   "mfa_states",
		Desc:    "add channel_id, purpose, challenge_state, evaluate_supported_channels to mfa_states table",
		Up: `ALTER TABLE mfa_states ADD COLUMN channel_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE mfa_states ADD COLUMN purpose INT8 NOT NULL DEFAULT 0;
			ALTER TABLE mfa_states ADD COLUMN challenge_state INT8 NOT NULL DEFAULT 0;
			ALTER TABLE mfa_states ADD COLUMN evaluate_supported_channels BOOL NOT NULL DEFAULT false;`,
		Down: `ALTER TABLE mfa_states DROP COLUMN channel_id;
			ALTER TABLE mfa_states DROP COLUMN purpose;
			ALTER TABLE mfa_states DROP COLUMN challenge_state;
			ALTER TABLE mfa_states DROP COLUMN evaluate_supported_channels;`,
	},
	{
		Version: 120,
		Table:   "oidc_login_sessions",
		Desc:    "add plex_token_id, mfa_channel_states to oidc_login_sessions table",
		Up: `ALTER TABLE oidc_login_sessions ADD COLUMN plex_token_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE oidc_login_sessions ADD COLUMN mfa_channel_states JSONB NOT NULL DEFAULT '{}';`,
		Down: `ALTER TABLE oidc_login_sessions DROP COLUMN plex_token_id;
			ALTER TABLE oidc_login_sessions DROP COLUMN mfa_channel_states;`,
	},
	{
		Version: 121,
		Table:   "columns",
		Desc:    "Add is_array to columns table",
		Up:      `ALTER TABLE columns ADD COLUMN is_array BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE columns DROP COLUMN is_array;`,
	},
	{
		Version: 122,
		Table:   "users",
		Desc:    "Add _version to users table",
		Up:      `ALTER TABLE users ADD COLUMN _version INT NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE users DROP COLUMN _version;`,
	},
	{
		Version: 123,
		Table:   "authns_social",
		Desc:    "add oidc_issuer_url column to authns_social",
		Up:      `ALTER TABLE authns_social ADD COLUMN oidc_issuer_url VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE authns_social DROP COLUMN oidc_issuer_url;`,
	},
	{
		Version: 124,
		Table:   "authns_social",
		Desc:    "incorporate oidc_issuer_url into authns_social index",
		Up: `DROP INDEX authns_social_type_oidc_sub_deleted_key CASCADE;
			ALTER TABLE authns_social
			ADD CONSTRAINT authns_social_type_issuer_url_oidc_sub_deleted_key
			UNIQUE (type ASC, oidc_issuer_url ASC, oidc_sub ASC, deleted ASC);`,
		Down: `DROP INDEX authns_social_type_issuer_url_oidc_sub_deleted_key CASCADE;
			ALTER TABLE authns_social
			ADD CONSTRAINT authns_social_type_oidc_sub_deleted_key
			UNIQUE (type ASC, oidc_sub ASC, deleted ASC);`,
	},
	{
		Version: 125,
		Table:   "oidc_login_sessions",
		Desc:    "add oidc_issuer_url column to oidc_login_sessions",
		Up:      `ALTER TABLE oidc_login_sessions ADD COLUMN oidc_issuer_url VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE oidc_login_sessions DROP COLUMN oidc_issuer_url;`,
	},
	{
		Version: 126,
		Table:   "mfa_requests",
		Desc:    "drop username, channel, and supported_channels from mfa_requests table",
		Up: `ALTER TABLE mfa_requests DROP COLUMN username;
			ALTER TABLE mfa_requests DROP COLUMN channel;
			ALTER TABLE mfa_requests DROP COLUMN supported_channels;`,
		Down: `ALTER TABLE mfa_requests ADD COLUMN username VARCHAR;
			ALTER TABLE mfa_requests ADD COLUMN channel JSONB DEFAULT '{}':::JSONB;
			ALTER TABLE mfa_requests ADD COLUMN supported_channels JSONB DEFAULT '{}':::JSONB;`,
	},
	{
		Version: 127,
		Table:   "users",
		Desc:    "drop require_mfa column from users table",
		Up:      `ALTER TABLE users DROP COLUMN require_mfa;`,
		Down:    `ALTER TABLE users ADD COLUMN require_mfa BOOL;`,
	},
	{
		Version: 128,
		Table:   "authns_social",
		Desc:    "backfill oidc_issuer_url column in authns_social",
		Up: `UPDATE authns_social
                          SET oidc_issuer_url='https://accounts.google.com'
                          WHERE type=1
			  AND deleted='0001-01-01 00:00:00':::TIMESTAMP;
			  UPDATE authns_social
			  SET oidc_issuer_url='https://www.facebook.com'
			  WHERE type=2
                          AND deleted='0001-01-01 00:00:00':::TIMESTAMP;
			  UPDATE authns_social
                          SET oidc_issuer_url='https://www.linkedin.com'
                          WHERE type=3
                          AND deleted='0001-01-01 00:00:00':::TIMESTAMP;`,
		Down: `/* noop */`,
	},
	{
		Version: 129,
		Table:   "oidc_login_sessions",
		Desc:    "backfill oidc_issuer_url column in oidc_login_sessions",
		Up: `UPDATE oidc_login_sessions
			  SET oidc_issuer_url='https://accounts.google.com'
                          WHERE social_provider=1
			  AND deleted='0001-01-01 00:00:00':::TIMESTAMP;
			  UPDATE oidc_login_sessions
			  SET oidc_issuer_url='https://www.facebook.com'
			  WHERE social_provider=2
                          AND deleted='0001-01-01 00:00:00':::TIMESTAMP;
			  UPDATE oidc_login_sessions
			  SET oidc_issuer_url='https://www.linkedin.com'
                          WHERE social_provider=3
                          AND deleted='0001-01-01 00:00:00':::TIMESTAMP;`,
		Down: `/* noop */`,
	},
	{
		Version: 130,
		Table:   "purposes",
		Desc:    "add purposes table",
		Up: `CREATE TABLE purposes (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			PRIMARY KEY (deleted, id),
			UNIQUE(deleted, name));`,
		Down: `DROP TABLE purposes;`,
	},
	{
		Version: 131,
		Table:   "access_policies",
		Desc:    "add more columns to access_policies to support new policy types",
		Up: `ALTER TABLE access_policies ADD COLUMN description VARCHAR NOT NULL DEFAULT '';
			ALTER TABLE access_policies ADD COLUMN policy_type INT8 NOT NULL DEFAULT 101;
			ALTER TABLE access_policies ADD COLUMN tag_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN role_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN purpose_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN organization_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN attribute_names VARCHAR[];
			ALTER TABLE access_policies ADD COLUMN greater_than BOOL NOT NULL DEFAULT false;
			ALTER TABLE access_policies ADD COLUMN age INT NOT NULL DEFAULT 0;
			ALTER TABLE access_policies ADD COLUMN component_policy_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN intersect_policies BOOL NOT NULL DEFAULT false;`,
		Down: `ALTER TABLE access_policies DROP COLUMN description;
			ALTER TABLE access_policies DROP COLUMN policy_type;
			ALTER TABLE access_policies DROP COLUMN tag_ids;
			ALTER TABLE access_policies DROP COLUMN role_ids;
			ALTER TABLE access_policies DROP COLUMN purpose_ids;
			ALTER TABLE access_policies DROP COLUMN organization_ids;
			ALTER TABLE access_policies DROP COLUMN attribute_names;
			ALTER TABLE access_policies DROP COLUMN greater_than;
			ALTER TABLE access_policies DROP COLUMN age;
			ALTER TABLE access_policies DROP COLUMN component_policy_ids;
			ALTER TABLE access_policies DROP COLUMN intersect_policies;`,
	},
	{
		Version: 132,
		Table:   "access_policies",
		Desc:    "update access_policies policy_type and remove defaults",
		Up: `ALTER TABLE access_policies ALTER COLUMN name DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN description DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN policy_type DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN greater_than DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN age DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN intersect_policies DROP DEFAULT;`,
		Down: `ALTER TABLE access_policies ALTER COLUMN name SET DEFAULT '';
			ALTER TABLE access_policies ALTER COLUMN description SET DEFAULT '';
			ALTER TABLE access_policies ALTER COLUMN policy_type SET DEFAULT 101;
			ALTER TABLE access_policies ALTER COLUMN greater_than SET DEFAULT false;
			ALTER TABLE access_policies ALTER COLUMN age SET DEFAULT 0;
			ALTER TABLE access_policies ALTER COLUMN intersect_policies SET DEFAULT false;`,
	},
	{
		Version: 133,
		Table:   "idp_sync_runs",
		Desc:    "use this table to track app import etc",
		Up: `ALTER TABLE idp_sync_runs
			ADD COLUMN type VARCHAR NOT NULL DEFAULT '',
			ADD COLUMN warning_records INTEGER NOT NULL DEFAULT 0;`,
		Down: `ALTER TABLE idp_sync_runs DROP COLUMN "type",
			DROP COLUMN warning_records;`,
	},
	{
		Version: 134,
		Table:   "idp_sync_records",
		Desc:    "use this table to track app import etc",
		Up: `ALTER TABLE idp_sync_records
			ADD COLUMN object_id VARCHAR NOT NULL DEFAULT '',
			ADD COLUMN warning VARCHAR NOT NULL DEFAULT '';`,
		Down: `ALTER TABLE idp_sync_records DROP COLUMN object_id,
			DROP COLUMN warning;`,
	},
	{
		Version: 135,
		Table:   "idp_sync_runs",
		Desc:    "up to now, everything was a user sync",
		Up:      `UPDATE idp_sync_runs SET type='user';`,
		Down:    `/* no-op */`,
	},
	{
		// technically this change should be run after dual read/write code but for this I'm not going to worry about it
		Version: 136,
		Table:   "idp_sync_records",
		Desc:    "backfill more generically-named column",
		Up:      `UPDATE idp_sync_records SET object_id=user_id WHERE id=id;`,
		Down:    `/* no-op */`,
	},
	{
		Version: 137,
		Table:   "access_policies",
		Desc:    "update access_policies defaults so columns can be removed",
		Up: `ALTER TABLE access_policies ALTER COLUMN greater_than SET DEFAULT false;
			ALTER TABLE access_policies ALTER COLUMN age SET DEFAULT 0;
			ALTER TABLE access_policies ALTER COLUMN intersect_policies SET DEFAULT false;`,
		Down: `ALTER TABLE access_policies ALTER COLUMN greater_than DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN age DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN intersect_policies DROP DEFAULT;`,
	},
	{
		Version: 138,
		Table:   "access_policies",
		Desc:    "remove type-specific columns from access_policies",
		Up: `ALTER TABLE access_policies DROP COLUMN role_ids;
			 ALTER TABLE access_policies DROP COLUMN purpose_ids;
			 ALTER TABLE access_policies DROP COLUMN organization_ids;
			 ALTER TABLE access_policies DROP COLUMN attribute_names;
			 ALTER TABLE access_policies DROP COLUMN greater_than;
			 ALTER TABLE access_policies DROP COLUMN age;
			ALTER TABLE access_policies DROP COLUMN intersect_policies;`,
		Down: `ALTER TABLE access_policies ADD COLUMN role_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN purpose_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN organization_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN attribute_names VARCHAR[];
			ALTER TABLE access_policies ADD COLUMN greater_than BOOL NOT NULL DEFAULT false;
			ALTER TABLE access_policies ADD COLUMN age INT NOT NULL DEFAULT 0;
			ALTER TABLE access_policies ADD COLUMN intersect_policies BOOL NOT NULL DEFAULT false;`,
	},
	{
		Version: 139,
		Table:   "transformers",
		Desc:    "Create transformers table",
		Up: `CREATE TABLE transformers (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			input_type INT NOT NULL,
			output_type INT NOT NULL,
			transform_type INT NOT NULL,
			function string NOT NULL,
			parameters string NOT NULL,
			tag_ids UUID[],
			PRIMARY KEY (deleted, id),
			UNIQUE(deleted, name),
			UNIQUE(deleted, input_type, output_type, transform_type, function, parameters)
		);`,
		Down: `DROP TABLE transformers;`,
	},
	{
		Version: 140,
		Table:   "generation_policies",
		Desc:    "Delete generation_policies table",
		Up:      `DROP TABLE generation_policies;`,
		Down: `CREATE TABLE generation_policies (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			function VARCHAR NOT NULL,
			parameters VARCHAR NULL,
			name VARCHAR NOT NULL DEFAULT '':::STRING,
			CONSTRAINT "primary" PRIMARY KEY (id ASC, deleted ASC),
			UNIQUE INDEX generation_policies_function_parameters_deleted (function ASC, parameters ASC, deleted ASC),
			UNIQUE INDEX generation_policies_name (name ASC, deleted ASC),
			FAMILY "primary" (id, created, updated, deleted, function, parameters, name)
		);`,
	},
	{
		Version: 141,
		Table:   "token_records",
		Desc:    "Rename generation_policy_id column to transformer_id",
		Up:      `ALTER TABLE token_records RENAME COLUMN generation_policy_id TO transformer_id;`,
		Down:    `ALTER TABLE token_records RENAME COLUMN transformer_id TO generation_policy_id;`,
	},
	{
		Version: 142,
		Table:   "accessors",
		Desc:    "Add transformer_ids, token_access_policy_id columns to accessors table, remove transformation_policy_id column",
		Up: `ALTER TABLE accessors ADD COLUMN transformer_ids UUID[] NOT NULL DEFAULT '{}';
			ALTER TABLE accessors ADD COLUMN token_access_policy_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE accessors ALTER COLUMN transformation_policy_id SET DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down: `ALTER TABLE accessors DROP COLUMN transformer_ids;
			ALTER TABLE accessors ALTER COLUMN transformation_policy_id DROP DEFAULT;
			ALTER TABLE accessors DROP COLUMN token_access_policy_id;`,
	},
	{
		Version: 143,
		Table:   "accessors",
		Desc:    "Drop transformation_policy_id column and column_names column from accessors table; remove all existing accessors",
		Up: `ALTER TABLE accessors DROP COLUMN transformation_policy_id;
			ALTER TABLE accessors DROP COLUMN column_names;
			DELETE FROM accessors;`,
		Down: `ALTER TABLE accessors ADD COLUMN transformation_policy_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE accessors ADD COLUMN column_names VARCHAR[] NOT NULL DEFAULT '{}';`,
	},
	{
		Version: 144,
		Table:   "mutators",
		Desc:    "Drop column_names column from mutators table; remove all existing mutators",
		Up: `ALTER TABLE mutators DROP COLUMN column_names;
			ALTER TABLE mutators ADD COLUMN validator_ids UUID[] NOT NULL DEFAULT '{}';
			ALTER TABLE mutators ALTER COLUMN validation_policy_id SET DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			DELETE FROM mutators;`,
		Down: `ALTER TABLE mutators ADD COLUMN column_names VARCHAR[] NOT NULL DEFAULT '{}';
			ALTER TABLE mutators DROP COLUMN validator_ids;
			ALTER TABLE mutators ALTER COLUMN validation_policy_id DROP DEFAULT;`,
	},
	{
		Version: 145,
		Table:   "mutators",
		Desc:    "Drop validation_policy_id column from mutators table",
		Up:      `ALTER TABLE mutators DROP COLUMN validation_policy_id;`,
		Down:    `ALTER TABLE mutators ADD COLUMN validation_policy_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
	},
	{
		Version: 146,
		Table:   "transformers",
		Desc:    "Add output_type default to transformers table so it can be deleted",
		Up: `ALTER TABLE transformers ALTER COLUMN output_type SET DEFAULT 0;
			DELETE FROM transformers;`,
		Down: `ALTER TABLE transformers ALTER COLUMN output_type DROP DEFAULT;`,
	},
	{
		Version: 147,
		Table:   "transformers",
		Desc:    "Drop output_type column from transformers table",
		Up: `ALTER TABLE transformers DROP COLUMN output_type;
			CREATE UNIQUE INDEX transformers_deleted_input_type_transform_type_function_parameters_key on transformers (deleted ASC, function ASC, input_type ASC,
			parameters ASC, transform_type ASC)`,
		Down: `ALTER TABLE transformers ADD COLUMN output_type INT NOT NULL DEFAULT 0;
			DROP INDEX transformers_deleted_input_type_transform_type_function_parameters_key CASCADE;
			CREATE UNIQUE INDEX transformers_deleted_input_type_output_type_transform_type_function_parameters_key on transformers (deleted ASC, function ASC, input_type ASC,
				output_type ASC, parameters ASC, transform_type ASC);
		`,
	},
	{
		Version: 148,
		Table:   "accessors",
		Desc:    "Add purpose_ids column to accessors table",
		Up:      `ALTER TABLE accessors ADD COLUMN purpose_ids UUID[] NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE accessors DROP COLUMN purpose_ids;`,
	},
	{
		Version: 149,
		Table:   "access_policies",
		Desc:    "Update ID for Allow All access policy",
		Up:      `UPDATE access_policies SET id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';`,
		Down:    `UPDATE access_policies SET id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE id = '3f380e42-0b21-4570-a312-91e1b80386fa';`,
	},
	{
		Version: 150,
		Table:   "transformers",
		Desc:    "Update ID for TransformerUUID",
		Up:      `UPDATE transformers SET id = 'e3743f5b-521e-4305-b232-ee82549e1477' WHERE id = 'f5bce640-f866-4464-af1a-9e7474c4a90c';`,
		Down:    `UPDATE transformers SET id = 'f5bce640-f866-4464-af1a-9e7474c4a90c' WHERE id = 'e3743f5b-521e-4305-b232-ee82549e1477';`,
	},
	{
		Version: 151,
		Table:   "accessors",
		Desc:    "Update ID for Allow All access policy",
		Up: `UPDATE accessors SET access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';
			UPDATE accessors SET token_access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE token_access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';`,
		Down: `UPDATE accessors SET access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa';
			UPDATE accessors SET token_access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE token_access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa';`,
	},
	{
		Version: 152,
		Table:   "mutators",
		Desc:    "Update ID for Allow All access policy",
		Up:      `UPDATE mutators SET access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';`,
		Down:    `UPDATE mutators SET access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa';`,
	},
	{
		Version: 153,
		Table:   "token_records",
		Desc:    "Update ID for Allow All access policy and TransformerUUID",
		Up: `UPDATE token_records SET access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';
			UPDATE token_records SET transformer_id = 'e3743f5b-521e-4305-b232-ee82549e1477' WHERE transformer_id = 'f5bce640-f866-4464-af1a-9e7474c4a90c';`,
		Down: `UPDATE token_records SET access_policy_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE access_policy_id = '3f380e42-0b21-4570-a312-91e1b80386fa';
			UPDATE token_records SET transformer_id = 'f5bce640-f866-4464-af1a-9e7474c4a90c' WHERE transformer_id = 'e3743f5b-521e-4305-b232-ee82549e1477';`,
	},
	{
		Version: 154,
		Table:   "objects",
		Desc:    "Update ID for Allow All access policy and TransformerUUID",
		Up: `UPDATE objects SET id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';
			UPDATE objects SET id = 'e3743f5b-521e-4305-b232-ee82549e1477' WHERE id = 'f5bce640-f866-4464-af1a-9e7474c4a90c';`,
		Down: `UPDATE objects SET id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE id = '3f380e42-0b21-4570-a312-91e1b80386fa';
			UPDATE objects SET id = 'f5bce640-f866-4464-af1a-9e7474c4a90c' WHERE id = 'e3743f5b-521e-4305-b232-ee82549e1477';`,
	},
	{
		Version: 155,
		Table:   "edges",
		Desc:    "Update ID for Allow All access policy and TransformerUUID",
		Up: `Update edges SET target_object_id = '3f380e42-0b21-4570-a312-91e1b80386fa' WHERE target_object_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe';
			Update edges SET target_object_id = 'e3743f5b-521e-4305-b232-ee82549e1477' WHERE target_object_id = 'f5bce640-f866-4464-af1a-9e7474c4a90c';`,
		Down: `Update edges SET target_object_id = '1bf2b775-e521-41d3-8b7e-78e89427e6fe' WHERE target_object_id = '3f380e42-0b21-4570-a312-91e1b80386fa';
			Update edges SET target_object_id = 'f5bce640-f866-4464-af1a-9e7474c4a90c' WHERE target_object_id = 'e3743f5b-521e-4305-b232-ee82549e1477';`,
	},
	{
		Version: 156,
		Table:   "oidc_login_sessions",
		Desc:    "Clear mfa_channel_states because of format change",
		Up:      `UPDATE oidc_login_sessions SET mfa_channel_states = '{}'::JSONB WHERE deleted = '0001-01-01 00:00:00':::TIMESTAMP;`,
		Down:    `/* noop */`,
	},
	{
		Version: 157,
		Table:   "access_policy_templates",
		Desc:    "Creating access_policy_templates table",
		Up: `CREATE TABLE access_policy_templates (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			name VARCHAR NOT NULL DEFAULT '':::STRING,
			description VARCHAR NOT NULL DEFAULT '':::STRING,
			function VARCHAR NOT NULL,
			version INT NOT NULL,
			CONSTRAINT "primary" PRIMARY KEY (deleted ASC, id ASC, version ASC),
			UNIQUE INDEX access_policy_templates_deleted_function_version (deleted ASC, function ASC, version ASC),
			UNIQUE INDEX access_policy_templates_deleted_name_version (deleted ASC, name ASC, version ASC)
		);`,
		Down: `DROP TABLE access_policy_templates;`,
	},
	{
		Version: 158,
		Table:   "access_policies",
		Desc:    "Updating access_policies table",
		Up: `ALTER TABLE access_policies ALTER COLUMN function SET DEFAULT '';
			ALTER TABLE access_policies ALTER COLUMN parameters SET DEFAULT '';`,
		Down: `ALTER TABLE access_policies ALTER COLUMN function DROP DEFAULT;
			ALTER TABLE access_policies ALTER COLUMN parameters DROP DEFAULT;`,
	},
	{
		Version: 159,
		Table:   "access_policies",
		Desc:    "Updating access_policies table",
		Up: `ALTER TABLE access_policies ADD COLUMN component_ids UUID[];
			ALTER TABLE access_policies ADD COLUMN component_parameters VARCHAR[];
			ALTER TABLE access_policies ADD COLUMN component_types INT4[];
			DROP INDEX access_policies_function_parameters_version_deleted CASCADE;`,
		Down: `ALTER TABLE access_policies DROP COLUMN component_ids;
			ALTER TABLE access_policies DROP COLUMN component_parameters;
			ALTER TABLE access_policies DROP COLUMN component_types;
			CREATE UNIQUE INDEX access_policies_function_parameters_version_deleted on access_policies (deleted ASC, function ASC, parameters ASC, version ASC);`,
	},
	{
		Version: 160,
		Table:   "token_records",
		Desc:    "Updating token_records table to include userstore references",
		Up: `ALTER TABLE token_records ADD COLUMN user_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE token_records ADD COLUMN column_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down: `ALTER TABLE token_records DROP COLUMN user_id;
			ALTER TABLE token_records DROP COLUMN column_id;`,
	},
	{
		Version: 161,
		Table:   "access_policies",
		Desc:    "Updating access_policies policy_type ",
		Up:      `UPDATE access_policies SET policy_type = 1 WHERE policy_type = 101;`,
		Down:    `UPDATE access_policies SET policy_type = 101 WHERE policy_type = 1 AND false;`,
	},
	{
		Version: 162,
		Table:   "access_policies",
		Desc:    "Updating access_policies names",
		Up:      `UPDATE access_policies SET name = replace(name, ' ', '') WHERE name LIKE '% %' AND deleted = '0001-01-01 00:00:00';`,
		Down:    `/* noop */`,
	},
	{
		Version: 163,
		Table:   "column_value_retention_durations",
		Desc:    "Creating column_value_retention_durations table",
		Up: `CREATE TABLE column_value_retention_durations (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			_version INT NOT NULL DEFAULT 0,
			column_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
			purpose_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
			duration_type INT NOT NULL,
			duration_unit INT NOT NULL,
			duration INT NOT NULL,
			CONSTRAINT "primary" PRIMARY KEY (deleted ASC, id ASC),
			UNIQUE INDEX column_value_retention_durations_deleted_unique (deleted ASC, duration_type ASC, purpose_id ASC, column_id ASC)
		);`,
		Down: `DROP TABLE column_value_retention_durations;`,
	},
	{
		Version: 164,
		Table:   "plex_config",
		Desc:    "Moving companyconfig.tenants_plex to tenantdb.plex_config",
		Up: `CREATE TABLE plex_config (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			plex_config JSONB NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			_version INT8 NOT NULL DEFAULT 0:::INT8,
			CONSTRAINT "primary" PRIMARY KEY (deleted ASC, id ASC)
		);`,
		Down: `DROP TABLE plex_config;`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 165,
		Table:   "delegation_invites",
		Desc:    "fix delegation_invites PK columns",
		Up: `ALTER TABLE delegation_invites DROP CONSTRAINT "primary";
			ALTER TABLE delegation_invites ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE delegation_invites DROP CONSTRAINT "primary";
			ALTER TABLE delegation_invites ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 166,
		Table:   "delegation_states",
		Desc:    "fix delegation_states PK columns",
		Up: `ALTER TABLE delegation_states DROP CONSTRAINT "primary";
			ALTER TABLE delegation_states ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE delegation_states DROP CONSTRAINT "primary";
			ALTER TABLE delegation_states ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 167,
		Table:   "mfa_states",
		Desc:    "fix mfa_states PK columns",
		Up: `ALTER TABLE mfa_states DROP CONSTRAINT "primary";
			ALTER TABLE mfa_states ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE mfa_states DROP CONSTRAINT "primary";
			ALTER TABLE mfa_states ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 168,
		Table:   "oidc_login_sessions",
		Desc:    "fix oidc_login_sessions PK columns",
		Up: `ALTER TABLE oidc_login_sessions DROP CONSTRAINT "primary";
			ALTER TABLE oidc_login_sessions ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE oidc_login_sessions DROP CONSTRAINT "primary";
			ALTER TABLE oidc_login_sessions ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 169,
		Table:   "otp_states",
		Desc:    "fix otp_states PK columns",
		Up: `ALTER TABLE otp_states DROP CONSTRAINT "primary";
			ALTER TABLE otp_states ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE otp_states DROP CONSTRAINT "primary";
			ALTER TABLE otp_states ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 170,
		Table:   "pkce_states",
		Desc:    "fix pkce_states PK columns",
		Up: `ALTER TABLE pkce_states DROP CONSTRAINT "primary";
			ALTER TABLE pkce_states ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE pkce_states DROP CONSTRAINT "primary";
			ALTER TABLE pkce_states ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Fix the primary key to order by deleted first since our filters are most commonly deleted='0001-01-01 00:00:00'::TIMESTAMP
		Version: 171,
		Table:   "plex_tokens",
		Desc:    "fix plex_tokens PK columns",
		Up: `ALTER TABLE plex_tokens DROP CONSTRAINT "primary";
			ALTER TABLE plex_tokens ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE plex_tokens DROP CONSTRAINT "primary";
			ALTER TABLE plex_tokens ADD CONSTRAINT "primary" PRIMARY KEY (id);`,
	},
	{
		// Update index for id in columns table to be unique
		Version: 172,
		Table:   "columns",
		Desc:    "update index type for id column to be 2 (unique)",
		Up:      `UPDATE columns SET index_type = 2 WHERE id = 'b1d12a3e-dbf7-4405-b5fc-7e3919d8e089';`,
		Down:    `UPDATE columns SET index_type = 0 WHERE id = 'b1d12a3e-dbf7-4405-b5fc-7e3919d8e089';`,
	},
	{
		Version: 173,
		Table:   "user_column_pre_delete_values",
		Desc:    "Creating user_column_pre_delete_values table",
		Up: `CREATE TABLE user_column_pre_delete_values (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			_version INT NOT NULL,
			column_id UUID NOT NULL,
			user_id UUID NOT NULL,
			varchar_value VARCHAR,
			varchar_unique_value VARCHAR,
			boolean_value BOOLEAN,
			int_value INT,
			int_unique_value INT,
			timestamp_value TIMESTAMP,
			uuid_value UUID,
			uuid_unique_value UUID,
			jsonb_value JSONB,
			ordering INT NOT NULL,
			consented_purpose_ids UUID[] NOT NULL,
			retention_timeouts STRING[] NOT NULL,
			CONSTRAINT "user_column_pre_delete_values_pk" PRIMARY KEY (id),
			UNIQUE (column_id, varchar_unique_value),
			UNIQUE (column_id, int_unique_value),
			UNIQUE (column_id, uuid_unique_value),
			INDEX user_column_pre_delete_values_column_user_varchar (column_id ASC, user_id ASC, varchar_value ASC),
			INDEX user_column_pre_delete_values_column_user_unique_varchar (column_id ASC, user_id ASC, varchar_unique_value ASC),
			INDEX user_column_pre_delete_values_column_user_boolean (column_id ASC, user_id ASC, boolean_value ASC),
			INDEX user_column_pre_delete_values_column_user_int (column_id ASC, user_id ASC, int_value ASC),
			INDEX user_column_pre_delete_values_column_user_unique_int (column_id ASC, user_id ASC, int_unique_value ASC),
			INDEX user_column_pre_delete_values_column_user_timestamp (column_id ASC, user_id ASC, timestamp_value ASC),
			INDEX user_column_pre_delete_values_column_user_uuid (column_id ASC, user_id ASC, uuid_value ASC),
			INDEX user_column_pre_delete_values_column_user_unique_uuid (column_id ASC, user_id ASC, uuid_unique_value ASC),
			INVERTED INDEX user_column_pre_delete_values_column_user_jsonb (column_id ASC, user_id ASC, jsonb_value)
		);`,
		Down: `DROP TABLE user_column_pre_delete_values;`,
	},
	{
		Version: 174,
		Table:   "user_column_post_delete_values",
		Desc:    "Creating user_column_post_delete_values table",
		Up: `CREATE TABLE user_column_post_delete_values (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			_version INT NOT NULL,
			column_id UUID NOT NULL,
			user_id UUID NOT NULL,
			varchar_value VARCHAR,
			boolean_value BOOLEAN,
			int_value INT,
			timestamp_value TIMESTAMP,
			uuid_value UUID,
			jsonb_value JSONB,
			ordering INT NOT NULL,
			consented_purpose_ids UUID[] NOT NULL,
			retention_timeouts STRING[] NOT NULL,
			CONSTRAINT "user_column_post_delete_values_pk" PRIMARY KEY (id),
			INDEX user_column_post_delete_values_varchar (column_id ASC, user_id ASC, varchar_value ASC),
			INDEX user_column_post_delete_values_boolean (column_id ASC, user_id ASC, boolean_value ASC),
			INDEX user_column_post_delete_values_int (column_id ASC, user_id ASC, int_value ASC),
			INDEX user_column_post_delete_values_timestamp (column_id ASC, user_id ASC, timestamp_value ASC),
			INDEX user_column_post_delete_values_uuid (column_id ASC, user_id ASC, uuid_value ASC),
			INVERTED INDEX user_column_post_delete_values_jsonb (column_id ASC, user_id ASC, jsonb_value)
		);`,
		Down: `DROP TABLE user_column_post_delete_values;`,
	},
	{
		Version: 175,
		Table:   "users",
		Desc:    "Ensure all users have a valid version",
		Up:      `UPDATE users SET _version = 1 WHERE deleted = '0001-01-01 00:00:00' and _version = 0;`,
		Down:    `/* noop */`,
	},
	{
		Version: 176,
		Table:   "transformers",
		Desc:    "add is_system to transformers",
		Up:      `ALTER TABLE transformers ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE transformers DROP COLUMN is_system;`,
	},
	{
		Version: 177,
		Table:   "access_policies",
		Desc:    "add is_system to access_policies",
		Up:      `ALTER TABLE access_policies ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE access_policies DROP COLUMN is_system;`,
	},
	{
		Version: 178,
		Table:   "access_policy_templates",
		Desc:    "add is_system to access_policy_templates",
		Up:      `ALTER TABLE access_policy_templates ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE access_policy_templates DROP COLUMN is_system;`,
	},
	{
		Version: 179,
		Table:   "accessors",
		Desc:    "add is_system to accessors",
		Up:      `ALTER TABLE accessors ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE accessors DROP COLUMN is_system;`,
	},
	{
		Version: 180,
		Table:   "mutators",
		Desc:    "add is_system to mutators",
		Up:      `ALTER TABLE mutators ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE mutators DROP COLUMN is_system;`,
	},
	{
		Version: 181,
		Table:   "purposes",
		Desc:    "add is_system to purposes",
		Up:      `ALTER TABLE purposes ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE purposes DROP COLUMN is_system;`,
	},
	{
		Version: 182,
		Table:   "auditlog",
		Desc:    "change auditlog table column 'type' type from STRING to VARCHAR",
		Up:      `ALTER TABLE auditlog ALTER COLUMN type TYPE VARCHAR;`,
		Down:    `ALTER TABLE auditlog ALTER COLUMN type TYPE STRING;`,
	},
	{
		Version: 183,
		Table:   "auditlog",
		Desc:    "change auditlog table column 'actor_id' type from STRING to VARCHAR",
		Up:      `ALTER TABLE auditlog ALTER COLUMN actor_id TYPE VARCHAR;`,
		Down:    `ALTER TABLE auditlog ALTER COLUMN actor_id TYPE STRING;`,
	},
	{
		Version: 184,
		Table:   "delegation_invites",
		Desc:    "change delegation_invites table column 'client_id' type from STRING to VARCHAR",
		Up:      `ALTER TABLE delegation_invites ALTER COLUMN client_id TYPE VARCHAR;`,
		Down:    `ALTER TABLE delegation_invites ALTER COLUMN client_id TYPE STRING;`,
	},
	{
		Version: 185,
		Table:   "delegation_invites",
		Desc:    "change delegation_invites table column 'invited_to_account_id' type from STRING to VARCHAR",
		Up:      `ALTER TABLE delegation_invites ALTER COLUMN invited_to_account_id TYPE VARCHAR;`,
		Down:    `ALTER TABLE delegation_invites ALTER COLUMN invited_to_account_id TYPE STRING;`,
	},
	{
		Version: 186,
		Table:   "transformers",
		Desc:    "change transformers table column 'function' type from STRING to VARCHAR",
		Up:      `ALTER TABLE transformers ALTER COLUMN function TYPE VARCHAR;`,
		Down:    `ALTER TABLE transformers ALTER COLUMN function TYPE STRING;`,
	},
	{
		Version: 187,
		Table:   "transformers",
		Desc:    "change transformers table column 'parameters' type from STRING to VARCHAR",
		Up:      `ALTER TABLE transformers ALTER COLUMN parameters TYPE VARCHAR;`,
		Down:    `ALTER TABLE transformers ALTER COLUMN parameters TYPE STRING;`,
	},
	{
		Version: 188,
		Table:   "user_column_pre_delete_values", // fake table name to make tests happy.
		Desc:    "change settings https://github.com/cockroachdb/cockroach/issues/49329?version=v22.2",
		Up:      `SET enable_experimental_alter_column_type_general = true;`,
		Down:    `SET enable_experimental_alter_column_type_general = false;`,
	},
	{
		Version: 189,
		Table:   "user_column_pre_delete_values",
		Desc:    "change user_column_pre_delete_values table column 'retention_timeouts' type from STRING[] to VARCHAR[]",
		Up:      `ALTER TABLE user_column_pre_delete_values ALTER COLUMN retention_timeouts TYPE VARCHAR[];`,
		Down:    `ALTER TABLE user_column_pre_delete_values ALTER COLUMN retention_timeouts TYPE STRING[];`,
	},
	{
		Version: 190,
		Table:   "user_column_post_delete_values",
		Desc:    "change user_column_post_delete_values table column 'retention_timeouts' type from STRING[] to VARCHAR[]",
		Up:      `ALTER TABLE user_column_post_delete_values ALTER COLUMN retention_timeouts TYPE VARCHAR[];`,
		Down:    `ALTER TABLE user_column_post_delete_values ALTER COLUMN retention_timeouts TYPE STRING[];`,
	},
	{
		Version: 191,
		Table:   "accessors",
		Desc:    "Add data_life_cycle_state to accessors table",
		Up:      `ALTER TABLE accessors ADD COLUMN data_life_cycle_state INT NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE accessors DROP COLUMN data_life_cycle_state;`,
	},
	{
		Version: 192,
		Table:   "accessors",
		Desc:    "Set data_life_cycle_state to DataLifeCycleStatePreDelete for all accessors",
		Up:      `UPDATE accessors SET data_life_cycle_state = 1;`,
		Down:    `UPDATE accessors SET data_life_cycle_state = 0;`,
	},
	{
		Version: 193,
		Table:   "users",
		Desc:    "Remove index for external_alias",
		Up:      `DROP INDEX users_external_alias_deleted_key CASCADE;`,
		Down:    `ALTER TABLE users ADD CONSTRAINT users_external_alias_deleted_key UNIQUE (external_alias, deleted);`,
	},
	{
		Version: 194,
		Table:   "users",
		Desc:    "Drop external_alias column",
		Up:      `ALTER TABLE users DROP COLUMN external_alias;`,
		Down:    `ALTER TABLE users ADD COLUMN external_alias VARCHAR;`,
	},
	{
		Version: 195,
		Table:   "access_policies",
		Desc:    "fix access_policies PK name",
		Up: `ALTER TABLE access_policies DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE access_policies ADD CONSTRAINT IF NOT EXISTS "access_policies_pkey" PRIMARY KEY (id, deleted, version);`,
		Down: `ALTER TABLE access_policies DROP CONSTRAINT "access_policies_pkey";
			ALTER TABLE access_policies ADD CONSTRAINT "primary" PRIMARY KEY (deleted, id, version);`,
	},
	{
		Version: 196,
		Table:   "access_policies",
		Desc:    "fix access_policies PK",
		Up: `ALTER TABLE access_policies DROP CONSTRAINT "access_policies_pkey";
			ALTER TABLE access_policies ADD CONSTRAINT "access_policies_pkey" PRIMARY KEY (deleted, id, version);`,
		Down: `ALTER TABLE access_policies DROP CONSTRAINT "access_policies_pkey";
			ALTER TABLE access_policies ADD CONSTRAINT "access_policies_pkey" PRIMARY KEY (id, deleted, version);`,
	},
	{
		Version: 197,
		Table:   "objects",
		Desc:    "fix objects PK name",
		Up: `ALTER TABLE objects DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE objects ADD CONSTRAINT IF NOT EXISTS "objects_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE objects RENAME CONSTRAINT "objects_pkey" TO "primary";`,
	},
	{
		Version: 198,
		Table:   "objects",
		Desc:    "fix objects PK",
		Up: `ALTER TABLE objects DROP CONSTRAINT "objects_pkey";
			ALTER TABLE objects ADD CONSTRAINT "objects_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE objects DROP CONSTRAINT "objects_pkey";
			ALTER TABLE objects ADD CONSTRAINT "objects_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 199,
		Table:   "users",
		Desc:    "fix users PK name",
		Up: `ALTER TABLE users DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE users ADD CONSTRAINT IF NOT EXISTS "users_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE users RENAME CONSTRAINT "users_pkey" TO "primary";`,
	},
	{
		Version: 200,
		Table:   "users",
		Desc:    "fix users PK",
		Up: `ALTER TABLE users DROP CONSTRAINT "users_pkey";
			ALTER TABLE users ADD CONSTRAINT "users_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE users DROP CONSTRAINT "users_pkey";
			ALTER TABLE users ADD CONSTRAINT "users_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 201,
		Table:   "acme_certificates",
		Desc:    "fix acme_certificates PK",
		Up: `ALTER TABLE acme_certificates DROP CONSTRAINT "acme_certificates_pkey";
			ALTER TABLE acme_certificates ADD CONSTRAINT "acme_certificates_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE acme_certificates DROP CONSTRAINT "acme_certificates_pkey";
			ALTER TABLE acme_certificates ADD CONSTRAINT "acme_certificates_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 202,
		Table:   "acme_orders",
		Desc:    "fix acme_orders PK",
		Up: `ALTER TABLE acme_orders DROP CONSTRAINT "acme_orders_pkey";
			ALTER TABLE acme_orders ADD CONSTRAINT "acme_orders_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE acme_orders DROP CONSTRAINT "acme_orders_pkey";
			ALTER TABLE acme_orders ADD CONSTRAINT "acme_orders_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 203,
		Table:   "accessors",
		Desc:    "fix accessors PK name",
		Up: `ALTER TABLE accessors DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE accessors ADD CONSTRAINT IF NOT EXISTS "accessors_pkey" PRIMARY KEY (id, version, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 204,
		Table:   "accessors",
		Desc:    "fix accessors PK columns",
		Up: `ALTER TABLE accessors DROP CONSTRAINT "accessors_pkey";
			ALTER TABLE accessors ADD CONSTRAINT "accessors_pkey" PRIMARY KEY (deleted, id, version);`,
		Down: `ALTER TABLE accessors DROP CONSTRAINT "accessors_pkey";
			ALTER TABLE accessors ADD CONSTRAINT "accessors_pkey" PRIMARY KEY (id, version, deleted);`,
	},
	{
		Version: 205,
		Table:   "auditlog",
		Desc:    "fix auditlog PK name",
		Up: `ALTER TABLE auditlog DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE auditlog ADD CONSTRAINT IF NOT EXISTS "auditlog_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE auditlog RENAME CONSTRAINT "auditlog_pkey" TO "primary";`,
	},
	{
		Version: 206,
		Table:   "auditlog",
		Desc:    "fix auditlog PK",
		Up: `ALTER TABLE auditlog DROP CONSTRAINT "auditlog_pkey";
			ALTER TABLE auditlog ADD CONSTRAINT "auditlog_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE auditlog DROP CONSTRAINT "auditlog_pkey";
			ALTER TABLE auditlog ADD CONSTRAINT "auditlog_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 207,
		Table:   "authns_password",
		Desc:    "fix authns_password PK",
		Up: `ALTER TABLE authns_password DROP CONSTRAINT "primary";
			ALTER TABLE authns_password ADD CONSTRAINT "authns_password_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE authns_password DROP CONSTRAINT "authns_password_pkey";
			ALTER TABLE authns_password ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 208,
		Table:   "authns_social",
		Desc:    "fix authns_social PK",
		Up: `ALTER TABLE authns_social DROP CONSTRAINT "primary";
			ALTER TABLE authns_social ADD CONSTRAINT "authns_social_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE authns_social DROP CONSTRAINT "authns_social_pkey";
			ALTER TABLE authns_social ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 209,
		Table:   "columns",
		Desc:    "fix columns PK name",
		Up: `ALTER TABLE columns DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE columns ADD CONSTRAINT IF NOT EXISTS "columns_pkey" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 210,
		Table:   "columns",
		Desc:    "fix columns PK",
		Up: `ALTER TABLE columns DROP CONSTRAINT "columns_pkey";
			ALTER TABLE columns ADD CONSTRAINT "columns_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE columns DROP CONSTRAINT "columns_pkey";
			ALTER TABLE columns ADD CONSTRAINT "columns_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 211,
		Table:   "custom_pages",
		Desc:    "fix custom_pages PK name",
		Up: `ALTER TABLE custom_pages DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE custom_pages ADD CONSTRAINT IF NOT EXISTS "custom_pages_pkey" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 212,
		Table:   "custom_pages",
		Desc:    "fix custom_pages PK",
		Up: `ALTER TABLE custom_pages DROP CONSTRAINT "custom_pages_pkey";
			ALTER TABLE custom_pages ADD CONSTRAINT "custom_pages_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE custom_pages DROP CONSTRAINT "custom_pages_pkey";
			ALTER TABLE custom_pages ADD CONSTRAINT "custom_pages_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 213,
		Table:   "edge_types",
		Desc:    "fix edge_types PK",
		Up: `ALTER TABLE edge_types DROP CONSTRAINT "primary";
			ALTER TABLE edge_types ADD CONSTRAINT "edge_types_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE edge_types DROP CONSTRAINT "edge_types_pkey";
			ALTER TABLE edge_types ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 214,
		Table:   "idp_sync_records",
		Desc:    "fix idp_sync_records PK name",
		Up: `ALTER TABLE idp_sync_records DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE idp_sync_records ADD CONSTRAINT IF NOT EXISTS "idp_sync_records_pkey" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 215,
		Table:   "idp_sync_records",
		Desc:    "fix idp_sync_records PK",
		Up: `ALTER TABLE idp_sync_records DROP CONSTRAINT "idp_sync_records_pkey";
			ALTER TABLE idp_sync_records ADD CONSTRAINT "idp_sync_records_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE idp_sync_records DROP CONSTRAINT "idp_sync_records_pkey";
			ALTER TABLE idp_sync_records ADD CONSTRAINT "idp_sync_records_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 216,
		Table:   "idp_sync_runs",
		Desc:    "fix idp_sync_runs PK name",
		Up: `ALTER TABLE idp_sync_runs DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE idp_sync_runs ADD CONSTRAINT IF NOT EXISTS "idp_sync_runs_pkey" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 217,
		Table:   "idp_sync_runs",
		Desc:    "fix idp_sync_runs PK",
		Up: `ALTER TABLE idp_sync_runs DROP CONSTRAINT "idp_sync_runs_pkey";
			ALTER TABLE idp_sync_runs ADD CONSTRAINT "idp_sync_runs_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE idp_sync_runs DROP CONSTRAINT "idp_sync_runs_pkey";
			ALTER TABLE idp_sync_runs ADD CONSTRAINT "idp_sync_runs_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 218,
		Table:   "mfa_requests",
		Desc:    "fix mfa_requests PK",
		Up: `ALTER TABLE mfa_requests DROP CONSTRAINT "primary";
			ALTER TABLE mfa_requests ADD CONSTRAINT "mfa_requests_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE mfa_requests DROP CONSTRAINT "mfa_requests_pkey";
			ALTER TABLE mfa_requests ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 219,
		Table:   "mutators",
		Desc:    "fix mutators PK name",
		Up: `ALTER TABLE mutators DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE mutators ADD CONSTRAINT IF NOT EXISTS "mutators_pkey" PRIMARY KEY (id, version, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 220,
		Table:   "mutators",
		Desc:    "fix mutators PK",
		Up: `ALTER TABLE mutators DROP CONSTRAINT "mutators_pkey";
			ALTER TABLE mutators ADD CONSTRAINT "mutators_pkey" PRIMARY KEY (deleted, id, version);`,
		Down: `ALTER TABLE mutators DROP CONSTRAINT "mutators_pkey";
			ALTER TABLE mutators ADD CONSTRAINT "mutators_pkey" PRIMARY KEY (id, version, deleted);`,
	},
	{
		Version: 221,
		Table:   "object_types",
		Desc:    "fix object_types PK",
		Up: `ALTER TABLE object_types DROP CONSTRAINT "primary";
			ALTER TABLE object_types ADD CONSTRAINT "object_types_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE object_types DROP CONSTRAINT "object_types_pkey";
			ALTER TABLE object_types ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 222,
		Table:   "token_records",
		Desc:    "fix token_records PK name",
		Up: `ALTER TABLE token_records DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE token_records ADD CONSTRAINT IF NOT EXISTS "token_records_pkey" PRIMARY KEY (token, deleted);`,
		Down: `ALTER TABLE token_records DROP CONSTRAINT "token_records_pkey";
			ALTER TABLE token_records ADD CONSTRAINT "primary" PRIMARY KEY (token, deleted);`,
	},
	{
		Version: 223,
		Table:   "token_records",
		Desc:    "fix token_records PK",
		Up: `ALTER TABLE token_records DROP CONSTRAINT "token_records_pkey";
			ALTER TABLE token_records ADD CONSTRAINT "token_records_pkey" PRIMARY KEY (deleted, token);`,
		Down: `ALTER TABLE token_records DROP CONSTRAINT "token_records_pkey";
			ALTER TABLE token_records ADD CONSTRAINT "token_records_pkey" PRIMARY KEY (token, deleted);`,
	},
	{
		Version: 224,
		Table:   "user_mfa_configuration",
		Desc:    "fix user_mfa_configuration PK name",
		Up: `ALTER TABLE user_mfa_configuration DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE user_mfa_configuration ADD CONSTRAINT IF NOT EXISTS "user_mfa_configuration_pkey" PRIMARY KEY (id, deleted);`,
		Down: `/* no op */`,
	},
	{
		Version: 225,
		Table:   "user_mfa_configuration",
		Desc:    "fix user_mfa_configuration PK",
		Up: `ALTER TABLE user_mfa_configuration DROP CONSTRAINT "user_mfa_configuration_pkey";
			ALTER TABLE user_mfa_configuration ADD CONSTRAINT "user_mfa_configuration_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE user_mfa_configuration DROP CONSTRAINT "user_mfa_configuration_pkey";
			ALTER TABLE user_mfa_configuration ADD CONSTRAINT "user_mfa_configuration_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 226,
		Table:   "access_policy_templates",
		Desc:    "fix access_policy_templates PK name",
		Up:      `ALTER TABLE access_policy_templates RENAME CONSTRAINT "primary" TO "access_policy_templates_pkey";`,
		Down:    `ALTER TABLE access_policy_templates RENAME CONSTRAINT "access_policy_templates_pkey" TO "primary";`,
	},
	{
		Version: 227,
		Table:   "column_value_retention_durations",
		Desc:    "fix column_value_retention_durations PK name",
		Up:      `ALTER TABLE column_value_retention_durations RENAME CONSTRAINT "primary" TO "column_value_retention_durations_pkey";`,
		Down:    `ALTER TABLE column_value_retention_durations RENAME CONSTRAINT "column_value_retention_durations_pkey" TO "primary";`,
	},
	{
		Version: 228,
		Table:   "delegation_invites",
		Desc:    "fix delegation_invites PK name",
		Up:      `ALTER TABLE delegation_invites RENAME CONSTRAINT "primary" TO "delegation_invites_pkey";`,
		Down:    `ALTER TABLE delegation_invites RENAME CONSTRAINT "delegation_invites_pkey" TO "primary";`,
	},
	{
		Version: 229,
		Table:   "delegation_states",
		Desc:    "fix delegation_states PK name",
		Up:      `ALTER TABLE delegation_states RENAME CONSTRAINT "primary" TO "delegation_states_pkey";`,
		Down:    `ALTER TABLE delegation_states RENAME CONSTRAINT "delegation_states_pkey" TO "primary";`,
	},
	{
		Version: 230,
		Table:   "edges",
		Desc:    "fix edges PK name",
		Up:      `ALTER TABLE edges RENAME CONSTRAINT "primary" TO "edges_pkey";`,
		Down:    `ALTER TABLE edges RENAME CONSTRAINT "edges_pkey" TO "primary";`,
	},
	{
		Version: 231,
		Table:   "mfa_states",
		Desc:    "fix mfa_states PK name",
		Up:      `ALTER TABLE mfa_states RENAME CONSTRAINT "primary" TO "mfa_states_pkey";`,
		Down:    `ALTER TABLE mfa_states RENAME CONSTRAINT "mfa_states_pkey" TO "primary";`,
	},
	{
		Version: 232,
		Table:   "oidc_login_sessions",
		Desc:    "fix oidc_login_sessions PK name",
		Up:      `ALTER TABLE oidc_login_sessions RENAME CONSTRAINT "primary" TO "oidc_login_sessions_pkey";`,
		Down:    `ALTER TABLE oidc_login_sessions RENAME CONSTRAINT "oidc_login_sessions_pkey" TO "primary";`,
	},
	{
		Version: 233,
		Table:   "organizations",
		Desc:    "fix organizations PK name",
		Up: `ALTER TABLE organizations DROP CONSTRAINT IF EXISTS "primary";
			ALTER TABLE organizations ADD CONSTRAINT IF NOT EXISTS "organizations_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE organizations DROP CONSTRAINT "organizations_pkey";
			ALTER TABLE organizations ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 234,
		Table:   "organizations",
		Desc:    "fix organizations PK",
		Up: `ALTER TABLE organizations DROP CONSTRAINT "organizations_pkey";
			ALTER TABLE organizations ADD CONSTRAINT "organizations_pkey" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE organizations DROP CONSTRAINT "organizations_pkey";
			ALTER TABLE organizations ADD CONSTRAINT "organizations_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 235,
		Table:   "otp_states",
		Desc:    "fix otp_states PK name",
		Up:      `ALTER TABLE otp_states RENAME CONSTRAINT "primary" TO "otp_states_pkey";`,
		Down:    `ALTER TABLE otp_states RENAME CONSTRAINT "otp_states_pkey" TO "primary";`,
	},
	{
		Version: 236,
		Table:   "pkce_states",
		Desc:    "fix pkce_states PK name",
		Up:      `ALTER TABLE pkce_states RENAME CONSTRAINT "primary" TO "pkce_states_pkey";`,
		Down:    `ALTER TABLE pkce_states RENAME CONSTRAINT "pkce_states_pkey" TO "primary";`,
	},
	{
		Version: 237,
		Table:   "plex_config",
		Desc:    "fix plex_config PK name",
		Up:      `ALTER TABLE plex_config RENAME CONSTRAINT "primary" TO "plex_config_pkey";`,
		Down:    `ALTER TABLE plex_config RENAME CONSTRAINT "plex_config_pkey" TO "primary";`,
	},
	{
		Version: 238,
		Table:   "plex_tokens",
		Desc:    "fix plex_tokens PK name",
		Up:      `ALTER TABLE plex_tokens RENAME CONSTRAINT "primary" TO "plex_tokens_pkey";`,
		Down:    `ALTER TABLE plex_tokens RENAME CONSTRAINT "plex_tokens_pkey" TO "primary";`,
	},
	{
		Version: 239,
		Table:   "plex_tokens",
		Desc:    "fake migration",
		Up:      `/* no op */`,
		Down:    `/* no op */`,
	},
	{
		Version: 240,
		Table:   "transformers",
		Desc:    "add output_type column to transformers table",
		Up:      `ALTER TABLE transformers ADD COLUMN output_type INT;`,
		Down:    `ALTER TABLE transformers DROP COLUMN output_type;`,
	},
	{
		Version: 241,
		Table:   "transformers",
		Desc:    "backfil output_type to DataTypeString in transformers table",
		Up:      `UPDATE transformers SET output_type=100;`,
		Down:    `/* no op */`,
	},
	{
		Version: 242,
		Table:   "transformers",
		Desc:    "make output_type non null in transformers table",
		Up:      `ALTER TABLE transformers ALTER COLUMN output_type SET NOT NULL;`,
		Down:    `ALTER TABLE transformers ALTER COLUMN output_type DROP NOT NULL;`,
	},
	{
		Version: 243,
		Table:   "transformers",
		Desc:    "add reuse_existing_token column to transformers table",
		Up:      `ALTER TABLE transformers ADD COLUMN reuse_existing_token BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE transformers DROP COLUMN reuse_existing_token;`,
	},
	{
		Version: 244,
		Table:   "transformers",
		Desc:    "backfil reuse_existing_token to DataTypeString in transformers table",
		Up:      `UPDATE transformers SET reuse_existing_token=false;`,
		Down:    `/* no op */`,
	},
	{
		Version: 245,
		Table:   "transformers",
		Desc:    "add reuse_existing_token to unique index for transformers table",
		Up: `DROP INDEX IF EXISTS transformers_deleted_input_type_transform_type_function_parameters_key CASCADE;
		DROP INDEX IF EXISTS transformers_deleted_function_input_type_parameters_transform_type_key CASCADE;
		CREATE UNIQUE INDEX transformers_deleted_function_input_type_output_type_parameters_reuse_existing_token_transform_type_key ON transformers (deleted ASC, function ASC, input_type ASC, output_type ASC, parameters ASC, reuse_existing_token ASC, transform_type ASC);
			`,
		Down: `DROP INDEX transformers_deleted_function_input_type_output_type_parameters_reuse_existing_token_transform_type_key CASCADE;
		CREATE UNIQUE INDEX transformers_deleted_function_input_type_parameters_transform_type_key ON transformers (deleted ASC, function ASC, input_type ASC, parameters ASC, transform_type ASC);`,
	},
	{
		Version: 246,
		Table:   "plex_config",
		Desc:    "add default disabled microsoft login OIDC provider",
		// this id=id is weird because this table is currently 1 entry per tenant, but if sql_safe_updates=true we need something
		Up: `UPDATE plex_config SET plex_config=JSONB_SET(plex_config,
				'{oidc_providers, providers}',
				plex_config->'oidc_providers'->'providers' || '{"additional_scopes": "", "can_use_local_host_redirect": false, "client_id": "", "client_secret": "", "default_scopes": "openid profile email", "description": "Microsoft", "is_native": true, "issuer_url": "https://login.microsoftonline.com/common/v2.0", "name": "microsoft", "type": "microsoft", "use_local_host_redirect": false}'
			)
			where id=id;`,
		Down: `UPDATE plex_config SET plex_config=JSONB_SET(plex_config, '{oidc_providers,providers}', providers) FROM (
			SELECT id, JSONB_AGG(provider) AS providers FROM (
				SELECT id, provider
					FROM (
						SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'oidc_providers'->'providers') as provider
							FROM plex_config
							WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
					WHERE provider->>'type'<>'microsoft'
				) GROUP BY id
			);`,
	},
	{
		Version: 247,
		Table:   "user_column_pre_delete_values",
		Desc:    "add user_id index for deletion perf",
		Up:      `CREATE INDEX user_column_pre_delete_values_user_id_idx ON user_column_pre_delete_values (user_id ASC);`,
		Down:    `DROP INDEX user_column_pre_delete_values_user_id_idx;`,
	},
	{
		Version: 248,
		Table:   "user_column_post_delete_values",
		Desc:    "add user_id index for deletion perf",
		Up:      `CREATE INDEX user_column_post_delete_values_user_id_idx ON user_column_post_delete_values (user_id ASC);`,
		Down:    `DROP INDEX user_column_post_delete_values_user_id_idx;`,
	},
	{
		Version: 249,
		Table:   "users",
		Desc:    "add region column to users table",
		Up:      `ALTER TABLE users ADD COLUMN region VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE users DROP COLUMN region;`,
	},
	{
		Version: 250,
		Table:   "plexconfig",
		Desc:    "remove null provider configs",
		Up: `UPDATE plex_config SET plex_config=JSONB_SET(plex_config, '{plex_map,providers}', providers) FROM (
			SELECT id, JSONB_AGG(provider) AS providers FROM (
				SELECT id, JSON_REMOVE_PATH(provider, '{auth0}') as provider
					FROM (
						SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
							FROM plex_config
							WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
					WHERE provider->>'type'<>'auth0'
				UNION
				SELECT id, provider
					FROM (
						SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
							FROM plex_config
							WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
					WHERE provider->>'type'='auth0'
			) GROUP BY id
		);
		UPDATE plex_config SET plex_config=JSONB_SET(plex_config, '{plex_map,providers}', providers) FROM (
			SELECT id, JSONB_AGG(provider) AS providers FROM (
				SELECT id, JSON_REMOVE_PATH(provider, '{uc}') as provider
					FROM (
						SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
							FROM plex_config
							WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
					WHERE provider->>'type'<>'uc'
				UNION
				SELECT id, provider
					FROM (
						SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
							FROM plex_config
							WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
					WHERE provider->>'type'='uc'
			) GROUP BY id
		);
		UPDATE plex_config SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map,employee_provider,auth0}')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
		Down: `/* no op */`,
	},
	{
		Version: 251,
		Table:   "transformers",
		Desc:    "add input_type_constraints and output_type_constraints columns to transformers table",
		Up: `ALTER TABLE transformers ADD COLUMN input_type_constraints JSONB NOT NULL DEFAULT '{}':::JSONB;
		ALTER TABLE transformers ADD COLUMN output_type_constraints JSONB NOT NULL DEFAULT '{}':::JSONB;`,
		Down: `ALTER TABLE transformers DROP COLUMN input_type_constraints;
		ALTER TABLE transformers DROP COLUMN output_type_constraints;`,
	},
	{
		Version: 252,
		Table:   "accessors",
		Desc:    "add is_audit_logged column to accessors table",
		Up:      `ALTER TABLE accessors ADD COLUMN is_audit_logged BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down:    `ALTER TABLE accessors DROP COLUMN is_audit_logged;`,
	},
	{
		Version: 253,
		Table:   "access_policies",
		Desc:    "add is_autogenerated to access_policies table",
		Up:      `ALTER TABLE access_policies ADD COLUMN is_autogenerated BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE access_policies DROP COLUMN is_autogenerated;`,
	},
	{
		Version: 254,
		Table:   "idp_data_import_jobs",
		Desc:    "add idp_data_import_jobs table",
		Up: `CREATE TABLE idp_data_import_jobs (
				id UUID NOT NULL,
				created TIMESTAMP NOT NULL DEFAULT now(),
				updated TIMESTAMP NOT NULL,
				deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
				last_run_time TIMESTAMP NOT NULL,
				import_type VARCHAR NOT NULL,
				status VARCHAR NOT NULL,
				error VARCHAR NOT NULL,
				s3_bucket VARCHAR NOT NULL,
				object_key VARCHAR NOT NULL,
				expiration_minutes INT NOT NULL,
				file_size INT8 NOT NULL,
				processed_size INT8 NOT NULL,
				processed_record_count INT8 NOT NULL,
				failed_record_count INT8 NOT NULL,
				failed_records JSONB NOT NULL,
				CONSTRAINT idp_data_import_jobs_pkey PRIMARY KEY (deleted, id)
			)`,
		Down: `DROP TABLE idp_data_import_jobs;`,
	},
	{
		Version: 255,
		Table:   "data_types",
		Desc:    "Creating data_types table",
		Up: `CREATE TABLE data_types (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			name VARCHAR NOT NULL,
			description VARCHAR NOT NULL,
			concrete_data_type_id UUID NOT NULL,
			composite_attributes JSONB NOT NULL DEFAULT '{}',
			CONSTRAINT "data_types_pk" PRIMARY KEY (deleted, id),
			UNIQUE (deleted, name)
		);`,
		Down: `DROP TABLE data_types;`,
	},
	{
		Version: 256,
		Table:   "columns",
		Desc:    "add data_type_id to columns table",
		Up:      `ALTER TABLE columns ADD COLUMN data_type_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down:    `ALTER TABLE columns DROP COLUMN data_type_id;`,
	},
	{
		Version: 257,
		Table:   "transformers",
		Desc:    "add input_data_type_id and output_data_type_id to transformers table",
		Up: `ALTER TABLE transformers ADD COLUMN input_data_type_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
		          ALTER TABLE transformers ADD COLUMN output_data_type_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down: `ALTER TABLE transformers DROP COLUMN input_data_type_id;
		          ALTER TABLE transformers DROP COLUMN output_data_type_id;`,
	},
	{
		Version: 258,
		Table:   "data_types",
		Desc:    "backfill all native data types in data_types table",
		Up: `/* no-match-cols-vals */
		INSERT INTO data_types
		(id, name, description, concrete_data_type_id, updated)
		VALUES
		('e16b5ead-54db-4b42-a55f-f21907cda9e4', 'boolean', 'a boolean', 'e16b5ead-54db-4b42-a55f-f21907cda9e4', now()),
		('22b8a1b6-e5a2-4c3c-9a99-746f0345b727', 'integer', 'an integer', '22b8a1b6-e5a2-4c3c-9a99-746f0345b727', now()),
		('d26b6d52-a8d7-4c2f-9efc-394eb90a3294', 'string', 'a string', 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294', now()),
		('8a84f041-c605-4ebf-b552-9e14f51c9e54', 'email', 'an email address', 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294', now()),
		('66a87f97-32c4-4ccc-91da-d8c880e21e5a', 'timestamp', 'a timestamp', '66a87f97-32c4-4ccc-91da-d8c880e21e5a', now()),
		('3e2546c0-14d6-49d3-8b95-a5000bb4ad6a', 'date', 'a date', '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a', now()),
		('76f0685b-dd42-4b3f-8c33-4c72e4eff73e', 'birthdate', 'a birthdate', '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a', now()),
		('d036bbba-6012-4d74-b7c4-9a2bbc09a749', 'uuid', 'a UUID', 'd036bbba-6012-4d74-b7c4-9a2bbc09a749', now()),
		('fba9f9bb-b9e0-4258-9fb8-6777792dbeba', 'ssn', 'a social security number', 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294', now()),
		('db6f892c-be7a-4779-947d-2cfcc699f48c', 'address', 'an address', 'db6f892c-be7a-4779-947d-2cfcc699f48c', now()),
		('ae962c31-2ca7-42e1-814b-32e6493dba82', 'phonenumber', 'a phone number', 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294', now()),
		('97f0ab8a-f2fd-43da-9feb-3d1f8aacc042', 'e164_phonenumber', 'an E.164 phone number', 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294', now());`,
		Down: `/* no op */`,
	},
	{
		Version: 259,
		Table:   "columns",
		Desc:    "backfill data type ids for native data types in columns table",
		Up: `UPDATE columns SET data_type_id = 'e16b5ead-54db-4b42-a55f-f21907cda9e4' WHERE type = 1;
		UPDATE columns SET data_type_id = '22b8a1b6-e5a2-4c3c-9a99-746f0345b727' WHERE type = 2;
		UPDATE columns SET data_type_id = 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294' WHERE type = 100;
		UPDATE columns SET data_type_id = '8a84f041-c605-4ebf-b552-9e14f51c9e54' WHERE type = 101;
		UPDATE columns SET data_type_id = '66a87f97-32c4-4ccc-91da-d8c880e21e5a' WHERE type = 200;
		UPDATE columns SET data_type_id = '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a' WHERE type = 201;
		UPDATE columns SET data_type_id = '76f0685b-dd42-4b3f-8c33-4c72e4eff73e' WHERE type = 202;
		UPDATE columns SET data_type_id = 'd036bbba-6012-4d74-b7c4-9a2bbc09a749' WHERE type = 300;
		UPDATE columns SET data_type_id = 'fba9f9bb-b9e0-4258-9fb8-6777792dbeba' WHERE type = 301;
		UPDATE columns SET data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c' WHERE type = 401;
		UPDATE columns SET data_type_id = 'ae962c31-2ca7-42e1-814b-32e6493dba82' WHERE type = 402;
		UPDATE columns SET data_type_id = '97f0ab8a-f2fd-43da-9feb-3d1f8aacc042' WHERE type = 403;`,
		Down: `/* no op */`,
	},
	{
		Version: 260,
		Table:   "transformers",
		Desc:    "backfill data type ids for native data types in transformers table",
		Up: `UPDATE transformers SET input_data_type_id = 'e16b5ead-54db-4b42-a55f-f21907cda9e4' WHERE input_type = 1;
		UPDATE transformers SET input_data_type_id = '22b8a1b6-e5a2-4c3c-9a99-746f0345b727' WHERE input_type = 2;
		UPDATE transformers SET input_data_type_id = 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294' WHERE input_type = 100;
		UPDATE transformers SET input_data_type_id = '8a84f041-c605-4ebf-b552-9e14f51c9e54' WHERE input_type = 101;
		UPDATE transformers SET input_data_type_id = '66a87f97-32c4-4ccc-91da-d8c880e21e5a' WHERE input_type = 200;
		UPDATE transformers SET input_data_type_id = '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a' WHERE input_type = 201;
		UPDATE transformers SET input_data_type_id = '76f0685b-dd42-4b3f-8c33-4c72e4eff73e' WHERE input_type = 202;
		UPDATE transformers SET input_data_type_id = 'd036bbba-6012-4d74-b7c4-9a2bbc09a749' WHERE input_type = 300;
		UPDATE transformers SET input_data_type_id = 'fba9f9bb-b9e0-4258-9fb8-6777792dbeba' WHERE input_type = 301;
		UPDATE transformers SET input_data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c' WHERE input_type = 401;
		UPDATE transformers SET input_data_type_id = 'ae962c31-2ca7-42e1-814b-32e6493dba82' WHERE input_type = 402;
		UPDATE transformers SET input_data_type_id = '97f0ab8a-f2fd-43da-9feb-3d1f8aacc042' WHERE input_type = 403;
		UPDATE transformers SET output_data_type_id = 'e16b5ead-54db-4b42-a55f-f21907cda9e4' WHERE output_type = 1;
		UPDATE transformers SET output_data_type_id = '22b8a1b6-e5a2-4c3c-9a99-746f0345b727' WHERE output_type = 2;
		UPDATE transformers SET output_data_type_id = 'd26b6d52-a8d7-4c2f-9efc-394eb90a3294' WHERE output_type = 100;
		UPDATE transformers SET output_data_type_id = '8a84f041-c605-4ebf-b552-9e14f51c9e54' WHERE output_type = 101;
		UPDATE transformers SET output_data_type_id = '66a87f97-32c4-4ccc-91da-d8c880e21e5a' WHERE output_type = 200;
		UPDATE transformers SET output_data_type_id = '3e2546c0-14d6-49d3-8b95-a5000bb4ad6a' WHERE output_type = 201;
		UPDATE transformers SET output_data_type_id = '76f0685b-dd42-4b3f-8c33-4c72e4eff73e' WHERE output_type = 202;
		UPDATE transformers SET output_data_type_id = 'd036bbba-6012-4d74-b7c4-9a2bbc09a749' WHERE output_type = 300;
		UPDATE transformers SET output_data_type_id = 'fba9f9bb-b9e0-4258-9fb8-6777792dbeba' WHERE output_type = 301;
		UPDATE transformers SET output_data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c' WHERE output_type = 401;
		UPDATE transformers SET output_data_type_id = 'ae962c31-2ca7-42e1-814b-32e6493dba82' WHERE output_type = 402;
		UPDATE transformers SET output_data_type_id = '97f0ab8a-f2fd-43da-9feb-3d1f8aacc042' WHERE output_type = 403;`,
		Down: `/* no op */`,
	},
	{
		Version: 261,
		Table:   "data_sources",
		Desc:    "add data_sources table",
		Up: `CREATE TABLE data_sources (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			type VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			config JSONB NOT NULL,
			metadata JSONB NOT NULL,
			PRIMARY KEY (deleted, id)
			);`,
		Down: `DROP TABLE data_sources;`,
	},
	{
		Version: 262,
		Table:   "data_source_elements",
		Desc:    "add data_source_elements table",
		Up: `CREATE TABLE data_source_elements (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			data_source_id UUID NOT NULL,
			path VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			metadata JSONB NOT NULL,
			PRIMARY KEY (deleted, id)
			);`,
		Down: `DROP TABLE data_source_elements;`,
	},
	{
		Version: 263,
		Table:   "access_policies",
		Desc:    "add metadata to access_policies",
		Up:      `ALTER TABLE access_policies ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::JSONB;`,
		Down:    `ALTER TABLE access_policies DROP COLUMN metadata;`,
	},
	{
		Version: 264,
		Table:   "data_types",
		Desc:    "backfill canonical_address data type",
		Up: `/* no-match-cols-vals */
		INSERT INTO data_types
		(id, name, description, composite_attributes, concrete_data_type_id, updated)
		VALUES
		('33dc5de6-94b6-4f08-94b6-e04d1f981671'::UUID, 'canonical_address', 'a canonical address', '{"fields": [{"camel_case_name": "AdministrativeArea", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Administrative_Area", "required": false, "struct_name": "administrative_area"}, {"camel_case_name": "Country", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Country", "required": false, "struct_name": "country"}, {"camel_case_name": "DependentLocality", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Dependent_Locality", "required": false, "struct_name": "dependent_locality"}, {"camel_case_name": "ID", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": true, "name": "ID", "required": false, "struct_name": "id"}, {"camel_case_name": "Locality", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Locality", "required": false, "struct_name": "locality"}, {"camel_case_name": "Name", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Name", "required": false, "struct_name": "name"}, {"camel_case_name": "Organization", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Organization", "required": false, "struct_name": "organization"}, {"camel_case_name": "PostCode", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Post_Code", "required": false, "struct_name": "post_code"}, {"camel_case_name": "SortingCode", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Sorting_Code", "required": false, "struct_name": "sorting_code"}, {"camel_case_name": "StreetAddressLine1", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Street_Address_Line_1", "required": false, "struct_name": "street_address_line_1"}, {"camel_case_name": "StreetAddressLine2", "data_type_id": "d26b6d52-a8d7-4c2f-9efc-394eb90a3294", "ignore_for_uniqueness": false, "name": "Street_Address_Line_2", "required": false, "struct_name": "street_address_line_2"}], "include_id": true}'::JSONB, 'd81658a7-848a-4504-9c6e-5fa17f90f1a6'::UUID, now());`,
		Down: `DELETE FROM data_types WHERE id = '33dc5de6-94b6-4f08-94b6-e04d1f981671' AND deleted = '0001-01-01 00:00:00'::TIMESTAMP;`,
	},
	{
		Version: 265,
		Table:   "columns",
		Desc:    "switch columns of type address to use canonical_address data type",
		Up: `UPDATE columns
		SET type = 400,
		data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671'::UUID,
		attributes = JSONB_SET(attributes,
		'{constraints, fields}',
		'[{"camel_case_name": "AdministrativeArea", "ignore_for_uniqueness": false, "name": "Administrative_Area", "required": false, "struct_name": "administrative_area", "type": 100}, {"camel_case_name": "Country", "ignore_for_uniqueness": false, "name": "Country", "required": false, "struct_name": "country", "type": 100}, {"camel_case_name": "DependentLocality", "ignore_for_uniqueness": false, "name": "Dependent_Locality", "required": false, "struct_name": "dependent_locality", "type": 100}, {"camel_case_name": "ID", "ignore_for_uniqueness": true, "name": "ID", "required": false, "struct_name": "id", "type": 100}, {"camel_case_name": "Locality", "ignore_for_uniqueness": false, "name": "Locality", "required": false, "struct_name": "locality", "type": 100}, {"camel_case_name": "Name", "ignore_for_uniqueness": false, "name": "Name", "required": false, "struct_name": "name", "type": 100}, {"camel_case_name": "Organization", "ignore_for_uniqueness": false, "name": "Organization", "required": false, "struct_name": "organization", "type": 100}, {"camel_case_name": "PostCode", "ignore_for_uniqueness": false, "name": "Post_Code", "required": false, "struct_name": "post_code", "type": 100}, {"camel_case_name": "SortingCode", "ignore_for_uniqueness": false, "name": "Sorting_Code", "required": false, "struct_name": "sorting_code", "type": 100}, {"camel_case_name": "StreetAddressLine1", "ignore_for_uniqueness": false, "name": "Street_Address_Line_1", "required": false, "struct_name": "street_address_line_1", "type": 100}, {"camel_case_name": "StreetAddressLine2", "ignore_for_uniqueness": false, "name": "Street_Address_Line_2", "required": false, "struct_name": "street_address_line_2", "type": 100}]'::JSONB)
		WHERE type = 401;`,
		Down: `UPDATE columns
		SET type = 401,
		data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID,
		attributes = JSONB_SET(attributes, '{constraints, fields}', '[]'::JSONB)
		WHERE data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671';`,
	},
	{
		Version: 266,
		Table:   "transformers",
		Desc:    "switch transformers with input type address to use canonical_address data type",
		Up: `UPDATE transformers
		SET input_type = 400,
		input_data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671',
		input_type_constraints = '{"fields": [{"camel_case_name": "AdministrativeArea", "ignore_for_uniqueness": false, "name": "Administrative_Area", "required": false, "struct_name": "administrative_area", "type": 100}, {"camel_case_name": "Country", "ignore_for_uniqueness": false, "name": "Country", "required": false, "struct_name": "country", "type": 100}, {"camel_case_name": "DependentLocality", "ignore_for_uniqueness": false, "name": "Dependent_Locality", "required": false, "struct_name": "dependent_locality", "type": 100}, {"camel_case_name": "ID", "ignore_for_uniqueness": false, "name": "ID", "required": false, "struct_name": "id", "type": 100}, {"camel_case_name": "Locality", "ignore_for_uniqueness": false, "name": "Locality", "required": false, "struct_name": "locality", "type": 100}, {"camel_case_name": "Name", "ignore_for_uniqueness": false, "name": "Name", "required": false, "struct_name": "name", "type": 100}, {"camel_case_name": "Organization", "ignore_for_uniqueness": false, "name": "Organization", "required": false, "struct_name": "organization", "type": 100}, {"camel_case_name": "PostCode", "ignore_for_uniqueness": false, "name": "Post_Code", "required": false, "struct_name": "post_code", "type": 100}, {"camel_case_name": "SortingCode", "ignore_for_uniqueness": false, "name": "Sorting_Code", "required": false, "struct_name": "sorting_code", "type": 100}, {"camel_case_name": "StreetAddressLine1", "ignore_for_uniqueness": false, "name": "Street_Address_Line_1", "required": false, "struct_name": "street_address_line_1", "type": 100}, {"camel_case_name": "StreetAddressLine2", "ignore_for_uniqueness": false, "name": "Street_Address_Line_2", "required": false, "struct_name": "street_address_line_2", "type": 100}], "immutable_required": false, "partial_updates": false, "unique_id_required": false, "unique_required": false}'::JSONB
		WHERE input_type = 401;`,
		Down: `UPDATE transformers
		SET input_type = 401,
		input_data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID,
		input_type_constraints = '{"fields": null, "immutable_required": false, "partial_updates": false, "unique_id_required": false, "unique_required": false}'::JSONB
		WHERE input_data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671';`,
	},
	{
		Version: 267,
		Table:   "transformers",
		Desc:    "switch transformers with output type address to use canonical_address data type",
		Up: `UPDATE transformers
		SET output_type = 400,
		output_data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671',
		output_type_constraints = '{"fields": [{"camel_case_name": "AdministrativeArea", "ignore_for_uniqueness": false, "name": "Administrative_Area", "required": false, "struct_name": "administrative_area", "type": 100}, {"camel_case_name": "Country", "ignore_for_uniqueness": false, "name": "Country", "required": false, "struct_name": "country", "type": 100}, {"camel_case_name": "DependentLocality", "ignore_for_uniqueness": false, "name": "Dependent_Locality", "required": false, "struct_name": "dependent_locality", "type": 100}, {"camel_case_name": "ID", "ignore_for_uniqueness": false, "name": "ID", "required": false, "struct_name": "id", "type": 100}, {"camel_case_name": "Locality", "ignore_for_uniqueness": false, "name": "Locality", "required": false, "struct_name": "locality", "type": 100}, {"camel_case_name": "Name", "ignore_for_uniqueness": false, "name": "Name", "required": false, "struct_name": "name", "type": 100}, {"camel_case_name": "Organization", "ignore_for_uniqueness": false, "name": "Organization", "required": false, "struct_name": "organization", "type": 100}, {"camel_case_name": "PostCode", "ignore_for_uniqueness": false, "name": "Post_Code", "required": false, "struct_name": "post_code", "type": 100}, {"camel_case_name": "SortingCode", "ignore_for_uniqueness": false, "name": "Sorting_Code", "required": false, "struct_name": "sorting_code", "type": 100}, {"camel_case_name": "StreetAddressLine1", "ignore_for_uniqueness": false, "name": "Street_Address_Line_1", "required": false, "struct_name": "street_address_line_1", "type": 100}, {"camel_case_name": "StreetAddressLine2", "ignore_for_uniqueness": false, "name": "Street_Address_Line_2", "required": false, "struct_name": "street_address_line_2", "type": 100}], "immutable_required": false, "partial_updates": false, "unique_id_required": false, "unique_required": false}'::JSONB
		WHERE output_type = 401;`,
		Down: `UPDATE transformers
		SET output_type = 401,
		output_data_type_id = 'db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID,
		output_type_constraints = '{"fields": null, "immutable_required": false, "partial_updates": false, "unique_id_required": false, "unique_required": false}'::JSONB
		WHERE output_data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671';`,
	},
	{
		Version: 268,
		Table:   "data_types",
		Desc:    "remove address data type",
		Up:      `DELETE FROM data_types WHERE id = 'db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID AND deleted = '0001-01-01 00:00:00'::TIMESTAMP;`,
		Down: `/* no-match-cols-vals */
		INSERT INTO data_types
		(id, name, description, concrete_data_type_id, updated)
		VALUES
		('db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID, 'address', 'an address', 'db6f892c-be7a-4779-947d-2cfcc699f48c'::UUID, now());`,
	},
	{
		Version: 269,
		Table:   "user_cleanup_candidates",
		Desc:    "create user_cleanup_candidates table",
		Up: `CREATE TABLE user_cleanup_candidates (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			user_id UUID NOT NULL,
			cleanup_reason INT8 NOT NULL,
			PRIMARY KEY (deleted, id)
			);`,
		Down: `DROP TABLE user_cleanup_candidates;`,
	},
	{
		Version: 270,
		Table:   "columns",
		Desc:    "set column fields for canonical address columns that did not have any attributes",
		Up: `UPDATE columns
			SET attributes = '{"constraints": {"fields": [{"camel_case_name": "AdministrativeArea", "ignore_for_uniqueness": false, "name": "Administrative_Area", "required": false, "struct_name": "administrative_area", "type": 100}, {"camel_case_name": "Country", "ignore_for_uniqueness": false, "name": "Country", "required": false, "struct_name": "country", "type": 100}, {"camel_case_name": "DependentLocality", "ignore_for_uniqueness": false, "name": "Dependent_Locality", "required": false, "struct_name": "dependent_locality", "type": 100}, {"camel_case_name": "ID", "ignore_for_uniqueness": true, "name": "ID", "required": false, "struct_name": "id", "type": 100}, {"camel_case_name": "Locality", "ignore_for_uniqueness": false, "name": "Locality", "required": false, "struct_name": "locality", "type": 100}, {"camel_case_name": "Name", "ignore_for_uniqueness": false, "name": "Name", "required": false, "struct_name": "name", "type": 100}, {"camel_case_name": "Organization", "ignore_for_uniqueness": false, "name": "Organization", "required": false, "struct_name": "organization", "type": 100}, {"camel_case_name": "PostCode", "ignore_for_uniqueness": false, "name": "Post_Code", "required": false, "struct_name": "post_code", "type": 100}, {"camel_case_name": "SortingCode", "ignore_for_uniqueness": false, "name": "Sorting_Code", "required": false, "struct_name": "sorting_code", "type": 100}, {"camel_case_name": "StreetAddressLine1", "ignore_for_uniqueness": false, "name": "Street_Address_Line_1", "required": false, "struct_name": "street_address_line_1", "type": 100}, {"camel_case_name": "StreetAddressLine2", "ignore_for_uniqueness": false, "name": "Street_Address_Line_2", "required": false, "struct_name": "street_address_line_2", "type": 100}], "immutable_required": false, "partial_updates": false, "unique_id_required": false, "unique_required": false}}'::JSONB
			WHERE deleted = '0001-01-01 00:00:00'
			AND data_type_id = '33dc5de6-94b6-4f08-94b6-e04d1f981671'
			AND attributes = '{}'::JSONB;`,
		Down: `/* no-op */`,
	},
	{
		Version: 271,
		Table:   "columns",
		Desc:    "add multiple table support for columns",
		Up: `ALTER TABLE columns ADD COLUMN tbl VARCHAR NOT NULL DEFAULT 'users';
			DROP INDEX columns_name_deleted_key CASCADE;
			ALTER TABLE columns ADD CONSTRAINT columns_deleted_tbl_name_key UNIQUE (deleted, tbl, name);`,
		Down: `DROP INDEX columns_deleted_tbl_name_key CASCADE;
			ALTER TABLE columns ADD CONSTRAINT columns_name_deleted_key UNIQUE (name, deleted);
			ALTER TABLE columns DROP COLUMN tbl;`,
	},
	{
		Version: 272,
		Table:   "access_policies",
		Desc:    "add thresholds to access_policies",
		Up:      `ALTER TABLE access_policies ADD COLUMN thresholds JSONB NOT NULL DEFAULT '{}'::JSONB;`,
		Down:    `ALTER TABLE access_policies DROP COLUMN thresholds;`,
	},
	{
		Version: 273,
		Table:   "sqlshim_databases",
		Desc:    "add sqlshim_databases table",
		Up: `CREATE TABLE sqlshim_databases (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			name VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			host VARCHAR NOT NULL,
			port INT NOT NULL,
			username VARCHAR NOT NULL,
			password VARCHAR NOT NULL,
			PRIMARY KEY (deleted, id)
			);`,
		Down: `DROP TABLE sqlshim_databases;`,
	},
	{
		Version: 274,
		Table:   "columns",
		Desc:    "Add sqlshim_database_id to columns table",
		Up:      `ALTER TABLE columns ADD COLUMN sqlshim_database_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down:    `ALTER TABLE columns DROP COLUMN sqlshim_database_id;`,
	},
	{
		Version: 275,
		Table:   "columns",
		Desc:    "Add default_transformer_id and default_token_access_policy_id to columns table",
		Up: `ALTER TABLE columns ADD COLUMN default_transformer_id UUID NOT NULL DEFAULT 'c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a'::UUID,
			ADD COLUMN default_token_access_policy_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
			ADD COLUMN access_policy_id UUID NOT NULL DEFAULT '3f380e42-0b21-4570-a312-91e1b80386fa'::UUID;`,
		Down: `ALTER TABLE columns DROP COLUMN default_transformer_id,
			DROP COLUMN default_token_access_policy_id,
			DROP COLUMN access_policy_id;`,
	},
	{
		Version: 276,
		Table:   "sqlshim_databases",
		Desc:    "add schemas_updated and schemas to sqlshim_databases",
		Up: `ALTER TABLE sqlshim_databases ADD COLUMN schemas_updated TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			ADD COLUMN schemas_update_scheduled TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			ADD COLUMN schemas VARCHAR[] NOT NULL DEFAULT '{}'::VARCHAR[];`,
		Down: `ALTER TABLE sqlshim_databases DROP COLUMN schemas_updated,
			DROP COLUMN schemas_update_scheduled,
			DROP COLUMN schemas;`,
	},
	{
		Version: 277,
		Table:   "accessors",
		Desc:    "add is_autogenerated to accessors table",
		Up:      `ALTER TABLE accessors ADD COLUMN is_autogenerated BOOLEAN NOT NULL DEFAULT false;`,
		Down:    `ALTER TABLE accessors DROP COLUMN is_autogenerated;`,
	},
	{
		Version: 278,
		Table:   "edges",
		Desc:    "Add index to make reads by updated time efficient",
		Up:      `CREATE INDEX edges_updated_time ON edges (updated) INCLUDE (created, edge_type_id, source_object_id, target_object_id) ;`,
		Down:    `DROP INDEX edges_updated_time;`,
	},
	{
		Version: 279,
		Table:   "accessors",
		Desc:    "Add are_column_access_policies_overridden and token_access_policy_ids to accessors",
		Up: `ALTER TABLE accessors ADD COLUMN are_column_access_policies_overridden BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN token_access_policy_ids UUID[] NOT NULL DEFAULT ARRAY[]::uuid[];`,
		Down: `ALTER TABLE accessors DROP COLUMN are_column_access_policies_overridden,
			DROP COLUMN token_access_policy_ids;`,
	},
	{
		Version: 280,
		Table:   "accessors",
		Desc:    "Migrate from token_access_policy_id column to token_access_policy_ids column",
		Up: `UPDATE accessors SET token_access_policy_ids = string_to_array(btrim(repeat(token_access_policy_id::STRING||',', array_length(column_ids, 1)), ','), ',')::UUID[]
		WHERE deleted = '0001-01-01' AND array_length(token_access_policy_ids, 1) IS NULL;`,
		Down: `/* no-op */`,
	},
	{
		Version: 281,
		Table:   "user_column_pre_delete_values",
		Desc:    "add trigram indices for varchar_value and varchar_unique_value",
		Up: `CREATE INDEX user_column_pre_delete_values_column_varchar_trgm ON user_column_pre_delete_values USING gin(column_id, varchar_value gin_trgm_ops);
		     CREATE INDEX user_column_pre_delete_values_column_varchar_unique_trgm ON user_column_pre_delete_values USING gin(column_id, varchar_unique_value gin_trgm_ops);`,
		Down: `DROP INDEX user_column_pre_delete_values_column_varchar_trgm;
			DROP INDEX user_column_pre_delete_values_column_varchar_unique_trgm;`,
	},
	{
		Version: 282,
		Table:   "shim_object_stores",
		Desc:    "add shim_object_stores table",
		Up: `CREATE TABLE shim_object_stores (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			name VARCHAR NOT NULL,
			type VARCHAR NOT NULL,
			region VARCHAR NOT NULL,
			access_key_id VARCHAR NOT NULL,
			secret_access_key VARCHAR NOT NULL,
			access_policy_id UUID NOT NULL,
			PRIMARY KEY (deleted, id)
			);`,
		Down: `DROP TABLE shim_object_stores;`,
	},
	{
		Version: 283,
		Table:   "transformers",
		Desc:    "add version to transformers table",
		Up: `ALTER TABLE transformers ADD COLUMN version INT NOT NULL DEFAULT 0;
			ALTER TABLE transformers DROP CONSTRAINT IF EXISTS transformers_deleted_function_input_type_output_type_parameters_reuse_existing_token_transform_type_key CASCADE;
			ALTER TABLE transformers DROP CONSTRAINT IF EXISTS transformers_deleted_name_key CASCADE;
			ALTER TABLE transformers ADD CONSTRAINT transformers_name_version_deleted_key UNIQUE (name, version, deleted);`,
		Down: `ALTER TABLE transformers DROP CONSTRAINT IF EXISTS transformers_name_version_deleted_key CASCADE;
			ALTER TABLE transformers ADD CONSTRAINT transformers_deleted_name_key UNIQUE (deleted, name);
			ALTER TABLE transformers ADD CONSTRAINT transformers_deleted_function_input_type_output_type_parameters_reuse_existing_token_transform_type_key UNIQUE (deleted, function, input_type, output_type, parameters, reuse_existing_token, transform_type);
			ALTER TABLE transformers DROP COLUMN version;`,
	},
	{
		Version: 284,
		Table:   "transformers",
		Desc:    "change primary key on transformers table",
		Up: `ALTER TABLE transformers DROP CONSTRAINT transformers_pkey;
			ALTER TABLE transformers ADD CONSTRAINT transformers_pkey PRIMARY KEY (deleted, id, version);`,
		Down: `ALTER TABLE transformers DROP CONSTRAINT transformers_pkey;
			ALTER TABLE transformers ADD CONSTRAINT transformers_pkey PRIMARY KEY (deleted, id);`,
	},
	{
		Version: 285,
		Table:   "transformers",
		Desc:    "add transformers_old_pkey to transformers table",
		Up:      `ALTER TABLE transformers ADD CONSTRAINT transformers_old_pkey UNIQUE (deleted, id);`,
		Down:    `ALTER TABLE transformers DROP CONSTRAINT IF EXISTS transformers_old_pkey CASCADE;`,
	},
	{
		Version: 286,
		Table:   "token_records",
		Desc:    "add transformer_version to token_records table",
		Up:      `ALTER TABLE token_records ADD COLUMN transformer_version INT NOT NULL DEFAULT 0`,
		Down:    `ALTER TABLE token_records DROP COLUMN transformer_version`,
	},
	{
		Version: 287,
		Table:   "shim_object_stores",
		Desc:    "add role_arn to shim_object_stores",
		Up:      `ALTER TABLE shim_object_stores ADD COLUMN role_arn VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE shim_object_stores DROP COLUMN role_arn;`,
	},
	{
		Version: 288,
		Table:   "user_column_pre_delete_values",
		Desc:    "add value_type column",
		Up:      `ALTER TABLE user_column_pre_delete_values ADD COLUMN value_type INT8 NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE user_column_pre_delete_values DROP COLUMN value_type`,
	},
	{
		Version: 289,
		Table:   "user_column_post_delete_values",
		Desc:    "add value_type column",
		Up:      `ALTER TABLE user_column_post_delete_values ADD COLUMN value_type INT8 NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE user_column_post_delete_values DROP COLUMN value_type`,
	},
	{
		Version: 290,
		Table:   "columns",
		Desc:    "add default value for type",
		Up:      `ALTER TABLE columns ALTER COLUMN type SET DEFAULT 0;`,
		Down:    `ALTER TABLE columns ALTER COLUMN type DROP DEFAULT;`,
	},
	{
		Version: 291,
		Table:   "transformers",
		Desc:    "add default value for input_type and output_type",
		Up: `ALTER TABLE transformers ALTER COLUMN input_type SET DEFAULT 0;
			  ALTER TABLE transformers ALTER COLUMN output_type SET DEFAULT 0;
			  `,
		Down: `ALTER TABLE transformers ALTER COLUMN input_type DROP DEFAULT;
			  ALTER TABLE transformers ALTER COLUMN output_type DROP DEFAULT;
			  `,
	},
	{
		Version: 292,
		Table:   "transformers",
		Desc:    "remove temporary index transformers_old_pkey",
		Up:      `ALTER TABLE transformers DROP CONSTRAINT IF EXISTS transformers_old_pkey CASCADE;`,
		Down:    `ALTER TABLE transformers ADD CONSTRAINT transformers_old_pkey UNIQUE (deleted, id);`,
	},
	{
		Version: 293,
		Table:   "columns",
		Desc:    "add search_indexed column",
		Up:      `ALTER TABLE columns ADD COLUMN search_indexed BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down:    `ALTER TABLE columns DROP COLUMN search_indexed;`,
	},
	{
		Version: 294,
		Table:   "accessors",
		Desc:    "add search_column_id and use_search_index columns",
		Up: `ALTER TABLE accessors ADD COLUMN search_column_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;
			ALTER TABLE accessors ADD COLUMN use_search_index BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down: `ALTER TABLE accessors DROP COLUMN search_column_id;
			 ALTER TABLE accessors DROP COLUMN use_search_index;`,
	},
	{
		Version: 295,
		Table:   "columns",
		Desc:    "remove fields from attributes.constraints",
		Up:      `UPDATE columns SET attributes=attributes::JSONB #- '{constraints, fields}' WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP AND attributes != '{}';`,
		Down:    `/* noop */`,
	},
	{
		Version: 296,
		Table:   "columns",
		Desc:    "remove type",
		Up:      `ALTER TABLE columns DROP COLUMN type;`,
		Down:    `ALTER TABLE columns ADD COLUMN type INT8 NOT NULL DEFAULT 0;`,
	},
	{
		Version: 297,
		Table:   "transformers",
		Desc:    "remove input_type, input_type_constraints, output_type, and output_type_constraints",
		Up: `ALTER TABLE transformers DROP COLUMN input_type;
			ALTER TABLE transformers DROP COLUMN input_type_constraints;
			ALTER TABLE transformers DROP COLUMN output_type;
			ALTER TABLE transformers DROP COLUMN output_type_constraints;`,
		Down: `ALTER TABLE transformers ADD COLUMN input_type INT8 NOT NULL DEFAULT 0;
			ALTER TABLE transformers ADD COLUMN input_type_constraints JSONB NOT NULL DEFAULT '{}';
			ALTER TABLE transformers ADD COLUMN output_type INT8 NOT NULL DEFAULT 0;
			ALTER TABLE transformers ADD COLUMN output_type_constraints JSONB NOT NULL DEFAULT '{}';`,
	},
	{
		Version: 298,
		Table:   "user_column_pre_delete_values",
		Desc:    "backfill value_type",
		Up: `UPDATE user_column_pre_delete_values SET value_type=1 WHERE varchar_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=2 WHERE varchar_unique_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=3 WHERE boolean_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=4 WHERE int_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=5 WHERE int_unique_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=6 WHERE timestamp_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=7 WHERE uuid_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=8 WHERE uuid_unique_value IS NOT NULL AND value_type=0;
			UPDATE user_column_pre_delete_values SET value_type=9 WHERE jsonb_value IS NOT NULL AND value_type=0;`,
		Down: `/* noop */`,
	},
	{
		Version: 299,
		Table:   "user_column_post_delete_values",
		Desc:    "backfill value_type",
		Up: `UPDATE user_column_post_delete_values SET value_type=1 WHERE varchar_value IS NOT NULL AND value_type=0;
			UPDATE user_column_post_delete_values SET value_type=3 WHERE boolean_value IS NOT NULL AND value_type=0;
			UPDATE user_column_post_delete_values SET value_type=4 WHERE int_value IS NOT NULL AND value_type=0;
			UPDATE user_column_post_delete_values SET value_type=6 WHERE timestamp_value IS NOT NULL AND value_type=0;
			UPDATE user_column_post_delete_values SET value_type=7 WHERE uuid_value IS NOT NULL AND value_type=0;
			UPDATE user_column_post_delete_values SET value_type=9 WHERE jsonb_value IS NOT NULL AND value_type=0;`,
		Down: `/* noop */`,
	},
	{
		Version: 300,
		Table:   "policy_secrets",
		Desc:    "add policy_secrets table",
		Up: `CREATE	TABLE policy_secrets (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL DEFAULT now(),
			name VARCHAR NOT NULL,
			value VARCHAR NOT NULL,
			PRIMARY KEY (deleted, id),
			UNIQUE (deleted, name)
		);`,
		Down: `DROP TABLE policy_secrets;`,
	},
	{
		Version: 301,
		Table:   "access_policy_templates",
		Desc:    "Remove unique constraint on function",
		Up:      `ALTER TABLE access_policy_templates DROP CONSTRAINT IF EXISTS access_policy_templates_deleted_function_version_key CASCADE;`,
		Down:    `ALTER TABLE access_policy_templates ADD CONSTRAINT access_policy_templates_deleted_function_version_key UNIQUE (deleted, function, version);`,
	},
	{
		Version: 302,
		Table:   "user_search_indices",
		Desc:    "add user_search_indices table",
		Up: `CREATE TABLE user_search_indices (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
                        created TIMESTAMP NOT NULL DEFAULT now(),
                        updated TIMESTAMP NOT NULL DEFAULT now(),
			name VARCHAR NOT NULL,
			description VARCHAR,
			data_life_cycle_state INT NOT NULL,
			type VARCHAR NOT NULL,
			settings JSONB NOT NULL DEFAULT '{}'::JSONB,
			column_ids UUID[] NULL,
			change_feed_job_id INT NOT NULL DEFAULT 0,
			bootstrapped TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			enabled TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			searchable TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
			PRIMARY KEY (deleted, id),
			UNIQUE (deleted, name)
		);`,
		Down: `DROP TABLE user_search_indices;`,
	},
	{
		Version: 303,
		Table:   "accessor_search_indices",
		Desc:    "add accessor_search_indices table",
		Up: `CREATE TABLE accessor_search_indices (
			id UUID NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP,
                        created TIMESTAMP NOT NULL DEFAULT now(),
                        updated TIMESTAMP NOT NULL DEFAULT now(),
			user_search_index_id UUID NOT NULL,
			query_type VARCHAR NOT NULL,
			PRIMARY KEY (deleted, id)
		);`,
		Down: `DROP TABLE accessor_search_indices;`,
	},
	{
		Version: 304,
		Table:   "user_search_indices",
		Desc:    "make change_feed_job_id explicitly bigint",
		Up:      `ALTER TABLE user_search_indices ALTER COLUMN change_feed_job_id TYPE BIGINT;`,
		Down:    `ALTER TABLE user_search_indices ALTER COLUMN change_feed_job_id TYPE INT;`,
	},
	{
		Version: 305,
		Table:   "user_column_pre_delete_values",
		Desc:    "add some more indexes",
		Up: `CREATE INDEX user_column_pre_delete_values_column_varchar ON user_column_pre_delete_values (column_id, varchar_value);
			CREATE INDEX user_column_pre_delete_values_column_unique_varchar ON user_column_pre_delete_values (column_id, varchar_unique_value);`,
		Down: `DROP INDEX user_column_pre_delete_values_column_varchar;
			DROP INDEX user_column_pre_delete_values_column_unique_varchar;`,
	},
	{
		Version: 306,
		Table:   "user_search_indices",
		Desc:    "add last_bootstrapped_user_id to user_search_indices table",
		Up:      `ALTER TABLE user_search_indices ADD COLUMN last_bootstrapped_user_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down:    `ALTER TABLE user_search_indices DROP COLUMN last_bootstrapped_user_id;`,
	},
	{
		Version: 307,
		Table:   "user_search_indices",
		Desc:    "add last_bootstrapped_value_id to user_search_indices table",
		Up:      `ALTER TABLE user_search_indices ADD COLUMN last_bootstrapped_value_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
		Down:    `ALTER TABLE user_search_indices DROP COLUMN last_bootstrapped_value_id;`,
	},
	{
		Version: 308,
		Table:   "user_search_indices",
		Desc:    "drop last_bootstrapped_user_id from user_search_indices table",
		Up:      `ALTER TABLE user_search_indices DROP COLUMN last_bootstrapped_user_id;`,
		Down:    `ALTER TABLE user_search_indices ADD COLUMN last_bootstrapped_user_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID;`,
	},
	{
		Version: 309,
		Table:   "columns",
		Desc:    "add sqlshim_database_id to unique constraint",
		Up: `ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_deleted_tbl_name_key CASCADE;
			ALTER TABLE columns DROP CONSTRAINT IF EXISTS columns_deleted_table_name_key CASCADE;
			ALTER TABLE columns ADD CONSTRAINT columns_deleted_sqlshim_database_id_tbl_name_key UNIQUE (deleted, sqlshim_database_id, tbl, name);`,
		Down: `ALTER TABLE columns DROP CONSTRAINT columns_deleted_sqlshim_database_id_tbl_name_key CASCADE;
			ALTER TABLE columns ADD CONSTRAINT columns_deleted_tbl_name_key UNIQUE (deleted, tbl, name);`,
	},
	{
		Version: 310,
		Table:   "user_search_indices",
		Desc:    "add last_regional_bootstrapped_value_ids to user_search_indices table",
		Up:      `ALTER TABLE user_search_indices ADD COLUMN last_regional_bootstrapped_value_ids JSONB NOT NULL DEFAULT '{}'::JSONB;`,
		Down:    `ALTER TABLE user_search_indices DROP COLUMN last_regional_bootstrapped_value_ids;`,
	},
}
