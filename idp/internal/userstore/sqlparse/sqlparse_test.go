package sqlparse_test

import (
	"strings"
	"testing"

	"userclouds.com/idp/internal/userstore/sqlparse"
	"userclouds.com/infra/assert"
)

func TestParseQuery(t *testing.T) {
	// Simple query
	query, err := sqlparse.ParseQuery("SELECT a, b, c FROM users WHERE a = 1 OR (b = a AND c = 2)")
	assert.IsNil(t, err)
	assert.Equal(t, query.Type, sqlparse.QueryTypeSelect)
	assert.Equal(t, query.Columns, []sqlparse.Column{{Table: "users", Name: "a"}, {Table: "users", Name: "b"}, {Table: "users", Name: "c"}})
	assert.Equal(t, query.Selector, "({a} = ? OR ({b} = {a} AND {c} = ?))")

	// Fails parsing
	_, err = sqlparse.ParseQuery("SELECT a, b, c FROM users WHERE a = 1 OR (b = a AND c = 2")
	assert.NotNil(t, err)

	// Query with comments
	query, err = sqlparse.ParseQuery("SELECT a, b, c FROM users /* comment 1 */ WHERE a = 1 /* comment 2 */")
	assert.IsNil(t, err)
	assert.Equal(t, query.Type, sqlparse.QueryTypeSelect)
	assert.Equal(t, query.Columns, []sqlparse.Column{{Table: "users", Name: "a"}, {Table: "users", Name: "b"}, {Table: "users", Name: "c"}})
	assert.Equal(t, query.Selector, "{a} = ?")

	// Query with join
	query, err = sqlparse.ParseQuery(`SELECT "users"."a", "users"."b", "users"."c", "other_table"."d" FROM "users" JOIN "other_table" ON "users"."id" = "other_table"."user_id" WHERE "users"."a" = 1 OR "d" = 2`)
	assert.IsNil(t, err)
	assert.Equal(t, query.Type, sqlparse.QueryTypeSelect)
	assert.Equal(t, query.Columns, []sqlparse.Column{{Table: "users", Name: "a"}, {Table: "users", Name: "b"}, {Table: "users", Name: "c"}, {Table: "other_table", Name: "d"}})
	assert.Equal(t, query.Selector, "({a} = ? OR {d} = ?)")

	// Query with star
	query, err = sqlparse.ParseQuery("SELECT * FROM users WHERE a = 1 OR (b = a AND c = 2)")
	assert.IsNil(t, err)
	assert.Equal(t, query.Columns, []sqlparse.Column{{Table: "users", Name: "*"}})
	assert.Equal(t, query.Selector, "({a} = ? OR ({b} = {a} AND {c} = ?))")

	// Query with star and join
	query, err = sqlparse.ParseQuery(`SELECT * FROM "users" JOIN "other_table" ON "users"."id" = "other_table"."user_id" WHERE "users"."a" = 1 OR "d" = 2`)
	assert.IsNil(t, err)
	assert.Equal(t, query.Type, sqlparse.QueryTypeSelect)
	assert.Equal(t, query.Columns, []sqlparse.Column{{Table: "users", Name: "*"}, {Table: "other_table", Name: "*"}})
	assert.Equal(t, query.Selector, "({a} = ? OR {d} = ?)")

	q := "SELECT `messages_identity`.`id` FROM messages_identity WHERE organization_id = 1589;"
	q = strings.ReplaceAll(strings.ReplaceAll(q, "\"", "'"), "`", "\"")
	_, err = sqlparse.ParseQuery(q)
	assert.IsNil(t, err)

	// Yext example query
	q = "SELECT `messages_identity`.`id`, `messages_identity`.`organization_id`, `messages_identity`.`live`, `messages_identity`.`identity_type`, `messages_identity`.`group_id`, `messages_identity`.`account_id`, `messages_identity`.`e164_number`, `messages_identity`.`first_name`, `messages_identity`.`last_name`, `messages_identity`.`attestation_state`, `messages_identity`.`attestation_updated`, `messages_identity`.`date_updated`, `messages_identity`.`reset_count`, `messages_identity`.`opt_in_guidance_count`, `messages_identity`.`sync_contact_disabled`, `messages_identity`.`account_e164_number` FROM `messages_identity` WHERE (`messages_identity`.`organization_id` = 1589 AND `messages_identity`.`live` = 1 AND `messages_identity`.`organization_id` = 1589 AND `messages_identity`.`account_id` = 111912 AND `messages_identity`.`e164_number` IN ('+14154200581') AND `messages_identity`.`organization_id` = 1589) ;"
	q = strings.ReplaceAll(strings.ReplaceAll(q, "\"", "'"), "`", "\"")
	_, err = sqlparse.ParseQuery(q)
	assert.IsNil(t, err)

	q = "SELECT foo FROM bar WHERE baz IS NULL;"
	_, err = sqlparse.ParseQuery(q)
	assert.IsNil(t, err)
}
