package authz

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// Delete deletes a group.
func (g Group) Delete(ctx context.Context) error {
	return ucerr.Wrap(g.gbacClient.client.DeleteObject(ctx, g.ID))
}

// AddGroupAttributeSet adds an attribute set between a group and another group
func (g Group) AddGroupAttributeSet(ctx context.Context, group Group, name string) (uuid.UUID, error) {
	return g.gbacClient.addAttributeSet(ctx, g.ID, group.ID, name)
}

// AddResourceAttributeSet adds an attribute set between a group and a resource
func (g Group) AddResourceAttributeSet(ctx context.Context, r Resource, name string) (uuid.UUID, error) {
	return g.gbacClient.addAttributeSet(ctx, g.ID, r.ID, name)
}

// DeleteGroupAttributeSet deletes an attribute set between a group and another group
func (g Group) DeleteGroupAttributeSet(ctx context.Context, group Group, name string) error {
	return ucerr.Wrap(g.gbacClient.deleteAttributeSet(ctx, g.ID, group.ID, name))
}

// DeleteResourceAttributeSet deletes an attribute set between a group and a resource
func (g Group) DeleteResourceAttributeSet(ctx context.Context, r Resource, as AttributeSet) error {
	return ucerr.Wrap(g.gbacClient.deleteAttributeSet(ctx, g.ID, r.ID, as.TypeName))
}

// DeleteAllGroupAttributeSets deletes all attribute sets between a group and another group
func (g Group) DeleteAllGroupAttributeSets(ctx context.Context, group Group) error {
	return ucerr.Wrap(g.gbacClient.deleteAllAttributeSets(ctx, g.ID, group.ID))
}

// DeleteAllResourceAttributeSets deletes all attribute sets between a group and a resource
func (g Group) DeleteAllResourceAttributeSets(ctx context.Context, r Resource) error {
	return ucerr.Wrap(g.gbacClient.deleteAllAttributeSets(ctx, g.ID, r.ID))
}

// CheckGroupAttribute checks if a group has a specific attribute on another group
func (g Group) CheckGroupAttribute(ctx context.Context, group Group, name string) (bool, error) {
	return g.gbacClient.checkAttribute(ctx, g.ID, group.ID, name)
}

// CheckResourceAttribute checks if a group has a specific attribute on a resource
func (g Group) CheckResourceAttribute(ctx context.Context, r Resource, name string) (bool, error) {
	return g.gbacClient.checkAttribute(ctx, g.ID, r.ID, name)
}

// ListGroupAttributes lists all attributes a group has on a group
func (g Group) ListGroupAttributes(ctx context.Context, group Group) ([]string, error) {
	return g.gbacClient.client.ListAttributes(ctx, g.ID, group.ID)
}

// ListResourceAttributes lists all attributes a group has on a resource
func (g Group) ListResourceAttributes(ctx context.Context, r Resource) ([]string, error) {
	return g.gbacClient.client.ListAttributes(ctx, g.ID, r.ID)
}
