package companyconfig

import (
	"fmt"
	"strings"

	"userclouds.com/infra/migrate"
)

//go:generate genschemas

// Migrations specifies all SQL commands necessary to bring a blank DB
// to Company Config's most current schema.
var Migrations = migrate.Migrations{
	{
		Version: 0,
		Table:   "organizations",
		Desc:    "create initial organizations table",
		// NB: explicitly not using IF NOT EXISTS here to ensure we fail loudly on table duplication
		Up: `CREATE TABLE organizations (
			id UUID PRIMARY KEY,
			name VARCHAR UNIQUE
			);`,
		Down: `DROP TABLE organizations;`,
	},
	{
		Version: 1,
		Table:   "tenants",
		Desc:    "create initial tenants table",
		// NB: explicitly not using IF NOT EXISTS here to ensure we fail loudly on table duplication
		Up: `CREATE TABLE tenants (
			id UUID PRIMARY KEY,
			name VARCHAR UNIQUE,
			organization_id UUID,
			config JSONB
			);`,
		Down: `DROP TABLE tenants;`,
	},
	{
		// TODO: coalesce these migrations up front to single creates, but this saves people a DB reset for now
		// NOTE: primary key is always implicitly NOT NULL, see https://www.postgresql.org/docs/current/sql-createtable.html
		Version: 2,
		Table:   "organizations",
		Desc:    "add not null constraints to table",
		Up: `ALTER TABLE organizations
			ALTER COLUMN name SET NOT NULL;`,
		Down: `ALTER TABLE organizations
			ALTER COLUMN name DROP NOT NULL;`,
	},
	{
		Version: 3,
		Table:   "tenants",
		Desc:    "add not null constraints to table",
		Up: `ALTER TABLE tenants
			ALTER COLUMN name SET NOT NULL,
			ALTER COLUMN organization_id SET NOT NULL,
			ALTER COLUMN config SET NOT NULL;`,
		Down: `ALTER TABLE tenants
			ALTER COLUMN name DROP NOT NULL,
			ALTER COLUMN organization_id DROP NOT NULL,
			ALTER COLUMN config DROP NOT NULL;`,
	},
	{
		Version: 4,
		Table:   "organizations",
		Desc:    "transition orgs to basemodel",
		// TODO: when we coalesce and recreate, drop default for updated. It's needed when data exists in table,
		//   but it's confusing to have going forward because all our queries need to explicitly update updated
		//   so the default doesn't actually matter.
		Up: `ALTER TABLE organizations
			ADD COLUMN created TIMESTAMP NOT NULL DEFAULT NOW(),
			ADD COLUMN updated TIMESTAMP NOT NULL DEFAULT NOW();`,
		Down: `ALTER TABLE organizations
			DROP COLUMN created,
			DROP COLUMN updated;`,
	},
	{
		Version: 5,
		Table:   "tenants",
		Desc:    "transition tenants to basemodel",
		// TODO: when we coalesce and recreate, drop default for updated. It's needed when data exists in table,
		//   but it's confusing to have going forward because all our queries need to explicitly update updated
		//   so the default doesn't actually matter.
		Up: `ALTER TABLE tenants
			ADD COLUMN created TIMESTAMP NOT NULL DEFAULT NOW(),
			ADD COLUMN updated TIMESTAMP NOT NULL DEFAULT NOW();`,
		Down: `ALTER TABLE tenants
			DROP COLUMN created,
			DROP COLUMN updated;`,
	},
	{
		// You can't add a UNIQUE & NOT NULL column at the same time as it breaks existing data
		// Instead we'll autogen a placeholder domain, but you can't do that in a default.
		// Note that UNIQUE special-cases NULLs so we can set that up front.
		Version: 6,
		Table:   "tenants",
		Desc:    "add domain to tenants",
		Up: `ALTER TABLE tenants
				ADD COLUMN domain VARCHAR UNIQUE;
			UPDATE tenants SET domain=CONCAT(id::STRING, '.local') WHERE id=id;
			ALTER TABLE tenants
				ALTER COLUMN domain SET NOT NULL;`,
		Down: `ALTER TABLE tenants
			DROP COLUMN domain;`,
	},
	{
		Version: 7,
		Table:   "tenants_plex",
		Desc:    "split config to separate tables, plex first",
		Up: `CREATE TABLE tenants_plex (
				id UUID PRIMARY KEY,
				created TIMESTAMP NOT NULL DEFAULT NOW(),
				updated TIMESTAMP NOT NULL,
				plex_config JSONB NOT NULL
				);`,
		Down: `DROP TABLE tenants_plex;`,
	},
	{
		Version: 8,
		Table:   "tenants_idp",
		Desc:    "split config into separate tables, idp next",
		Up: `CREATE TABLE tenants_idp (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			idp_config JSONB NOT NULL
			);`,
		Down: `DROP TABLE tenants_idp;`,
	},
	{
		Version: 9,
		Table:   "tenants",
		Desc:    "finish splitting config out",
		Up: `ALTER TABLE tenants
			DROP COLUMN config;`,
		Down: `ALTER TABLE tenants
				ADD COLUMN config JSONB NOT NULL DEFAULT '{}';
			ALTER TABLE tenants
				ALTER COLUMN config DROP DEFAULT;`,
	},
	{
		Version: 10,
		Table:   "download_keys",
		Desc:    "create download_keys",
		Up: `CREATE TABLE download_keys (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			key VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOLEAN NOT NULL,
			tenant_id UUID NOT NULL
			);`,
		Down: `DROP TABLE download_keys;`,
	},
	// Note that migrations 11 & 12 could fail on down migration if you duped names while at 12+,
	// but the whole problem should go away when we coalesce these migrations to a "starting" schema
	{
		Version: 11,
		Table:   "organizations",
		Desc:    "drop unique name constraint, it's unreasonable for users",
		Up:      `DROP INDEX organizations_name_key CASCADE;`,
		Down:    `ALTER TABLE organizations ADD CONSTRAINT organizations_name_key UNIQUE (name);`,
	},
	{
		Version: 12,
		Table:   "tenants",
		Desc:    "drop unique tenant name constraint, it's also unreasonable especially across users",
		Up:      `DROP INDEX tenants_name_key CASCADE;`,
		Down:    `ALTER TABLE tenants ADD CONSTRAINT tenants_name_key UNIQUE (name);`,
	},
	{
		Version: 13,
		Table:   "download_keys",
		Desc:    "add primary key...oops",
		Up: `DELETE FROM download_keys WHERE id IN (SELECT id FROM (SELECT id, COUNT(*) AS c FROM download_keys GROUP BY id) WHERE c>1);
			ALTER TABLE download_keys ADD PRIMARY KEY (id);`,
		Down: `BEGIN;
			ALTER TABLE download_keys DROP CONSTRAINT "primary";
			ALTER TABLE download_keys ADD CONSTRAINT "primary" PRIMARY KEY (rowid);
			COMMIT;`,
	},
	{
		// Convert domain to tenant_url, then drop it.
		// Dev & prod have different policies for http scheme (http vs https),
		// so do our best to upconvert sensibly (without depending on config here).
		// Also this commit changes the tenant service port in dev from 3002 -> 3009,
		// and the ":3002" ensures we won't match any real prod tenants.
		// NOTE: this migration would not be safe in a zero-downtime deploy,
		// and requires code to be in sync with the migration.
		Version: 14,
		Table:   "tenants",
		Desc:    "change domain to tenantURL",
		Up: `ALTER TABLE tenants ADD COLUMN tenant_url VARCHAR UNIQUE;
			 UPDATE tenants SET tenant_url=CONCAT(
				(CASE WHEN domain LIKE '%.com' THEN 'https://'
				 ELSE 'http://' END), REPLACE(domain, ':3002', ':3009'));
			 ALTER TABLE tenants ALTER COLUMN tenant_url SET NOT NULL;
			 ALTER TABLE tenants DROP COLUMN domain;`,
		Down: `ALTER TABLE tenants ADD COLUMN domain VARCHAR UNIQUE;
			UPDATE tenants SET domain=REPLACE(SPLIT_PART(tenant_url, '://', 2), ':3009', ':3002');
			ALTER TABLE tenants ALTER COLUMN domain SET NOT NULL;
			ALTER TABLE tenants DROP COLUMN tenant_url;`,
	},
	{
		Version: 15,
		Table:   "organizations",
		Desc:    "add alive column to orgs table",
		Up:      `ALTER TABLE organizations ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE organizations DROP COLUMN alive;`,
	},
	{
		Version: 16,
		Table:   "tenants",
		Desc:    "add alive column to tenants table",
		Up:      `ALTER TABLE tenants ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE tenants DROP COLUMN alive;`,
	},
	{
		Version: 17,
		Table:   "tenants_idp",
		Desc:    "add alive column to tenants_idp table",
		Up:      `ALTER TABLE tenants_idp ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE tenants_idp DROP COLUMN alive;`,
	},
	{
		Version: 18,
		Table:   "tenants_plex",
		Desc:    "add alive column to tenants_plex table",
		Up:      `ALTER TABLE tenants_plex ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE tenants_plex DROP COLUMN alive;`,
	},
	{
		Version: 19,
		Table:   "download_keys",
		Desc:    "add alive column to download_keys table",
		Up:      `ALTER TABLE download_keys ADD COLUMN alive BOOL DEFAULT TRUE;`,
		Down:    `ALTER TABLE download_keys DROP COLUMN alive;`,
	},
	{
		// You can't add a NOT NULL column at the same time as it breaks existing data
		// Instead we'll create a placeholder log_config, and fix it in provisioning.
		Version: 20,
		Table:   "tenants_idp",
		Desc:    "add log config to tenants_idp",
		Up: `ALTER TABLE tenants_idp
				ADD COLUMN log_config JSONB;
			UPDATE tenants_idp SET log_config='{}';
			ALTER TABLE tenants_idp
				ALTER COLUMN log_config SET NOT NULL;`,
		Down: `ALTER TABLE tenants_idp
			DROP COLUMN log_config;`,
	},
	{
		Version: 21,
		Table:   "tenants_idp",
		Desc:    "rename tenants_idp to tenants_internal",
		Up:      `ALTER TABLE tenants_idp RENAME TO tenants_internal;`,
		Down:    `ALTER TABLE tenants_internal RENAME TO tenants_idp;`,
	},
	{
		Version: 22,
		Table:   "tenants",
		Desc:    "fix unique constraint to support soft-delete",
		Up: `DROP INDEX tenants_tenant_url_key CASCADE;
			ALTER TABLE tenants ADD CONSTRAINT tenants_tenant_url_alive_key UNIQUE (tenant_url, alive);`,
		Down: `DROP INDEX tenants_tenant_url_alive_key CASCADE;
			ALTER TABLE tenants ADD CONSTRAINT tenants_tenant_url_key UNIQUE (tenant_url)`,
	},
	{
		Version: 23,
		Table:   "download_keys",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE download_keys ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE download_keys SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE download_keys DROP COLUMN alive;
			ALTER TABLE download_keys ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX download_keys_id_key CASCADE;`,
		Down: `DELETE FROM download_keys WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM download_keys GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE download_keys ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE download_keys ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE download_keys SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE download_keys DROP COLUMN deleted;`,
	},
	{
		Version: 24,
		Table:   "organizations",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE organizations ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE organizations SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE organizations DROP COLUMN alive;
			ALTER TABLE organizations ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX organizations_id_key CASCADE;`,
		Down: `DELETE FROM organizations WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM organizations GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE organizations ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE organizations ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE organizations SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE organizations DROP COLUMN deleted;`,
	},
	{
		Version: 25,
		Table:   "tenants",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE tenants ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE tenants SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE tenants ADD CONSTRAINT tenants_tenant_url_deleted_key UNIQUE (tenant_url, deleted);
			ALTER TABLE tenants DROP COLUMN alive;
			ALTER TABLE tenants ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX tenants_id_key CASCADE;`,
		Down: `DELETE FROM tenants WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM tenants GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE tenants ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE tenants ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE tenants SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE tenants ADD CONSTRAINT tenants_tenant_url_alive_key UNIQUE (tenant_url, alive);
			ALTER TABLE tenants DROP COLUMN deleted;`,
	},
	{
		Version: 26,
		Table:   "tenants_internal",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE tenants_internal ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE tenants_internal SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE tenants_internal DROP COLUMN alive;
			ALTER TABLE tenants_internal ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX tenants_internal_id_key CASCADE;`,
		Down: `DELETE FROM tenants_internal WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM tenants_internal GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE tenants_internal ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE tenants_internal ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE tenants_internal SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE tenants_internal DROP COLUMN deleted;`,
	},
	{
		Version: 27,
		Table:   "tenants_plex",
		Desc:    "convert alive to deleted",
		Up: `ALTER TABLE tenants_plex ADD COLUMN deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE tenants_plex SET deleted=updated WHERE alive IS NULL;
			ALTER TABLE tenants_plex DROP COLUMN alive;
			ALTER TABLE tenants_plex ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX tenants_plex_id_key CASCADE;`,
		Down: `DELETE FROM tenants_plex WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP AND id IN (
				SELECT id FROM (
					SELECT id, COUNT(*) AS c FROM tenants_plex GROUP BY id
				) WHERE c>1
			);
			ALTER TABLE tenants_plex ALTER PRIMARY KEY USING COLUMNS (id);
			ALTER TABLE tenants_plex ADD COLUMN alive BOOLEAN DEFAULT TRUE;
			UPDATE tenants_plex SET alive=NULL WHERE deleted<>'0001-01-01 00:00:00'::TIMESTAMP;
			ALTER TABLE tenants_plex DROP COLUMN deleted;`,
	},
	{
		// NB: this defaults *all* tenants to hosted starting now (Console etc), which seems right
		// but is an explicit choice. We could alternatively go FALSE and write a separate query.
		Version: 28,
		Table:   "tenants_plex",
		Desc:    "add hosted bit to plex",
		Up:      `ALTER TABLE tenants_plex ADD COLUMN hosted BOOL NOT NULL DEFAULT TRUE;`,
		Down:    `ALTER TABLE tenants_plex DROP COLUMN hosted;`,
	},
	{
		// NB: this is a column rename that's ok for now but not soon :)
		Version: 29,
		Table:   "tenants_internal",
		Desc:    "rename IDPconfig to actually be tenants DB config",
		Up:      `ALTER TABLE tenants_internal RENAME COLUMN idp_config TO tenant_db_config;`,
		Down:    `ALTER TABLE tenants_internal RENAME COLUMN tenant_db_config TO idp_config;`,
	},
	{
		// NB: we leave the old data alone for now and just un-layer the plex_map
		// TODO: clean up eg. email etc?
		Version: 30,
		Table:   "tenants_plex",
		Desc:    "transition from old plex config to new separated tenant-only plex config",
		Up:      `UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['plex_map'], plex_config->'plex_map'->0);`,
		Down:    `UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['plex_map'], JSONB_BUILD_ARRAY(plex_config->'plex_map'));`,
	},
	{
		// clean up prod tenant URLs
		Version: 31,
		Table:   "tenants",
		Desc:    "get rid of idp.prod in tenant URLs",
		Up: `UPDATE tenants SET tenant_url=REPLACE(tenant_url, 'idp.prod.', '') WHERE tenant_url LIKE '%idp.prod%';
			UPDATE tenants_plex SET plex_config=REPLACE(plex_config::STRING, 'idp.prod.', '')::JSONB WHERE plex_config::STRING LIKE '%idp.prod%';
			UPDATE tenants_plex SET plex_config=REPLACE(plex_config::STRING, 'logserver.prod.', 'logserver.')::JSONB WHERE plex_config::STRING LIKE '%logserver.prod%';`,
		Down: `UPDATE tenants SET tenant_url=REPLACE(tenant_url, 'tenant.', 'tenant.idp.prod.') WHERE tenant_url LIKE '%tenant.userclouds%';
			UPDATE tenants_plex SET plex_config=REPLACE(plex_config::STRING, 'tenant.', 'tenant.idp.prod.')::JSONB WHERE plex_config::STRING LIKE '%tenant.userclouds%';
			UPDATE tenants_plex SET plex_config=REPLACE(plex_config::STRING, 'logserver.', 'logserver.prod.')::JSONB WHERE plex_config::STRING LIKE '%logserver.userclouds%';`,
	},
	{
		Version: 32,
		Table:   "tenants_plex",
		Desc:    "add tenant_url to tenants_plex for multitenant",
		Up: `UPDATE tenants_plex SET plex_config=JSONB_SET(
				plex_config, ARRAY['tenant_url'], CONCAT('"', t.tenant_url, '"')::JSONB
			) FROM tenants as t WHERE t.id = tenants_plex.id;`,
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['tenant_url']);`,
	},
	{
		Version: 33,
		Table:   "tenants_plex",
		Desc:    "add tenant_id to tenants_plex for logging",
		Up: `UPDATE tenants_plex SET plex_config=JSONB_SET(
				plex_config, ARRAY['tenant_id'], CONCAT('"', id::STRING, '"')::JSONB
			);`,
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['tenant_id']);`,
	},
	{
		// :face_palm: we'll never need more then 640k of memory!
		// we're explicitly *not* undoing this in the down migration because in the same
		// commit, we change the default, so this will actually keep tests correct
		Version: 34,
		Table:   "migrations",
		Desc:    "support longer migrations :/",
		Up: `ALTER TABLE migrations ADD COLUMN up2 VARCHAR(10000);
			UPDATE migrations SET up2=up;
			ALTER TABLE migrations ALTER COLUMN up2 SET NOT NULL;
			ALTER TABLE migrations DROP COLUMN up;
			ALTER TABLE migrations RENAME COLUMN up2 to up;`,
		Down: `/* noop */`,
	},
	{
		// NB: the format required here is PKCS#1 PEM for private key, and PKIX for public.
		// it's annoying to get these right eg. in tests because PKCS#1 and PKIX look similar in pubkeys
		// obviously these keys will get changed shortly to be different per-tenant,
		// and not in a migration that's checked in, but this will make it work like it
		// does today for now.
		// Also, note that we do the strange `strings.Replace` trick because JSON doesn't allow
		// newlines inside quoted strings, but cut&pasting keys that aren't formatting this way
		// is annoying and error prone, so we replace actual newlines with escaped newlines,
		// and then the sqlx layer (specifically Valuer) handles the double-escaping for us
		// We need to create the keys dictionary first (JSONB_SET should do this but isn't?)
		Version: 35,
		Table:   "tenants_plex",
		Desc:    "add JWT keys to plex tenant config",
		Up: fmt.Sprintf(`UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['keys'], '{}');
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['keys', 'private_key'], '%s'::JSONB);
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['keys', 'public_key'], '%s'::JSONB);`,
			strings.ReplaceAll(defaultPrivateKey, "\n", `\n`),
			strings.ReplaceAll(defaultPublicKey, "\n", `\n`)),
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['keys']);`,
	},
	{
		// TODO: can we merge this with download_keys?
		Version: 36,
		Table:   "invite_keys",
		Desc:    "one-time use codes & privileges associated with user invites",
		Up: `CREATE TABLE invite_keys (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			type INT8 NOT NULL,
			key VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOL NOT NULL,
			organization_id UUID,
			role VARCHAR,
			invitee_user_id UUID,
			INDEX (invitee_user_id),
			INDEX (key)
		);`,
		Down: `DROP TABLE invite_keys;`,
	},
	{
		Version: 37,
		Table:   "tenants_plex",
		Desc:    "add per-tenant key ID",
		// I'm not sure why SUBSTRING(x, 0, 9) is required to get an 8-char ID, but experimentally it is
		Up:   `UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, ARRAY['keys', 'key_id'], CONCAT('"', SUBSTRING(GEN_RANDOM_UUID()::STRING, 0, 9),'"')::JSONB);`,
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['keys', 'key_id']);`,
	},
	{
		Version: 38,
		Table:   "tenants_user_store",
		Desc:    "add table for per-tenant user store settings",
		Up: `CREATE TABLE tenants_user_store (
			id UUID PRIMARY KEY,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			schema JSONB NOT NULL
		);`,
		Down: `DROP TABLE tenants_user_store;`,
	},
	{
		Version: 39,
		Table:   "tenants_user_store",
		Desc:    "add blank entry for every tenant which doesn't have one yet",
		// Too hard to fix the linter for this :)
		Up: `/* lint-sql-ok */ INSERT INTO tenants_user_store (id, updated, schema)
			SELECT id, NOW(), '{}'::JSONB FROM tenants
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			ON CONFLICT DO NOTHING;`,
		Down: `/* noop */`,
	},
	{
		Version: 40,
		Table:   "tenants_internal",
		Desc:    "convert all keys to dev secret resolver (unless they're already aws)",
		Up: `UPDATE tenants_plex SET plex_config=JSONB_SET(
				plex_config,
				ARRAY['keys', 'private_key'],
				TO_JSONB(
					CONCAT(
						'dev://',
						ENCODE((plex_config->'keys'->>'private_key')::BYTES, 'base64')
					)
				)
			) WHERE plex_config->'keys'->>'private_key' NOT LIKE 'aws://%';`,
		Down: `UPDATE tenants_plex SET plex_config=JSONB_SET(
				plex_config,
				ARRAY['keys', 'private_key'],
				TO_JSONB(
					CONVERT_FROM(
						DECODE(
							SUBSTRING(plex_config->'keys'->>'private_key', 7),
							'base64'
						),
						'utf8'
					)
				)
			) WHERE plex_config->'keys'->>'private_key' LIKE 'dev://%';`,
	},
	{
		Version: 41,
		Table:   "tenants_user_store",
		Desc:    "fix PK on tenants_user_store",
		Up: `ALTER TABLE tenants_user_store ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX tenants_user_store_id_key CASCADE;`,
		Down: `ALTER TABLE tenants_user_store ALTER PRIMARY KEY USING COLUMNS (id);
			DROP INDEX tenants_user_store_id_deleted_key CASCADE;`,
	},
	{
		Version: 42,
		Table:   "invite_keys",
		Desc:    "fix invite_keys PK",
		Up: `ALTER TABLE invite_keys ALTER PRIMARY KEY USING COLUMNS (id, deleted);
			DROP INDEX invite_keys_id_key CASCADE;`,
		Down: `ALTER TABLE invite_keys ALTER PRIMARY KEY USING COLUMNS (id);
			DROP INDEX invite_keys_id_deleted_key CASCADE;`,
	},
	{
		// TODO: orgconfig really shouldn't hold console-specific data, but we don't yet
		// have a separate console DB and I don't have time tonight (live site issue with union.ai)
		Version: 43,
		Table:   "sessions",
		Desc:    "add a sessions table to orgconfig until console has its own DB",
		Up: `CREATE TABLE sessions (
				id UUID,
				created TIMESTAMP NOT NULL DEFAULT NOW(),
				updated TIMESTAMP NOT NULL,
				deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
				id_token VARCHAR,
				state VARCHAR,
				PRIMARY KEY (id, deleted)
			);`,
		Down: `DROP TABLE sessions;`,
	},
	{
		// NB: this defaults *all* tenants to hosted if we downgrade, but as of this change
		// there are no self-hosted tenants in staging or prod.
		Version: 44,
		Table:   "tenants_plex",
		Desc:    "remove hosted bit from plex",
		Up:      `ALTER TABLE tenants_plex DROP COLUMN hosted;`,
		Down:    `ALTER TABLE tenants_plex ADD COLUMN hosted BOOL NOT NULL DEFAULT TRUE;`,
	},
	{
		Version: 45,
		Table:   "download_keys",
		Desc:    "drop download_keys",
		Up:      `DROP TABLE download_keys;`,
		Down: `CREATE TABLE download_keys (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			key VARCHAR NOT NULL,
			expires TIMESTAMP NOT NULL,
			used BOOLEAN NOT NULL,
			tenant_id UUID NOT NULL,
			rowid INT8 NOT VISIBLE NOT NULL DEFAULT unique_rowid(),
			PRIMARY KEY (id, deleted)
			);`,
	},
	{
		Version: 46,
		Table:   "tenants_urls",
		Desc:    "add tenants_urls",
		Up: `CREATE TABLE tenants_urls (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT NOW(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			tenant_id UUID NOT NULL,
			tenant_url VARCHAR NOT NULL,
			PRIMARY KEY (id, deleted),
			UNIQUE (tenant_url, deleted)
			);`,
		Down: `DROP TABLE tenants_urls;`,
	},
	{
		// Note: if this migration is run after these URLs already exist (eg at this point in time
		// if you run provisioning and then migration), you will get duplicate tenant URLs. This isn't
		// a big deal, it's just not cool, but the effort to fix it seems not worth it. We can always
		// clean up dupes at a later date (even with another migration...)
		Version: 47,
		Table:   "tenants_urls",
		Desc:    "add default per-region tenant URLs for each tenant",
		Up: `INSERT INTO tenants_urls (id, updated, tenant_id, tenant_url)
			(SELECT GEN_RANDOM_UUID(), NOW(), id, REPLACE(tenant_url, '.tenant.', '.tenant-aws-us-east-1.') as tenant_url FROM tenants WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP AND tenant_url LIKE '%.tenant.%');
			INSERT INTO tenants_urls (id, updated, tenant_id, tenant_url)
			(SELECT GEN_RANDOM_UUID(), NOW(), id, REPLACE(tenant_url, '.tenant.', '.tenant-aws-us-west-2.') as tenant_url FROM tenants WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP AND tenant_url LIKE '%.tenant.%');
			/* lint-sql-ok because weird syntax, lint-system-table because we use two table names */`,
		// NB: this down clause is hard to scope perfectly...
		Down: `DELETE FROM tenants_urls WHERE tenant_url LIKE '%.tenant-aws-us-%';`,
	},
	{
		// NOTE: This migration had a bug in it - the WHERE clause in the inner SELECT matched
		//       all tenants, so google was added for all social providers. If you want to adapt
		//       this style of query for use elsewhere, swap out:
		//
		//       AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
		//
		//       for:
		//
		//       AND plex_config->'social_providers'->'google'->>'client_id'='' IS FALSE
		Version: 48,
		Table:   "tenants_plex",
		Desc:    "add google as an authentication method if it is a configured social provider",
		Up: `UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 4, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless,google"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 5);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 3, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless,google"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 4);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 2, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless,google"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 3);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 1, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless,google"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 2);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 0, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless,google"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->'client_id' IS NOT NULL
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 1);
				`,
		Down: `/* noop */`,
	},
	{
		Version: 49,
		Table:   "tenants_plex",
		Desc:    "remove erroneously added google authentication method if it is not a configured social provider",
		Up: `UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 4, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->>'client_id'='' IS NOT FALSE
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 5);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 3, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->>'client_id'='' IS NOT FALSE
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 4);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 2, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->>'client_id'='' IS NOT FALSE
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 3);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 1, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->>'client_id'='' IS NOT FALSE
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 2);
		     UPDATE tenants_plex
		     SET plex_config=JSONB_SET(plex_config,
					       '{plex_map, apps, 0, page_parameters}',
				   	       '{"every_page": {
						 	"authenticationMethods": {
								"parameter_name": "authenticationMethods",
								"parameter_type": "authentication_methods",
								"parameter_value": "password,passwordless"}
							}
						}'::JSONB)
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
		     AND id IN (SELECT id
			  	FROM tenants_plex
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
				AND plex_config->'social_providers'->'google'->>'client_id'='' IS NOT FALSE
				AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 1);
				`,
		Down: `/* noop */`,
	},
	{
		Version: 50,
		Table:   "tenants_plex",
		Desc:    "move google social provider configuration from social_providers->google to social_providers->google_provider",
		Up: `UPDATE tenants_plex
                     SET plex_config=JSONB_SET(plex_config,
                                               '{social_providers, google_provider}',
                                               plex_config->'social_providers'->'google')
                     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
                     AND plex_config->'social_providers'->'google'->>'client_id' <> '';
		     UPDATE tenants_plex
		     SET plex_config=JSON_REMOVE_PATH(plex_config, '{social_providers, google}')
		     WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;
                     `,
		Down: `UPDATE tenants_plex
                       SET plex_config=JSONB_SET(plex_config,
                                                 '{social_providers, google}',
                                                 '{"client_id": "", "client_secret": ""}'::JSONB)
                       WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;
                       UPDATE tenants_plex
                       SET plex_config=JSONB_SET(plex_config,
                                                 '{social_providers, google}',
                                                 plex_config->'social_providers'->'google_provider')
                       WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
                       AND plex_config->'social_providers'->'google_provider'->>'client_id' <> '';
                       UPDATE tenants_plex
                       SET plex_config=JSON_REMOVE_PATH(plex_config, '{social_providers, google_provider}')
		       WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
	},
	{
		Version: 51,
		Table:   "tenants_user_store",
		Desc:    "add accessors column for userstore",
		Up:      `ALTER TABLE tenants_user_store ADD COLUMN accessors JSONB NOT NULL DEFAULT '[]';`,
		Down:    `ALTER TABLE tenants_user_store DROP COLUMN accessors;`,
	},
	{
		// the only thing in schema currenty is fields, so we just start with an empty set and move fields->cols
		// rather than the normal add + delete JSONB keys. We don't technically need this on deleted rows but no harm.
		Version: 52,
		Table:   "tenants_user_store",
		Desc:    "rename fields to columns",
		Up:      `UPDATE tenants_user_store SET schema=JSONB_SET('{}', '{columns}', schema->'fields'); /* lint-deleted-ok */`,
		Down:    `UPDATE tenants_user_store SET schema=JSONB_SET('{}', '{fields}', schema->'columns'); /* lint-deleted-ok */`,
	},
	{
		Version: 53,
		Table:   "tenants_user_store",
		Desc:    "add version column to userstore schema",
		Up:      `ALTER TABLE tenants_user_store ADD COLUMN _version INT NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE tenants_user_store DROP COLUMN _version;`,
	},
	{
		Version: 54,
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
			attributes JSONB NOT NULL DEFAULT '{}',
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
		Version: 55,
		Table:   "sessions",
		Desc:    "add access_token and refresh_token to session store",
		Up: `ALTER TABLE sessions ADD COLUMN access_token VARCHAR NOT NULL DEFAULT '';
	         ALTER TABLE sessions ADD COLUMN refresh_token VARCHAR NOT NULL DEFAULT '';
			 ALTER TABLE sessions ALTER COLUMN id_token SET DEFAULT '';
			 ALTER TABLE sessions ALTER COLUMN id_token SET NOT NULL;
			 ALTER TABLE sessions ALTER COLUMN state SET DEFAULT '';
			 ALTER TABLE sessions ALTER COLUMN state SET NOT NULL;`,
		Down: `ALTER TABLE sessions DROP COLUMN refresh_token;
			 ALTER TABLE sessions DROP COLUMN access_token;
			 ALTER TABLE sessions ALTER COLUMN id_token DROP DEFAULT;
			 ALTER TABLE sessions ALTER COLUMN state DROP DEFAULT;
			 ALTER TABLE sessions ALTER COLUMN id_token DROP NOT NULL;
			 ALTER TABLE sessions ALTER COLUMN state DROP NOT NULL;`,
	},
	{
		Version: 56,
		Table:   "invite_keys",
		Desc:    "add invitee_email to invite_keys table",
		Up:      `ALTER TABLE invite_keys ADD COLUMN invitee_email VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE invite_keys DROP COLUMN invitee_email;`,
	},
	{
		// renames are bad but we're still early I guess? this will definitely be a breaking change
		Version: 57,
		Table:   "organizations",
		Desc:    "rename orgs -> companies",
		Up:      `ALTER TABLE organizations RENAME TO companies;`,
		Down:    `ALTER TABLE companies RENAME TO organizations;`,
	},
	{
		Version: 58,
		Table:   "tenants",
		Desc:    "rename orgs -> companies",
		Up:      `ALTER TABLE tenants RENAME COLUMN organization_id TO company_id;`,
		Down:    `ALTER TABLE tenants RENAME COLUMN company_id TO organization_id;`,
	},
	{
		Version: 59,
		Table:   "invite_keys",
		Desc:    "rename orgs -> companies",
		Up:      `ALTER TABLE invite_keys RENAME COLUMN organization_id TO company_id;`,
		Down:    `ALTER TABLE invite_keys RENAME COLUMN company_id TO organization_id;`,
	},
	{
		Version: 60,
		Table:   "tenants",
		Desc:    "add back orgs :) via use_organizations flag",
		Up:      `ALTER TABLE tenants ADD COLUMN use_organizations BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down:    `ALTER TABLE tenants DROP COLUMN use_organizations;`,
	},
	{
		Version: 61,
		Table:   "tenants_plex",
		Desc:    "provision employee_provider and employee_app in plex_map",
		Up: `UPDATE tenants_plex
		          SET plex_config=JSONB_SET(plex_config,
			 	 '{plex_map, employee_provider}',
				  CONCAT('{ "id": "',
					  GEN_RANDOM_UUID()::STRING,
					  '", "name": "Employee IDP Provider", "type":"uc", "uc": { "apps": [ { "id": "',
					  GEN_RANDOM_UUID()::STRING,
					  '", "name": "Employee IDP App" } ], "idp_url": "',
					  (SELECT tenant_url
					  	FROM tenants
						WHERE tenant_url LIKE 'http%console%'
						AND deleted='0001-01-01 00:00:00'
						LIMIT 1),
					  '" } }')::JSONB)
			  WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;
		     UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
				'{plex_map, employee_app}',
				CONCAT('{ "client_id": "',
					GEN_RANDOM_UUID()::STRING,
					'", "client_secret": "',
					GEN_RANDOM_UUID()::STRING,
					'", "id": "',
					GEN_RANDOM_UUID()::STRING,
					'", "name": "Employee Plex App", "provider_app_ids": [ "',
					plex_config->'plex_map'->'employee_provider'->'uc'->'apps'->0->>'id',
					'" ] }')::JSONB)
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
		Down: `UPDATE tenants_plex
				SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['plex_map', 'employee_provider'])
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;
		       UPDATE tenants_plex
			SET plex_config=JSON_REMOVE_PATH(plex_config, ARRAY['plex_map', 'employee_app'])
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
	},
	{
		Version: 62,
		Table:   "tenants_user_store",
		Desc:    "remove accessors from tenants_user_store",
		Up:      `ALTER TABLE tenants_user_store DROP COLUMN accessors;`,
		Down:    `ALTER TABLE tenants_user_store ADD COLUMN accessors JSONB NOT NULL DEFAULT '[]';`,
	},
	{
		Version: 63,
		Table:   "tenants",
		Desc:    "fix up .tenant. in dev URLs",
		Up: `UPDATE tenants set tenant_url=REPLACE(tenant_url, '.127.0.0.1', '.tenant.127.0.0.1')
				WHERE tenant_url LIKE '%.127.0.0.1.nip.io:3009' AND
					tenant_url NOT LIKE '%.tenant.127%';
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{tenant_url}', TO_JSONB(tenant_url)) FROM (
				SELECT id, tenant_url FROM tenants WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			) AS tenants WHERE tenants_plex.id=tenants.id AND
				tenants.tenant_url LIKE '%127.0.0.1.nip.io:3009';
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{plex_map, providers}', providers) FROM (
				SELECT id, JSONB_AGG(provider) AS providers FROM (
					SELECT id, JSONB_SET(
							provider,
							'{uc, idp_url}',
							TO_JSONB(tenant_url)
						) AS provider
						FROM (
							SELECT tp.id, t.tenant_url, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
								FROM tenants_plex tp
								JOIN tenants t ON t.id = tp.id
								WHERE tp.deleted='0001-01-01 00:00:00'::TIMESTAMP
						)
					) GROUP BY id
				) AS q WHERE tenants_plex.id=q.id AND
					tenants_plex.plex_config->>'tenant_url' LIKE '%127.0.0.1.nip.io:3009';`,
		Down: `/* no-op */`,
	},
	{
		Version: 64,
		Table:   "tenants_user_store",
		Desc:    "drop tenants_user_store table now that we don't use it",
		Up:      `DROP TABLE tenants_user_store;`,
		Down: `CREATE TABLE public.tenants_user_store (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now():::TIMESTAMP,
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00':::TIMESTAMP,
			schema JSONB NOT NULL,
			_version INT8 NOT NULL DEFAULT 0:::INT8,
			CONSTRAINT "primary" PRIMARY KEY (id ASC, deleted ASC),
			FAMILY "primary" (id, created, updated, deleted, schema, _version)
		)`,
	},
	{
		Version: 65,
		Table:   "tenants_plex",
		Desc:    "by default, all apps support all grant types for now",
		Up: `UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{plex_map, apps}', apps) FROM (
				SELECT id, JSONB_AGG(app) AS apps FROM (
					SELECT id, JSONB_SET(
							app,
							'{grant_types}',
							'["authorization_code", "refresh_token", "client_credentials"]'::JSONB
						) AS app
						FROM (
							SELECT tp.id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'apps') as app
								FROM tenants_plex tp
								WHERE tp.deleted='0001-01-01 00:00:00'::TIMESTAMP
						)
					GROUP BY id, app
				) GROUP BY id
			) AS q WHERE tenants_plex.id=q.id;`,
		Down: `/* no-op */`,
	},
	{
		Version: 66,
		Table:   "tenants_plex",
		Desc:    "add version safety to tenants_plex",
		Up:      `ALTER TABLE tenants_plex ADD COLUMN _version INT NOT NULL DEFAULT 0;`,
		Down:    `ALTER TABLE tenants_plex DROP COLUMN _version;`,
	},
	{
		Version: 67,
		Table:   "tenants_plex",
		Desc:    "set employee_provider type to employee",
		Up: `UPDATE tenants_plex
			       SET plex_config=JSONB_SET(plex_config,
			       '{plex_map, employee_provider, type}',
			       TO_JSONB('employee'))
			      WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			      AND plex_config->'plex_map'->'employee_provider' IS NOT NULL;`,
		Down: `UPDATE tenants_plex
                               SET plex_config=JSONB_SET(plex_config,
                               '{plex_map, employee_provider, type}',
                               TO_JSONB('uc'))
                              WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
                              AND plex_config->'plex_map'->'employee_provider' IS NOT NULL;`,
	},
	{
		Version: 68,
		Table:   "tenants_plex",
		Desc:    "remove employee_provider idp_url",
		Up: `UPDATE tenants_plex
                               SET plex_config=JSON_REMOVE_PATH(plex_config,
			       '{plex_map, employee_provider, uc, idp_url}')
                               WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
                              AND plex_config->'plex_map'->'employee_provider' IS NOT NULL;`,
		Down: `UPDATE tenants_plex
                               SET plex_config=JSONB_SET(plex_config,
                               '{plex_map, employee_provider, uc, idp_url}',
                               TO_JSONB((SELECT tenant_url
                                                FROM tenants
                                                WHERE tenant_url LIKE 'http%console%'
                                                AND deleted='0001-01-01 00:00:00'
                                                LIMIT 1)))
                              WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
                              AND plex_config->'plex_map'->'employee_provider' IS NOT NULL;`,
	},
	{
		Version: 69,
		Table:   "tenants_plex",
		Desc:    "by default, all employee apps support all grant types for now",
		Up: `UPDATE tenants_plex
                              SET plex_config=JSONB_SET(plex_config,
			      '{plex_map, employee_app, grant_types}',
			      '["authorization_code", "refresh_token", "client_credentials"]'::JSONB)
			      WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			      AND plex_config->'plex_map'->'employee_provider' IS NOT NULL;`,
		Down: `/* no-op */`,
	},
	{
		Version: 70,
		Table:   "tenants_plex",
		Desc:    "add default token expiration to plex config",
		Up: `UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{plex_map, apps}', app) FROM (
					SELECT id, JSONB_AGG(app) as app FROM (
						SELECT id, JSONB_SET(app, '{token_validity}', '{"access": 86400, "refresh": 2592000, "impersonate_user": 3600}'::JSONB) AS app FROM (
							SELECT id, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'apps') AS app FROM tenants_plex WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
						)
					) GROUP BY id) as t where tenants_plex.id = t.id;
				UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{plex_map, employee_app}', employee_app) FROM (
					SELECT id, JSONB_SET(employee_app, '{token_validity}', '{"access": 86400, "refresh": 2592000, "impersonate_user": 3600}'::JSONB) AS employee_app FROM (
						SELECT id, JSONB_EXTRACT_PATH(plex_config->'plex_map'->'employee_app') AS employee_app FROM tenants_plex WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
					)
				) as t WHERE tenants_plex.id = t.id AND t.employee_app IS NOT NULL;`,
		// NB: this is an imperfect down query since I didn't bother to write it for all lengths, but it's good enough for now
		// removing a path that doesn't exist is simply a no-op, and no one has >5 apps yet :)
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 0, token_validity}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 1, token_validity}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 2, token_validity}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 3, token_validity}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 4, token_validity}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, employee_app, token_validity}');`,
	},
	{
		Version: 71,
		Table:   "sessions",
		Desc:    "add impersonator tokens to sessions",
		Up: `ALTER TABLE sessions ADD COLUMN impersonator_id_token VARCHAR NOT NULL DEFAULT '';
			ALTER TABLE sessions ADD COLUMN impersonator_access_token VARCHAR NOT NULL DEFAULT '';
			ALTER TABLE sessions ADD COLUMN impersonator_refresh_token VARCHAR NOT NULL DEFAULT '';`,
		Down: `ALTER TABLE sessions DROP COLUMN impersonator_id_token;
			ALTER TABLE sessions DROP COLUMN impersonator_access_token;
			ALTER TABLE sessions DROP COLUMN impersonator_refresh_token;`,
	},
	{
		Version: 72,
		Table:   "tenants_urls",
		Desc:    "add validated column",
		Up: `ALTER TABLE tenants_urls ADD COLUMN validated BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN system BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN active BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN dns_verifier VARCHAR NOT NULL DEFAULT '';`,
		Down: `ALTER TABLE tenants_urls DROP COLUMN validated, DROP COLUMN system, DROP COLUMN active, DROP COLUMN dns_verifier;`,
	},
	{
		// default to false, but mark existing ones as validated since they're all userclouds.com
		// for some reason you can't ALTER COLUMN and UPDATE the same column in a single statement
		Version: 73,
		Table:   "tenants_urls",
		Desc:    "mark existing urls as validated",
		Up:      `UPDATE tenants_urls SET validated=TRUE, system=TRUE, active=TRUE WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
		Down:    `/* no op */`,
	},
	{
		Version: 74,
		Table:   "tenants_urls",
		Desc:    "add cert expiry to tenants_urls",
		Up:      `ALTER TABLE tenants_urls ADD COLUMN certificate_valid_until TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00'::TIMESTAMP;`,
		Down:    `ALTER TABLE tenants_urls DROP COLUMN certificate_valid_until;`,
	},
	{
		Version: 75,
		Table:   "invite_keys",
		Desc:    "add tenant_roles to invite_keys",
		Up:      `ALTER TABLE invite_keys ADD COLUMN tenant_roles JSONB NOT NULL DEFAULT '{}'::JSONB;`,
		Down:    `ALTER TABLE invite_keys DROP COLUMN tenant_roles;`,
	},
	{
		Version: 76,
		Table:   "tenants_plex",
		Desc:    "add native oidc providers to tenants_plex",
		Up: `UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config, '{oidc_providers}',
			'{"providers": [
			{
				"type": "facebook",
				"name": "facebook",
				"description": "Facebook",
				"issuer_url": "https://www.facebook.com",
				"client_id": "",
				"client_secret": "",
				"can_use_local_host_redirect": true,
				"use_local_host_redirect": false,
				"default_scopes": "openid public_profile email",
				"is_native": true
			},
			{
				"type": "google",
				"name": "google",
				"description": "Google",
				"issuer_url": "https://accounts.google.com",
				"client_id": "",
				"client_secret": "",
				"can_use_local_host_redirect": false,
				"use_local_host_redirect": false,
				"default_scopes": "openid profile email",
				"is_native": true
			},
			{
				"type": "linkedin",
				"name": "linkedin",
				"description": "LinkedIn",
				"issuer_url": "https://www.linkedin.com",
				"client_id": "",
				"client_secret": "",
				"can_use_local_host_redirect": false,
				"use_local_host_redirect": false,
				"default_scopes": "openid profile email",
				"is_native": true
			} ] }'::JSONB)
				WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 0, client_id}',
			plex_config->'social_providers'->'facebook_provider'->'client_id')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'facebook_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 0, client_secret}',
			plex_config->'social_providers'->'facebook_provider'->'client_secret')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'facebook_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 0, use_local_host_redirect}',
			plex_config->'social_providers'->'facebook_provider'->'use_local_host_redirect')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'facebook_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 1, client_id}',
			plex_config->'social_providers'->'google_provider'->'client_id')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'google_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 1, client_secret}',
			plex_config->'social_providers'->'google_provider'->'client_secret')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'google_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 2, client_id}',
			plex_config->'social_providers'->'linkedin_provider'->'client_id')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'linkedin_provider'->>'client_id' <> '';
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{oidc_providers, providers, 2, client_secret}',
			plex_config->'social_providers'->'linkedin_provider'->'client_secret')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND plex_config->'social_providers'->'linkedin_provider'->>'client_id' <> '';`,
		Down: `UPDATE tenants_plex
                        SET plex_config=JSON_REMOVE_PATH(plex_config, '{oidc_providers}')
                        WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
	},
	{
		Version: 77,
		Table:   "tenants_plex",
		Desc:    "remove social_providers from tenants_plex",
		Up: `UPDATE tenants_plex
			SET plex_config=JSON_REMOVE_PATH(plex_config, '{social_providers}')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
		Down: `UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config, '{social_providers}', '{}')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
	},
	{
		Version: 78,
		Table:   "tenants_plex",
		Desc:    "copy email_elements to message_elements in tenants_plex",
		Up: `UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
						'{plex_map, apps, 0, message_elements}',
						plex_config->'plex_map'->'apps'->0->'email_elements')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 1
			AND plex_config->'plex_map'->'apps'->0->'email_elements' IS NOT NULL;
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
						'{plex_map, apps, 1, message_elements}',
						plex_config->'plex_map'->'apps'->1->'email_elements')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 2
			AND plex_config->'plex_map'->'apps'->1->'email_elements' IS NOT NULL;
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
						'{plex_map, apps, 2, message_elements}',
						plex_config->'plex_map'->'apps'->2->'email_elements')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 3
			AND plex_config->'plex_map'->'apps'->2->'email_elements' IS NOT NULL;
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
						'{plex_map, apps, 3, message_elements}',
						plex_config->'plex_map'->'apps'->3->'email_elements')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 4
			AND plex_config->'plex_map'->'apps'->3->'email_elements' IS NOT NULL;
			UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
						'{plex_map, apps, 4, message_elements}',
						plex_config->'plex_map'->'apps'->4->'email_elements')
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			AND JSONB_ARRAY_LENGTH(plex_config->'plex_map'->'apps') >= 5
			AND plex_config->'plex_map'->'apps'->4->'email_elements' IS NOT NULL;`,
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 0, message_elements}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 1, message_elements}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 2, message_elements}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 3, message_elements}');
			UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, apps, 4, message_elements}');`,
	},
	{
		Version: 79,
		Table:   "tenants_plex",
		Desc:    "add telephony_provider to tenants_plex plex_config plex_map",
		Up: `UPDATE tenants_plex
			SET plex_config=JSONB_SET(plex_config,
			'{plex_map, telephony_provider}',
			'{"type": "none", "properties": {}}'::JSONB)
			WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP;`,
		Down: `UPDATE tenants_plex SET plex_config=JSON_REMOVE_PATH(plex_config, '{plex_map, telephony_provider}');`,
	},
	{
		Version: 80,
		Table:   "tenants",
		Desc:    "Append '-INTERNAL' to tenant names (name column in tenants table) if the tenant name is one charecter",
		Up:      `UPDATE tenants SET name=CONCAT(name, '-INTERNAL') WHERE LENGTH(name) < 2;`,
		Down:    `/* no op */`,
	},
	{
		Version: 81,
		Table:   "tenants_urls",
		Desc:    "soft-delete tenant_urls assosiated with deleted tenants",
		Up: `UPDATE tenants_urls SET deleted=NOW() WHERE id IN (
			SELECT tu.id FROM tenants_urls tu
				JOIN tenants t ON tu.tenant_id=t.id
				WHERE tu.deleted = '0001-01-01 00:00:00'::TIMESTAMP AND
					t.deleted <> '0001-01-01 00:00:00'::TIMESTAMP);`,
		Down: `/* no op */`,
	},
	{
		Version: 82,
		Table:   "tenants",
		Desc:    "add state to tenants",
		Up:      `ALTER TABLE tenants ADD COLUMN state VARCHAR NOT NULL DEFAULT 'creating';`,
		Down:    `ALTER TABLE tenants DROP COLUMN state;`,
	},
	{
		Version: 83,
		Table:   "tenants",
		Desc:    "set all existing tenants to active",
		Up:      `UPDATE tenants SET state='active';`,
		Down:    `/* no op */`,
	},
	{
		Version: 84,
		Table:   "tenants",
		Desc:    "replace 127.0.0.1.nip.io:3009 with dev.userclouds.tools:3333",
		Up: `UPDATE tenants set tenant_url=REPLACE(REPLACE(tenant_url, '127.0.0.1.nip.io:3009', 'dev.userclouds.tools:3333'), 'http', 'https')
				WHERE tenant_url LIKE '%127.0.0.1.nip.io:3009';
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{tenant_url}', TO_JSONB(tenant_url)) FROM (
				SELECT id, tenant_url FROM tenants WHERE deleted='0001-01-01 00:00:00'::TIMESTAMP
			) AS tenants WHERE tenants_plex.id=tenants.id AND
				tenants.tenant_url LIKE '%dev.userclouds.tools:3333';
			UPDATE tenants_plex SET plex_config=JSONB_SET(plex_config, '{plex_map, providers}', providers) FROM (
				SELECT id, JSONB_AGG(provider) AS providers FROM (
					SELECT id, JSONB_SET(
							provider,
							'{uc, idp_url}',
							TO_JSONB(tenant_url)
						) AS provider
						FROM (
							SELECT tp.id, t.tenant_url, JSONB_ARRAY_ELEMENTS(plex_config->'plex_map'->'providers') as provider
								FROM tenants_plex tp
								JOIN tenants t ON t.id = tp.id
								WHERE tp.deleted='0001-01-01 00:00:00'::TIMESTAMP
						)
					) GROUP BY id
				) AS q WHERE tenants_plex.id=q.id AND
					tenants_plex.plex_config->>'tenant_url' LIKE '%dev.userclouds.tools:3333';
			UPDATE tenants_urls SET tenant_url=REPLACE(REPLACE(tenant_url, '127.0.0.1.nip.io:3009', 'dev.userclouds.tools:3333'), 'http', 'https')
				WHERE tenant_url LIKE '%127.0.0.1.nip.io:3009';`,
		Down: `/* no-op */`,
	},
	{
		Version: 85,
		Table:   "tenants_internal",
		Desc:    "add cache config column",
		Up:      `ALTER TABLE tenants_internal ADD COLUMN cache_config JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE tenants_internal DROP COLUMN cache_config;`,
	},
	{
		Version: 86,
		Table:   "tenant_user_column_value_migrations",
		Desc:    "creating temporary table for managing user column value migrations",
		Up: `CREATE TABLE tenant_user_column_value_migrations (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			run_id VARCHAR NOT NULL,
			CONSTRAINT "tenant_user_column_value_migrations_pk" PRIMARY KEY (deleted, id)
		);`,
		Down: `DROP TABLE tenant_user_column_value_migrations;`,
	},
	{
		Version: 87,
		Table:   "tenant_user_column_value_migrations",
		Desc:    "drop temporary table for managing user column value migrations",
		Up:      `DROP TABLE tenant_user_column_value_migrations;`,
		Down: `CREATE TABLE tenant_user_column_value_migrations (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			run_id VARCHAR NOT NULL,
			CONSTRAINT "tenant_user_column_value_migrations_pk" PRIMARY KEY (deleted, id)
		);`,
	},
	{
		Version: 88,
		Table:   "tenant_user_column_cleanup",
		Desc:    "creating temporary table for managing user column cleanup",
		Up: `CREATE TABLE tenant_user_column_cleanup (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			run_id VARCHAR NOT NULL,
			CONSTRAINT "tenant_user_column_cleanup_pk" PRIMARY KEY (deleted, id)
		);`,
		Down: `DROP TABLE tenant_user_column_cleanup;`,
	}, {
		Version: 89,
		Table:   "companies",
		Desc:    "fix companies PK",
		Up: `ALTER TABLE companies DROP CONSTRAINT "primary";
			 ALTER TABLE companies ADD CONSTRAINT "companies_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE companies DROP CONSTRAINT "companies_pk";
			   ALTER TABLE companies ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 90,
		Table:   "invite_keys",
		Desc:    "fix invite_keys PK",
		Up: `ALTER TABLE invite_keys DROP CONSTRAINT "primary";
			 ALTER TABLE invite_keys ADD CONSTRAINT "invite_keys_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE invite_keys DROP CONSTRAINT "invite_keys_pk";
			   ALTER TABLE invite_keys ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 91,
		Table:   "tenants",
		Desc:    "fix tenants PK",
		Up: `ALTER TABLE tenants DROP CONSTRAINT "primary";
		     ALTER TABLE tenants ADD CONSTRAINT "tenants_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE tenants DROP CONSTRAINT "tenants_pk";
		       ALTER TABLE tenants ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 92,
		Table:   "tenants_internal",
		Desc:    "fix tenants_internal PK",
		Up: `ALTER TABLE tenants_internal DROP CONSTRAINT "primary";
		 	 ALTER TABLE tenants_internal ADD CONSTRAINT "tenants_internal_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE tenants_internal DROP CONSTRAINT "tenants_internal_pk";
		       ALTER TABLE tenants_internal ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 93,
		Table:   "tenants_plex",
		Desc:    "fix tenants_plex PK",
		Up: `ALTER TABLE tenants_plex DROP CONSTRAINT "primary";
			 ALTER TABLE tenants_plex ADD CONSTRAINT "tenants_plex_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE tenants_plex DROP CONSTRAINT "tenants_plex_pk";
		       ALTER TABLE tenants_plex ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 94,
		Table:   "event_metadata",
		Desc:    "Fix event_metadata PK from primary to event_metadata_pkey",
		Up: `ALTER TABLE event_metadata DROP CONSTRAINT IF EXISTS "primary";
			 ALTER TABLE event_metadata ADD CONSTRAINT IF NOT EXISTS "event_metadata_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE event_metadata DROP CONSTRAINT "event_metadata_pkey";
			   ALTER TABLE event_metadata ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 95,
		Table:   "event_metadata",
		Desc:    "Fix event_metadata PK from event_metadata_pkey to event_metadata_pk",
		Up: `ALTER TABLE event_metadata DROP CONSTRAINT "event_metadata_pkey";
			 ALTER TABLE event_metadata ADD CONSTRAINT "event_metadata_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE event_metadata DROP CONSTRAINT "event_metadata_pk";
			   ALTER TABLE event_metadata ADD CONSTRAINT "event_metadata_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 96,
		Table:   "sessions",
		Desc:    "Fix sessions PK from primary to sessions_pkey",
		Up: `ALTER TABLE sessions DROP CONSTRAINT IF EXISTS "primary";
			 ALTER TABLE sessions ADD CONSTRAINT IF NOT EXISTS "sessions_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE sessions DROP CONSTRAINT "sessions_pkey";
			   ALTER TABLE sessions ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 97,
		Table:   "sessions",
		Desc:    "Fix sessions PK from sessions_pkey to sessions_pk",
		Up: `ALTER TABLE sessions DROP CONSTRAINT "sessions_pkey";
		     ALTER TABLE sessions ADD CONSTRAINT "sessions_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE sessions DROP CONSTRAINT "sessions_pk";
			   ALTER TABLE sessions ADD CONSTRAINT "sessions_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 98,
		Table:   "tenants_urls",
		Desc:    "Fix tenants_urls PK from primary to tenants_urls_pkey",
		Up: `ALTER TABLE tenants_urls DROP CONSTRAINT IF EXISTS "primary";
			 ALTER TABLE tenants_urls ADD CONSTRAINT IF NOT EXISTS "tenants_urls_pkey" PRIMARY KEY (id, deleted);`,
		Down: `ALTER TABLE tenants_urls DROP CONSTRAINT "tenants_urls_pkey";
			   ALTER TABLE tenants_urls ADD CONSTRAINT "primary" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 99,
		Table:   "tenants_urls",
		Desc:    "Fix tenants_urls PK from tenants_urls_pkey to tenants_urls_pk",
		Up: `ALTER TABLE tenants_urls DROP CONSTRAINT "tenants_urls_pkey";
		     ALTER TABLE tenants_urls ADD CONSTRAINT "tenants_urls_pk" PRIMARY KEY (deleted, id);`,
		Down: `ALTER TABLE tenants_urls DROP CONSTRAINT "tenants_urls_pk";
			   ALTER TABLE tenants_urls ADD CONSTRAINT "tenants_urls_pkey" PRIMARY KEY (id, deleted);`,
	},
	{
		Version: 100,
		Table:   "tenants",
		Desc:    "fake migration",
		Up:      `/* no op */`,
		Down:    `/* no op */`,
	},
	{
		Version: 101,
		Table:   "tenants_internal",
		Desc:    "add connect_on_startup to tenants_internal",
		Up:      `ALTER TABLE tenants_internal ADD COLUMN connect_on_startup BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down:    `ALTER TABLE tenants_internal DROP COLUMN connect_on_startup;`,
	},
	{
		Version: 102,
		Table:   "tenants_plex",
		Desc:    "remove tenants_plex now that we use tenantdb to store this config",
		Up:      `DROP TABLE tenants_plex;`,
		Down: `CREATE TABLE tenants_plex (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			plex_config JSONB NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT TIMESTAMP '0001-01-01 00:00:00',
			_version BIGINT NOT NULL DEFAULT 0,
			CONSTRAINT tenants_plex_pk PRIMARY KEY (deleted, id)
		);`,
	},
	{
		Version: 103,
		Table:   "tenants",
		Desc:    "add primary_region to tenants",
		Up:      `ALTER TABLE tenants ADD COLUMN primary_region VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE tenants DROP COLUMN primary_region;`,
	},
	{
		Version: 104,
		Table:   "tenants",
		Desc:    "add sync_users bool to tenants",
		Up:      `ALTER TABLE tenants ADD COLUMN sync_users BOOLEAN NOT NULL DEFAULT FALSE;`,
		Down:    `ALTER TABLE tenants DROP COLUMN sync_users;`,
	},
	{
		Version: 105,
		Table:   "companies",
		Desc:    "add company type column to companies",
		Up:      `ALTER TABLE companies ADD COLUMN type VARCHAR NOT NULL DEFAULT 'internal';`,
		Down:    `ALTER TABLE companies DROP COLUMN type;`,
	},
	{
		Version: 106,
		Table:   "tenants",
		Desc:    "add psqlshim_config to tenants",
		Up:      `ALTER TABLE tenants ADD COLUMN psqlshim_config JSONB;`,
		Down:    `ALTER TABLE tenants DROP COLUMN psqlshim_config;`,
	},
	{
		Version: 107,
		Table:   "tenants",
		Desc:    "add sqlshim_config to tenants",
		Up:      `ALTER TABLE tenants ADD COLUMN sqlshim_config JSONB;`,
		Down:    `ALTER TABLE tenants DROP COLUMN sqlshim_config;`,
	},
	{
		Version: 108,
		Table:   "tenants",
		Desc:    "drop psqlshim_config from tenants",
		Up:      `ALTER TABLE tenants DROP COLUMN psqlshim_config;`,
		Down:    `ALTER TABLE tenants ADD COLUMN psqlshim_config JSONB;`,
	},
	{
		Version: 109,
		Table:   "sqlshim_proxies",
		Desc:    "add sqlshim_proxies table",
		Up: `CREATE TABLE sqlshim_proxies (
			id UUID NOT NULL,
			created TIMESTAMP NOT NULL DEFAULT now(),
			updated TIMESTAMP NOT NULL,
			deleted TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00',
			host VARCHAR NOT NULL,
			port INT NOT NULL,
			certificates JSONB NOT NULL,
			tenant_id UUID NOT NULL,
			database_id UUID NOT NULL,
			PRIMARY KEY (deleted, id),
			UNIQUE (deleted, port)
		);`,
		Down: `DROP TABLE sqlshim_proxies;`,
	},
	{
		Version: 110,
		Table:   "sqlshim_proxies",
		Desc:    "add public key to sqlshim_proxies",
		Up:      `ALTER TABLE sqlshim_proxies ADD COLUMN public_key VARCHAR NOT NULL DEFAULT '';`,
		Down:    `ALTER TABLE sqlshim_proxies DROP COLUMN public_key;`,
	},
	{
		Version: 111,
		Table:   "tenants_internal",
		Desc:    "add tenant_migration_replica_db_config to tenants_internal",
		Up:      `ALTER TABLE tenants_internal ADD COLUMN tenant_migration_replica_db_config JSONB NOT NULL DEFAULT '{}';`,
		Down:    `ALTER TABLE tenants_internal DROP COLUMN tenant_migration_replica_db_config;`,
	},
	{
		Version: 112,
		Table:   "tenants_internal",
		Desc:    "add remote_user_region_db_configs and primary_user_region to tenants_internal",
		Up: `ALTER TABLE tenants_internal ADD COLUMN remote_user_region_db_configs JSONB NOT NULL DEFAULT '{}';
			ALTER TABLE tenants_internal ADD COLUMN primary_user_region VARCHAR NOT NULL DEFAULT '';`,
		Down: `ALTER TABLE tenants_internal DROP COLUMN remote_user_region_db_configs;
			ALTER TABLE tenants_internal DROP COLUMN primary_user_region;`,
	},
}

// these should eventually be used nowhere (obviously, since they're checked in),
// but will get us from state (bad) to state (slightly less bad).
const defaultPrivateKey = `"-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----"`

const defaultPublicKey = `"-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA61hdA8fRvjRynndI/oXc
lnom45Ro04hYkShC4UFZnPh97yFtDQbfgtEfylG9AJdrq8a3FF1mjU0o4SQMpSn5
aX+44bUeWu1Qn+vEKKKr1TD6Iy9/3vYljEM/EQDv8NIhXmwSm9yTE40IZ2Qm3qjX
O4aJBgmZZqozyOrZt4/NGa7VzrxurLhKwoU1g21NK3Rnve8+QyrUQIkyF4BO+u5g
z/fDwBqnwPnq3PW7j5aFZ5u1SpyEQwKIg2EgIt75E7Q7RKtAC/w783BPS61l+gC5
zdRnv2g8urATfxQbn3xu4xjWXg8t9grQtLHY0sK1r0LpPN4gDFTVQg3CipWxYfGQ
IHVpolyEOmsMnOp7PJSvcA3rVTfH9EOugg64rnsbuppoqUq3KHsLQkNk3Kb9DREL
yP3yUKLoLSg1JeEo6zo1k8YFeCYXplToO2FZ6HOwVSQ6rjSQCKH2JMr9HK8f9Ir5
3sEc39WHIVoUOTqRLlYxdhkmUX3UdBgJvqkQ7i7T7HD7HUA0LS2NItwginZ3bFjW
VywGpKmIZLNZR86jc+7q2BuMNzg2ZRrhV3u98OjrvkPr5sqDcY6ruzlHb4bwByZJ
VumF0PF1M/bFzWZq1+YIZnjur/y2WBeHJ0uN31HVtXVOc1h+sP+5WiCU24v88W5P
02rDPQ85lr3JGAaQbhGXGacCAwEAAQ==
-----END PUBLIC KEY-----"`
