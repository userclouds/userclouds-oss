package loginapp

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/tenantplex"
)

// CheckLoginAccessForUser returns true if the user has access to the app
func CheckLoginAccessForUser(ctx context.Context, tc tenantplex.TenantConfig, app *tenantplex.App, userID string) (bool, error) {
	if app.RestrictedAccess {

		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithClientSecret(ctx, app.ClientID, app.ClientSecret)
		if err != nil {
			return false, ucerr.Wrap(err)
		}

		userUUID, err := uuid.FromString(userID)
		if err != nil {
			return false, ucerr.Wrap(err)
		}

		resp, err := authzClient.CheckAttribute(ctx, userUUID, app.ID, authz.CanLoginAttribute)
		if err != nil {
			return false, ucerr.Wrap(err)
		}

		if !resp.HasAttribute {
			return false, nil
		}
	}

	return true, nil
}

// AddLoginAccessForUserIfNecessary adds a login access edge for the user if the app is restricted
func AddLoginAccessForUserIfNecessary(ctx context.Context, tc tenantplex.TenantConfig, app *tenantplex.App, userID string) error {
	if app.RestrictedAccess {
		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithClientSecret(ctx, app.ClientID, app.ClientSecret)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if userUUID, err := uuid.FromString(userID); err == nil {
			if _, err := authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userUUID, app.ID, authz.CanLoginEdgeTypeID); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}
