package tenantmap

import (
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
)

// AssertNoHostInMap asserts that the given host is not in the map.
func (tm *StateMap) AssertNoHostInMap(t *testing.T, host string) {
	_, ok := tm.tenantsByHost[host]
	assert.False(t, ok)
}

// AssertNoIDInMap asserts that the given tenant ID is not in the map.
func (tm *StateMap) AssertNoIDInMap(t *testing.T, tenantID uuid.UUID) {
	_, ok := tm.tenantsByID[tenantID]
	assert.False(t, ok)
}
