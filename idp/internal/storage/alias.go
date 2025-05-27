package storage

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
)

// SaveAlias will update an authz object for a user ID with the alias
func (s *UserStorage) SaveAlias(ctx context.Context, userID uuid.UUID, alias string) error {
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if featureflags.IsEnabledForTenant(ctx, featureflags.AsyncAuthzForUserStore, s.tenantID) {
		go func(ctxInner context.Context) {
			retryCount := 0
			for {
				// Since the creation is async we need to wait for the object to be created before updating it
				if _, err := authzClient.GetObject(ctxInner, userID, authz.Source("idp")); err != nil {
					if errors.Is(err, authz.ErrObjectNotFound) && retryCount < 20 {
						time.Sleep(100 * time.Millisecond)
						retryCount++
						continue
					}
					// TODO clean up
					uclog.Errorf(ctxInner, "Failed to get the AuthZ object for user %v with %v ", userID, err)
					return
				}
				break
			}
			retryCount = 0
			for {
				if _, err := authzClient.UpdateObject(ctxInner, userID, &alias, authz.Source("idp")); err != nil {
					if jsonclient.IsHTTPStatusConflict(err) && retryCount < 20 {
						time.Sleep(100 * time.Millisecond)
						retryCount++
						continue
					}
					// TODO clean up
					uclog.Errorf(ctxInner, "Failed to update the AuthZ object for user %v alias %v with %v ", userID, &alias, err)
					return
				}
				break
			}
		}(context.WithoutCancel(ctx))
	} else {
		if _, err := authzClient.UpdateObject(ctx, userID, &alias, authz.Source("idp")); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// DeleteAlias will clear the alias from the authz object for a user ID
func (s *UserStorage) DeleteAlias(ctx context.Context, userID uuid.UUID) error {
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if _, err := authzClient.UpdateObject(ctx, userID, nil, authz.Source("idp")); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
