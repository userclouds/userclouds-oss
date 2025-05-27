package authz

import (
	"math/rand"
	"time"

	"userclouds.com/infra/cache"
)

// CacheTTLProvider implements the cache.CacheTTLProvider interface
type CacheTTLProvider struct {
	objTypeTTL  time.Duration
	edgeTypeTTL time.Duration
	objTTL      time.Duration
	edgeTTL     time.Duration
	orgTTL      time.Duration
	exprWindow  time.Duration
}

// NewCacheTTLProvider creates a new Configurablecache.CacheTTLProvider
func NewCacheTTLProvider(objTypeTTL time.Duration, edgeTypeTTL time.Duration, objTTL time.Duration, edgeTTL time.Duration, exprWindow time.Duration) *CacheTTLProvider {
	return &CacheTTLProvider{objTypeTTL: objTypeTTL, edgeTypeTTL: edgeTypeTTL, objTTL: objTTL, edgeTTL: edgeTTL, orgTTL: objTypeTTL, exprWindow: exprWindow}
}

const (
	// ObjectTypeTTL is the TTL for object types
	ObjectTypeTTL = "OBJ_TYPE_TTL"
	// EdgeTypeTTL is the TTL for edge types
	EdgeTypeTTL = "EDGE_TYPE_TTL"
	// ObjectTTL is the TTL for objects
	ObjectTTL = "OBJ_TTL"
	// EdgeTTL is the TTL for edges
	EdgeTTL = "EDGE_TTL"
	// OrganizationTTL is the TTL for organizations
	OrganizationTTL = "ORG_TTL"
)

// TTL returns the TTL for given type
func (c *CacheTTLProvider) TTL(id cache.KeyTTLID) time.Duration {
	var shiftTTL time.Duration

	if c.exprWindow != 0 {
		shiftTTL = time.Duration(rand.Intn(int(c.exprWindow.Nanoseconds())))
	}
	switch id {
	case ObjectTypeTTL:
		return c.objTypeTTL + shiftTTL
	case EdgeTypeTTL:
		return c.edgeTypeTTL + shiftTTL
	case ObjectTTL:
		return c.objTTL + shiftTTL
	case EdgeTTL:
		return c.edgeTTL + shiftTTL
	case OrganizationTTL:
		return c.orgTTL + shiftTTL
	}
	return cache.SkipCacheTTL
}
