package authz

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// AddGroupAttributeSet adds an attribute set between a user and a group
func (u User) AddGroupAttributeSet(ctx context.Context, g Group, name string) (uuid.UUID, error) {
	return u.gbacClient.addAttributeSet(ctx, u.ID, g.ID, name)
}

// AddResourceAttributeSet adds an attribute set between a user and a resource
func (u User) AddResourceAttributeSet(ctx context.Context, r Resource, name string) (uuid.UUID, error) {
	return u.gbacClient.addAttributeSet(ctx, u.ID, r.ID, name)
}

// DeleteGroupAttributeSet deletes an attribute set between a user and a group
func (u User) DeleteGroupAttributeSet(ctx context.Context, g Group, name string) error {
	return ucerr.Wrap(u.gbacClient.deleteAttributeSet(ctx, u.ID, g.ID, name))
}

// DeleteResourceAttributeSet deletes an attribute set between a user and a resource
func (u User) DeleteResourceAttributeSet(ctx context.Context, r Resource, as AttributeSet) error {
	return ucerr.Wrap(u.gbacClient.deleteAttributeSet(ctx, u.ID, r.ID, as.TypeName))
}

// DeleteAllGroupAttributeSets deletes all attribute sets between a user and a group
func (u User) DeleteAllGroupAttributeSets(ctx context.Context, g Group) error {
	return ucerr.Wrap(u.gbacClient.deleteAllAttributeSets(ctx, u.ID, g.ID))
}

// DeleteAllResourceAttributeSets deletes all attribute sets between a user and a resource
func (u User) DeleteAllResourceAttributeSets(ctx context.Context, r Resource) error {
	return ucerr.Wrap(u.gbacClient.deleteAllAttributeSets(ctx, u.ID, r.ID))
}

// CheckGroupAttribute checks if a user has a specific attribute on a group
func (u User) CheckGroupAttribute(ctx context.Context, g Group, name string) (bool, error) {
	return u.gbacClient.checkAttribute(ctx, u.ID, g.ID, name)
}

// CheckResourceAttribute checks if a user has a specific attribute on a resource
func (u User) CheckResourceAttribute(ctx context.Context, r Resource, name string) (bool, error) {
	return u.gbacClient.checkAttribute(ctx, u.ID, r.ID, name)
}

// ListGroupAttributes lists all attributes a user has on a group
func (u User) ListGroupAttributes(ctx context.Context, g Group) ([]string, error) {
	return u.gbacClient.client.ListAttributes(ctx, u.ID, g.ID)
}

// ListResourceAttributes lists all attributes a user has on a resource
func (u User) ListResourceAttributes(ctx context.Context, r Resource) ([]string, error) {
	return u.gbacClient.client.ListAttributes(ctx, u.ID, r.ID)
}
