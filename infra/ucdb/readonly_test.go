package ucdb_test

import (
	"strings"
	"testing"

	"userclouds.com/infra/assert"
)

func TestReadOnlyQuery(t *testing.T) {
	assert.True(t, isReadOnlyQuery("SELECT * FROM tmp;"))
	assert.True(t, isReadOnlyQuery("SELECT a, b, c FROM tmp;"))
	assert.True(t, isReadOnlyQuery("SELECT id, created, updated, deleted FROM tmp WHERE updated='2023-01-01 00:01:02"))

	assert.False(t, isReadOnlyQuery("DELETE FROM tmp;"))
	assert.False(t, isReadOnlyQuery("UPDATE tmp SET a=1;"))
	assert.False(t, isReadOnlyQuery("INSERT INTO tmp (a, b, c) VALUES (1, 2, 3);"))
	assert.False(t, isReadOnlyQuery("UPSERT INTO tmp (a, b, c) VALUES (1, 2, 3);"))
	assert.False(t, isReadOnlyQuery("INSERT INTO tmp (a, b, c) VALUES (1, 2, 3) ON CONFLICT UPDATE tmp set a=1, b=2 where c=3;"))
}

func isReadOnlyQuery(q string) bool {
	lcq := strings.ToLower(q)
	return strings.HasPrefix(lcq, "select")
}
