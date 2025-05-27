package companyconfig_test

import (
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/companyconfig"
)

func mustGenerateSafeHostname(t *testing.T, tenantName string) string {
	hostname, err := companyconfig.GenerateSafeHostname(tenantName)
	assert.NoErr(t, err)
	return hostname
}

func mustNotGenerateSafeHostname(t *testing.T, tenantName string) string {
	hostname, err := companyconfig.GenerateSafeHostname(tenantName)
	assert.NotNil(t, err)
	return hostname
}

func TestHostnameGeneration(t *testing.T) {
	assert.Equal(t, mustGenerateSafeHostname(t, "a"), "a")
	assert.Equal(t, mustGenerateSafeHostname(t, "some tenant name"), "sometenantname")
	assert.Equal(t, mustGenerateSafeHostname(t, "-!@$SOME tenant.na---me.123--"), "sometenantna-me123")

	longString := "abcdefghijklmnopqrstuvwxyz-01234567890"
	assert.Equal(t, mustGenerateSafeHostname(t, longString), longString)
	veryLongString := longString + longString + longString
	assert.True(t, len(mustGenerateSafeHostname(t, veryLongString)) < len(veryLongString))

	mustNotGenerateSafeHostname(t, "")
	mustNotGenerateSafeHostname(t, "...")
	mustNotGenerateSafeHostname(t, "-")
	mustNotGenerateSafeHostname(t, "---")
}
