package uuid_test

import (
	"regexp"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	ucuuid "userclouds.com/infra/uctypes/uuid"
)

func badMatch(t *testing.T, s string) {
	t.Helper()

	p, err := regexp.Compile(ucuuid.UUIDPattern)
	assert.NoErr(t, err)
	assert.False(t, p.MatchString(s))
}

func goodMatch(t *testing.T, s string) {
	t.Helper()

	p, err := regexp.Compile(ucuuid.UUIDPattern)
	assert.NoErr(t, err)
	assert.True(t, p.MatchString(s))
}

func TestUUIDPattern(t *testing.T) {
	badMatch(t, "")
	badMatch(t, "foo")
	badMatch(t, "f47ac10b-58cc4372a5670e02b2c3d479")

	goodMatch(t, "f47aC10b58cc4372A5670e02b2c3d479")
	goodMatch(t, "f47aC10b-58cc-4372-A567-0e02b2c3d479")
	goodMatch(t, uuid.Must(uuid.NewV4()).String())
}
