package userstore

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/plex/manager"
)

// IdpS3ShimController is the controller an S3 shim
type IdpS3ShimController struct {
	ts          *tenantmap.TenantState
	jwtVerifier auth.Verifier
	cacheConfig *cache.Config
	objectStore *storage.ShimObjectStore
}

// NewIdpS3ShimController creates a new idpS3ShimController
func NewIdpS3ShimController(ts *tenantmap.TenantState, jwtVerifier auth.Verifier, cacheConfig *cache.Config, objectStore *storage.ShimObjectStore) *IdpS3ShimController {
	return &IdpS3ShimController{
		ts:          ts,
		jwtVerifier: jwtVerifier,
		cacheConfig: cacheConfig,
		objectStore: objectStore,
	}
}

// CheckPermission implements the S3Shim Controller interface
func (c *IdpS3ShimController) CheckPermission(ctx context.Context, jwt string, path string) (bool, error) {

	tokenSource, err := m2m.GetM2MTokenSource(ctx, c.ts.ID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	azc, err := authz.NewClient(c.ts.GetTenantURL(), authz.JSONClient(tokenSource))
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	clientContext := policy.ClientContext{"path": path}
	if ctxToken, err := auth.AddTokenToContext(ctx, jwt, c.jwtVerifier, false); err == nil {
		ctx = ctxToken
	} else {
		mgr := manager.NewFromDB(c.ts.TenantDB, c.cacheConfig)
		apps, err := mgr.GetLoginApps(ctx, c.ts.ID, c.ts.CompanyID)
		if err != nil {
			return false, ucerr.Wrap(err)
		}
		if len(apps) == 0 {
			return false, ucerr.Wrap(ucerr.Friendlyf(nil, "no login apps found for tenant ID %v", c.ts.ID))
		}
		ctx = auth.SetSubjectTypeAndUUID(ctx, apps[0].ID, authz.ObjectTypeLoginApp)
	}

	apContext := tokenizer.BuildBaseAPContext(ctx, clientContext, policy.ActionExecute)

	s := storage.NewFromTenantState(ctx, c.ts)

	_, s3shimAP, thresholdAP, err :=
		s.GetAccessPolicies(
			ctx,
			c.ts.ID,
			policy.AccessPolicyGlobalAccessorID,
			c.objectStore.AccessPolicyID)
	if err != nil {
		return false, ucerr.Wrap(err)
	}

	allowed, err := thresholdAP.CheckRateThreshold(ctx, s, apContext, uuid.Nil)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	if !allowed {
		return false, nil
	}

	allowed, _, err = tokenizer.ExecuteAccessPolicy(ctx, s3shimAP, apContext, azc, s)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	if !allowed {
		return false, nil
	}

	return true, nil
}

// TransformData implements the S3Shim Controller interface
func (c *IdpS3ShimController) TransformData(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}
