package authz

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// Membership is a tuple of a User, a Group, and the User's Role, which refers to the
// AttributeSet that applies for the User/Group relationship, or Edge. The ID uniquely
// identifies this relationship.
type Membership struct {
	ID    uuid.UUID `json:"id" validate:"notnil"`
	User  User      `json:"user"`
	Group Group     `json:"group"`
	Role  string    `json:"role" validate:"notempty"`
}

//go:generate genvalidate Membership

// RBACClient is a high-level RBAC client for the AuthZ service
type RBACClient struct {
	gbacClient *GBACClient
}

// NewRBACClient creates a new RBAC AuthZ client, wrapping a GBAC client
func NewRBACClient(client *Client) *RBACClient {
	return &RBACClient{
		gbacClient: NewGBACClient(client),
	}
}

// FlushCache flushes underlying caches
func (c *RBACClient) FlushCache() error {
	return ucerr.Wrap(c.gbacClient.FlushCache())
}

// CreateGroup creates an AuthZ group object with given UUID and name.
func (c *RBACClient) CreateGroup(ctx context.Context, id uuid.UUID, name string) (*Group, error) {
	return c.gbacClient.CreateGroup(ctx, id, name)
}

// CreateRole creates a new AuthZ role attribute set for user/group memberships.
func (c *RBACClient) CreateRole(ctx context.Context, roleName string) (*AttributeSet, error) {
	as, err := c.gbacClient.CreateUserToGroupAttributeSet(ctx, roleName, Attributes{Attribute{Name: roleName, Direct: true}})
	return as, ucerr.Wrap(err)
}

// GetGroup retrieves an AuthZ group object by UUID.
func (c *RBACClient) GetGroup(ctx context.Context, id uuid.UUID) (*Group, error) {
	return c.gbacClient.GetGroup(ctx, id)
}

// GetRole retrieves an AuthZ role attribute set for the specified role name
func (c *RBACClient) GetRole(ctx context.Context, roleName string) (*AttributeSet, error) {
	roleID, err := c.gbacClient.client.FindEdgeTypeID(ctx, roleName)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	role, err := c.gbacClient.client.GetEdgeType(ctx, roleID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	r := AttributeSet(*role)
	return &r, nil
}

// GetUser retrieves an AuthZ user object by UUID.
func (c *RBACClient) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return c.gbacClient.GetUser(ctx, id)
}

// AddGroupRole creates a membership between the user and a group with the specified role.
func (u User) AddGroupRole(ctx context.Context, g Group, role string) (*Membership, error) {
	membershipID, err := u.AddGroupAttributeSet(ctx, g, role)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &Membership{
		ID:    membershipID,
		User:  u,
		Group: g,
		Role:  role,
	}, nil
}

// GetGroupRoles returns all roles that a user has in a group.
func (u User) GetGroupRoles(ctx context.Context, g Group) ([]string, error) {
	roles, err := u.ListGroupAttributes(ctx, g)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return roles, nil
}

// GetMemberships retrieves all user-group memberships for the user (duplicate entries if multiple edges exist)
func (u User) GetMemberships(ctx context.Context) ([]Membership, error) {
	authZClient := u.gbacClient.client

	memberships := []Membership{}
	cursor := pagination.CursorBegin
	for {
		resp, err := authZClient.ListEdgesOnObject(ctx, u.ID, Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, edge := range resp.Data {
			// Group roles always have group ID as "source" and user ID as "target";
			// ignore other edges that an app may have created.
			if edge.SourceObjectID != u.ID {
				continue
			}

			edgeType, err := authZClient.GetEdgeType(ctx, edge.EdgeTypeID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			// Ignore non-group objects
			if edgeType.TargetObjectTypeID != GroupObjectTypeID {
				continue
			}

			obj, err := authZClient.GetObject(ctx, edge.TargetObjectID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			memberships = append(memberships, Membership{
				ID:    edge.ID,
				User:  u,
				Group: u.gbacClient.newGroup(obj.ID, *obj.Alias),
				Role:  edgeType.TypeName,
			})
		}
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	return memberships, nil
}

// RemoveFromGroup removes a user from a group across all roles.
func (u User) RemoveFromGroup(ctx context.Context, g Group) error {
	return ucerr.Wrap(u.DeleteAllGroupAttributeSets(ctx, g))
}

// RemoveGroupRole removes a specific user-group role.
func (u User) RemoveGroupRole(ctx context.Context, g Group, role string) error {
	return ucerr.Wrap(u.DeleteGroupAttributeSet(ctx, g, role))
}

// RemoveMembership removes a specific user-group role by its membership ID
func (u User) RemoveMembership(ctx context.Context, m Membership) error {
	return ucerr.Wrap(u.gbacClient.client.DeleteEdge(ctx, m.ID))
}

// AddUserRole creates a membership between the group and a user with the specified role.
func (g Group) AddUserRole(ctx context.Context, u User, role string) (*Membership, error) {
	return u.AddGroupRole(ctx, g, role)
}

// GetMemberships retrieves all user-group memberships for the group (duplicates if multiple edges exist)
func (g Group) GetMemberships(ctx context.Context) ([]Membership, error) {
	authZClient := g.gbacClient.client

	memberships := []Membership{}
	cursor := pagination.CursorBegin
	for {
		resp, err := authZClient.ListEdgesOnObject(ctx, g.ID, Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, edge := range resp.Data {
			if edge.TargetObjectID != g.ID {
				continue
			}

			edgeType, err := authZClient.GetEdgeType(ctx, edge.EdgeTypeID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			// Ignore non-user objects
			if edgeType.SourceObjectTypeID != UserObjectTypeID {
				continue
			}

			obj, err := authZClient.GetObject(ctx, edge.SourceObjectID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			memberships = append(memberships, Membership{
				ID:    edge.ID,
				User:  g.gbacClient.newUser(obj.ID),
				Group: g,
				Role:  edgeType.TypeName,
			})
		}
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	return memberships, nil
}

// GetUserRoles returns all roles that a user has in a group.
func (g Group) GetUserRoles(ctx context.Context, u User) ([]string, error) {
	roles, err := u.GetGroupRoles(ctx, g)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return roles, nil
}

// RemoveMembership removes a specific user-group role by its membership ID
func (g Group) RemoveMembership(ctx context.Context, m Membership) error {
	return ucerr.Wrap(g.gbacClient.client.DeleteEdge(ctx, m.ID))
}

// RemoveUser removes a user from a group across all roles.
func (g Group) RemoveUser(ctx context.Context, u User) error {
	return ucerr.Wrap(u.RemoveFromGroup(ctx, g))
}

// RemoveUserRole removes a specific user-group role.
func (g Group) RemoveUserRole(ctx context.Context, u User, role string) error {
	return ucerr.Wrap(u.RemoveGroupRole(ctx, g, role))
}
