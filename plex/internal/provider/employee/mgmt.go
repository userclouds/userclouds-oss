package employee

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
)

type mgmtClient struct {
	iface.BaseManagementClient
	id   uuid.UUID
	name string
}

// NewManagementClient returns a new client that is configured to only perform management tasks
func NewManagementClient(ctx context.Context, id uuid.UUID, name string) (iface.ManagementClient, error) {
	return &mgmtClient{
		id:   id,
		name: name,
	}, nil
}

func (mc *mgmtClient) GetUser(ctx context.Context, userID string) (*iface.UserProfile, error) {
	// TODO: need to actually create a user object in the tenant IDP as part of the create employee flow, and
	// return that user profile object as part of this call.
	return &iface.UserProfile{
		ID:     userID,
		Authns: []idp.UserAuthn{},
	}, nil
}

func (mc mgmtClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v'", tenantplex.ProviderTypeEmployee, mc.name, mc.id)
}
