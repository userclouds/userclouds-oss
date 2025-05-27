package authz

import "userclouds.com/infra/ucerr"

// ErrObjectNotFound is returned if an object is not found.
var ErrObjectNotFound = ucerr.Friendlyf(nil, "object not found")

// ErrRelationshipTypeNotFound is returned if a relationship type name
// (e.g. "editor") is not found.
var ErrRelationshipTypeNotFound = ucerr.Friendlyf(nil, "relationship type not found")

// ErrEdgeNotFound is returned if an edge is not found.
var ErrEdgeNotFound = ucerr.Friendlyf(nil, "edge not found")

// ErrEdgeTypeNotFound is returned if an edge is not found.
var ErrEdgeTypeNotFound = ucerr.Friendlyf(nil, "edge type not found")

// ErrObjectTypeNotFound is returned if an object is not found.
var ErrObjectTypeNotFound = ucerr.Friendlyf(nil, "object type not found")
