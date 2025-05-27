package main

import (
	"context"
	"log"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// NB: most of the methods in this file should end up in the public SDK in some form, as they're generally useful
// for idempotent creation of objects, types, etc.

func provisionObjectType(ctx context.Context, authZClient *authz.Client, typeName string) (uuid.UUID, error) {
	ot, err := authZClient.CreateObjectType(ctx, uuid.Nil, typeName, authz.IfNotExists())
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return ot.ID, nil
}

func provisionEdgeType(ctx context.Context, authZClient *authz.Client, sourceObjectID, targetObjectID uuid.UUID, typeName string, attributes authz.Attributes) (uuid.UUID, error) {
	et, err := authZClient.CreateEdgeType(ctx, uuid.Nil, sourceObjectID, targetObjectID, typeName, attributes, authz.IfNotExists())
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return et.ID, nil
}

func provisionObject(ctx context.Context, authZClient *authz.Client, typeID uuid.UUID, alias string) (uuid.UUID, error) {
	obj, err := authZClient.CreateObject(ctx, uuid.Nil, typeID, alias, authz.IfNotExists())
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return obj.ID, nil
}

func provisionUser(ctx context.Context, idpClient *idp.Client, name string) (uuid.UUID, error) {
	// Create a new user
	profile := userstore.Record{}
	profile["name"] = name

	id, err := idpClient.CreateUser(ctx, profile)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return id, nil
}

// mustID panics if a UUID-producing operation returns an error, otherwise it returns the UUID
func mustID(id uuid.UUID, err error) uuid.UUID {
	if err != nil {
		log.Fatalf("mustID error: %v", err)
	}
	if id.IsNil() {
		log.Fatal("mustID error: unexpected nil uuid")
	}
	return id
}

// mustEdge panics if an edge-producing operation returns an error, otherwise it returns the Edge
func mustEdge(edge *authz.Edge, err error) *authz.Edge {
	if err != nil {
		log.Fatalf("mustEdge error: %v", err)
	}
	if edge.ID.IsNil() {
		log.Fatal("mustEdge error: unexpected nil uuid")
	}
	return edge
}
