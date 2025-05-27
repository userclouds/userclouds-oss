package uc

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
)

type mgmtClient struct {
	iface.BaseManagementClient
	id   uuid.UUID
	name string
	p    tenantplex.UCProvider

	idp *idp.ManagementClient
}

func newMgmtClient(ctx context.Context, tc *tenantplex.TenantConfig, tenantURL string, appID uuid.UUID, appOrgID uuid.UUID) (*idp.ManagementClient, error) {
	// We don't use the subject field, and the audience field is the URL of the tenant for now
	jwt, err := newJWT(ctx, tc, tenantURL, appID, appOrgID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	mgmtClient, err := idp.NewManagementClient(tenantURL, jsonclient.HeaderAuthBearer(jwt), security.PassXForwardedFor())
	return mgmtClient, ucerr.Wrap(err)
}

func (c mgmtClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', name: '%s', id: '%v'", tenantplex.ProviderTypeUC, c.name, c.id)
}

// NewManagementClient returns a new client that is configured to only perform management tasks
// Note that because we don't actually have different IDP auth or anything yet, we just use the same
// object with less config (and client implements both iface.Client & iface.ManagementClient), but
// that will likely change as we get more mature.
func NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, id uuid.UUID, name string, uc tenantplex.UCProvider, appID uuid.UUID, appOrgID uuid.UUID) (iface.ManagementClient, error) {
	mc, err := newMgmtClient(ctx, tc, uc.IDPURL, appID, appOrgID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &mgmtClient{
		id:   id,
		name: name,
		p:    uc,
		idp:  mc,
	}, nil
}

func (c *mgmtClient) CreateUserWithPassword(ctx context.Context, username, password string, profile iface.UserProfile) (string, error) {
	userID, err := c.idp.CreateUserWithPassword(ctx, username, password, profile.ToUserstoreRecord())
	return userID.String(), ucerr.Wrap(err)
}

func (c *mgmtClient) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile iface.UserProfile) (string, error) {
	userID, err := c.idp.CreateUserWithOIDC(ctx, provider, issuerURL, oidcSubject, profile.ToUserstoreRecord())
	return userID.String(), ucerr.Wrap(err)
}

func (c *mgmtClient) GetUserForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, _ string) (*iface.UserProfile, error) {
	resp, err := c.idp.GetUserBaseProfileForOIDC(ctx, provider, issuerURL, oidcSubject)
	if err != nil {
		return nil, ucerr.Wrap(iface.ClassifyGetUserError(err))
	}

	return (*iface.UserProfile)(resp), nil
}

func (c *mgmtClient) GetUser(ctx context.Context, userID string) (*iface.UserProfile, error) {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	resp, err := c.idp.GetUserBaseProfileAndAuthN(ctx, userUUID)
	if err != nil {
		return nil, ucerr.Wrap(iface.ClassifyGetUserError(err))
	}
	return (*iface.UserProfile)(resp), nil
}

func (c *mgmtClient) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]iface.UserProfile, error) {
	resp, err := c.idp.ListUserBaseProfilesAndAuthNForEmail(ctx, email, authnType)
	if err != nil {
		return nil, ucerr.Wrap(err)

	}
	users := []iface.UserProfile{}
	for _, user := range resp {
		users = append(users, (iface.UserProfile)(user))
	}
	return users, nil
}

func setEmailVerified(ctx context.Context, mc *idp.ManagementClient, userID uuid.UUID, verified bool) error {
	if _, err := mc.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: userstore.Record{
			"email_verified": verified,
		},
	}); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func (c *mgmtClient) SetEmailVerified(ctx context.Context, userID string, verified bool) error {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(setEmailVerified(ctx, c.idp, userUUID, verified))
}

func (c *mgmtClient) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	return ucerr.Wrap(iface.ClassifyGetUserError(c.idp.UpdateUsernamePassword(ctx, username, password)))
}

func (c *mgmtClient) UpdateUser(ctx context.Context, userID string, profile iface.UserProfile) error {
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if _, err := c.idp.UpdateUser(ctx, userUUID, idp.UpdateUserRequest{
		Profile: profile.ToUserstoreRecord(),
	}); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func (c *mgmtClient) AddPasswordAuthnToUser(ctx context.Context, userID, username, password string) error {
	return ucerr.Wrap(iface.ClassifyGetUserError(c.idp.AddPasswordAuthnToUser(ctx, userID, username, password)))
}

func (c *mgmtClient) AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	return ucerr.Wrap(iface.ClassifyGetUserError(c.idp.AddOIDCAuthnToUser(ctx, userID, provider, issuerURL, oidcSubject)))
}
