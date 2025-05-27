package authz

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// In our graph-based access control, there are Users, Groups, and other Objects that require permissions to
// access and modify. Permissions are represented by Attributes of Edges connecting nodes in the graph.
// AttributeSets are EdgeTypes under the hood, and represent a fixed set of attributes.
//
// We assume for now that Users may be connected to Groups and Resources, Groups may be connected to other Groups
// and Resources, and Resources may be connected to other Resources. A couple of examples:
//
// U1 --(user to group)-> G1 --(group to group)-> G2 --(group to resource)-> R1 --(resource to resource)-> R2
// U --(user to resource)-> R

// GBACClient is a high-level GBAC client for the AuthZ service
type GBACClient struct {
	client *Client
}

//go:generate genvalidate GBACClient

// AttributeSet is an alias for EdgeType
type AttributeSet EdgeType

// User represents a user in the AuthZ GBAC and RBAC systems.
type User struct {
	ID         uuid.UUID  `json:"id" yaml:"id" validate:"notnil"`
	gbacClient GBACClient // this is a client convenience, not required for eg. saving
}

//go:generate genvalidate User

// Group represents a group in the AuthZ GBAC and RBAC systems.
type Group struct {
	ID         uuid.UUID  `json:"id" yaml:"id" validate:"notnil"`
	Name       string     `json:"name" yaml:"name" validate:"notempty"`
	gbacClient GBACClient // this is a client convenience, not required for eg. saving
}

//go:generate genvalidate Group

// Resource represents a resource in the AuthZ GBAC and RBAC systems.
type Resource struct {
	ID uuid.UUID `json:"id" yaml:"id" validate:"notnil"`
}

//go:generate genvalidate Resource

// NewGBACClient creates a new GBAC AuthZ client, wrapping an existing low-level AuthZ client
func NewGBACClient(client *Client) *GBACClient {
	return &GBACClient{
		client: client,
	}
}

func (c GBACClient) newUser(id uuid.UUID) User {
	return User{
		ID:         id,
		gbacClient: c,
	}
}

func (c GBACClient) newGroup(id uuid.UUID, name string) Group {
	return Group{
		ID:         id,
		Name:       name,
		gbacClient: c,
	}
}

// FlushCache flushes underlying caches
func (c GBACClient) FlushCache() error {
	return ucerr.Wrap(c.client.FlushCache())
}

// GetUser returns a wrapped user UUID
func (c GBACClient) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	obj, err := c.client.GetObject(ctx, id)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if obj.TypeID != UserObjectTypeID {
		return nil, ucerr.Errorf("object '%v' has object type '%v', not '%v'", obj.ID, obj.TypeID, UserObjectTypeID)
	}
	user := c.newUser(obj.ID)
	return &user, nil
}

// GetGroup returns a wrapped group UUID and name
func (c GBACClient) GetGroup(ctx context.Context, id uuid.UUID) (*Group, error) {
	obj, err := c.client.GetObject(ctx, id)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if obj.TypeID != GroupObjectTypeID {
		return nil, ucerr.Errorf("object '%v' has object type '%v', not '%v'", obj.ID, obj.TypeID, GroupObjectTypeID)
	}
	group := c.newGroup(obj.ID, *(obj.Alias))
	return &group, nil
}

// CreateGroup creates a group with the given ID and name
func (c GBACClient) CreateGroup(ctx context.Context, id uuid.UUID, name string) (*Group, error) {
	obj, err := c.client.CreateObject(ctx, id, GroupObjectTypeID, name)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	group := c.newGroup(obj.ID, name)
	return &group, nil
}

// CreateUserToGroupAttributeSet creates an attribute set that can be used to connect users to groups
func (c GBACClient) CreateUserToGroupAttributeSet(ctx context.Context, name string, attributes Attributes) (*AttributeSet, error) {
	et, err := c.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), UserObjectTypeID, GroupObjectTypeID, name, attributes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	as := AttributeSet(*et)
	return &as, nil
}

// CreateGroupToGroupAttributeSet creates an attribute set that can be used to connect groups to other groups
func (c GBACClient) CreateGroupToGroupAttributeSet(ctx context.Context, name string, attributes Attributes) (*AttributeSet, error) {
	et, err := c.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), GroupObjectTypeID, GroupObjectTypeID, name, attributes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	as := AttributeSet(*et)
	return &as, nil
}

// CreateUserToResourceAttributeSet creates an attribute set that can be used to connect users to resources
func (c GBACClient) CreateUserToResourceAttributeSet(ctx context.Context, name string, resourceTypeID uuid.UUID, attributes Attributes) (*AttributeSet, error) {
	et, err := c.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), UserObjectTypeID, resourceTypeID, name, attributes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	as := AttributeSet(*et)
	return &as, nil
}

// CreateGroupToResourceAttributeSet creates an attribute set that can be used to connect groups to resources
func (c GBACClient) CreateGroupToResourceAttributeSet(ctx context.Context, name string, resourceTypeID uuid.UUID, attributes Attributes) (*AttributeSet, error) {
	et, err := c.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), GroupObjectTypeID, resourceTypeID, name, attributes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	as := AttributeSet(*et)
	return &as, nil
}

// CreateResourceToResourceAttributeSet creates an attribute set that can be used to connect resources to other resources
func (c GBACClient) CreateResourceToResourceAttributeSet(ctx context.Context, name string, sourceResourceTypeID uuid.UUID, targetResourceTypeID uuid.UUID, attributes Attributes) (*AttributeSet, error) {
	et, err := c.client.CreateEdgeType(ctx, uuid.Must(uuid.NewV4()), sourceResourceTypeID, targetResourceTypeID, name, attributes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	as := AttributeSet(*et)
	return &as, nil
}

func (c GBACClient) addAttributeSet(ctx context.Context, sourceID uuid.UUID, targetID uuid.UUID, name string) (uuid.UUID, error) {
	edgeTypeID, err := c.client.FindEdgeTypeID(ctx, name)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	edge, err := c.client.CreateEdge(ctx, uuid.Nil, sourceID, targetID, edgeTypeID, IfNotExists())
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return edge.ID, nil
}

func (c GBACClient) deleteAttributeSet(ctx context.Context, id1, id2 uuid.UUID, name string) error {
	edgeTypeID, err := c.client.FindEdgeTypeID(ctx, name)
	if err != nil {
		return ucerr.Wrap(err)
	}
	edge, err := c.client.FindEdge(ctx, id1, id2, edgeTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(c.client.DeleteEdge(ctx, edge.ID))
}

func (c GBACClient) deleteAllAttributeSets(ctx context.Context, id1, id2 uuid.UUID) error {
	edges, err := c.client.ListEdgesBetweenObjects(ctx, id1, id2)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, edge := range edges {
		if err := c.client.DeleteEdge(ctx, edge.ID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func (c GBACClient) checkAttribute(ctx context.Context, id1, id2 uuid.UUID, name string) (bool, error) {
	resp, err := c.client.CheckAttribute(ctx, id1, id2, name)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	return resp.HasAttribute, nil
}
