package authz

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// MigrationRequest is the request body for the migration methods
type MigrationRequest struct {
	OrganizationID uuid.UUID `json:"organization_id" yaml:"organization_id"`
}

// AddOrganizationToObject adds the specified organization id to the user
func (c *Client) AddOrganizationToObject(ctx context.Context, objectID uuid.UUID, organizationID uuid.UUID) (*Object, error) {
	cm := cache.NewManager(c.cp, c.np, c.ttlP)
	req := &MigrationRequest{
		OrganizationID: organizationID,
	}
	obj := Object{BaseModel: ucdb.NewBaseWithID(objectID)}
	s, err := cache.TakeItemLock(ctx, cache.Update, cm, obj)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock[Object](ctx, cm, cache.Update, obj, s)

	var resp Object
	if err := c.client.Put(ctx, fmt.Sprintf("/authz/migrate/objects/%s", objectID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemToCache(ctx, cm, resp, s, true, nil)
	return &resp, nil
}

// AddOrganizationToEdgeType adds the specified organization id to the edge type
func (c *Client) AddOrganizationToEdgeType(ctx context.Context, edgeTypeID uuid.UUID, organizationID uuid.UUID) (*EdgeType, error) {
	cm := cache.NewManager(c.cp, c.np, c.ttlP)

	req := &MigrationRequest{
		OrganizationID: organizationID,
	}

	eT := EdgeType{BaseModel: ucdb.NewBaseWithID(edgeTypeID)}
	s, err := cache.TakeItemLock(ctx, cache.Update, cm, eT)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock[EdgeType](ctx, cm, cache.Update, eT, s)

	var resp EdgeType
	if err := c.client.Put(ctx, fmt.Sprintf("/authz/migrate/edgetypes/%s", edgeTypeID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemToCache(ctx, cm, resp, s, true, nil)
	return &resp, nil
}
