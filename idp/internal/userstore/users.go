package userstore

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/shared"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

// OpenAPI Summary: Create User With Mutator
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint creates a user and updates it with the specified mutator.
func (h *handler) createUserstoreUser(ctx context.Context, req idp.CreateUserWithMutatorRequest) (uuid.UUID, int, []auditlog.Entry, error) {
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)

	mutator, err := s.GetLatestMutator(ctx, req.MutatorID)
	if err != nil {
		return uuid.Nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if !mutator.UsableForCreate() {
		return uuid.Nil, http.StatusBadRequest, nil, ucerr.Errorf("mutator '%s' must have a selector that selects a single user by ID", mutator.ID)
	}

	organizationID, err := shared.ValidateUserOrganizationForRequest(ctx, req.OrganizationID)
	if err != nil {
		return uuid.Nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if !ts.UseOrganizations {
		req.OrganizationID = ts.CompanyID
	} else {
		if organizationID.IsNil() {
			return uuid.Nil, http.StatusBadRequest, nil, ucerr.Errorf("organization ID is required")
		}
		req.OrganizationID = organizationID
	}

	baseUser, code, err := CreateUserHelper(ctx, h.searchUpdateConfig, req, true)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return uuid.Nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return uuid.Nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusForbidden:
			return uuid.Nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		default:
			return uuid.Nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return baseUser.ID, http.StatusOK, nil, nil
}

// CreateUserHelper is the helper function for createUserstoreUser as well as authn's CreateUser method
func CreateUserHelper(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.CreateUserWithMutatorRequest,
	executeMutator bool,
) (*storage.BaseUser, int, error) {
	baseUser := storage.BaseUser{
		VersionBaseModel: ucdb.NewVersionBase(),
		OrganizationID:   req.OrganizationID,
		Region:           req.Region,
	}
	if req.ID != uuid.Nil {
		baseUser.ID = req.ID
	}

	ts := multitenant.MustGetTenantState(ctx)

	var userStorage *storage.UserStorage
	var reg region.DataRegion
	if req.Region != "" {
		regDB, ok := ts.UserRegionDbMap[req.Region]
		if !ok {
			return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "data region '%s' is not available for tenant", req.Region)
		}
		userStorage = storage.NewUserStorage(ctx, regDB, req.Region, ts.ID)
		reg = req.Region
	} else {
		userStorage = storage.NewUserStorage(ctx, ts.TenantDB, ts.PrimaryUserRegion, ts.ID)
		reg = ts.PrimaryUserRegion
	}

	u := &storage.User{BaseUser: baseUser}
	if err := userStorage.SaveUserWithAsyncAuthz(ctx, u); err != nil {
		go func() {
			if err := userStorage.DeletePartiallyCreatedUser(context.WithoutCancel(ctx), u.ID); err != nil {
				uclog.Errorf(ctx, "error deleting user '%s' after failed save: %v", u.ID, err)
			}
		}()
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if !executeMutator {
		return &baseUser, http.StatusOK, nil
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	userIDs, code, err := ExecuteMutator(
		ctx,
		idp.ExecuteMutatorRequest{
			MutatorID:      req.MutatorID,
			Context:        req.Context,
			SelectorValues: []any{u.ID},
			RowData:        req.RowData,
			Region:         reg,
		},
		ts.ID,
		authzClient,
		searchUpdateConfig,
	)

	if err != nil {
		go func() {
			if err := userStorage.DeletePartiallyCreatedUser(context.WithoutCancel(ctx), u.ID); err != nil {
				uclog.Errorf(ctx, "error deleting user '%s' after failed mutator: %v", u.ID, err)
			}
		}()
		return nil, code, ucerr.Wrap(err)
	} else if len(userIDs) != 1 {
		go func() {
			if err := userStorage.DeletePartiallyCreatedUser(context.WithoutCancel(ctx), u.ID); err != nil {
				uclog.Errorf(ctx, "error deleting user '%s' after failed mutator: %v", u.ID, err)
			}
		}()

		// This can happen if the mutator access policy evaluates to false for the user.
		return nil,
			http.StatusForbidden,
			ucerr.Friendlyf(nil, "mutator could not be executed - check access policy")
	}

	return &baseUser, http.StatusOK, nil
}

// OpenAPI Summary: Delete User
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint deletes a user by ID
func (h *handler) deleteUserstoreUser(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	code, err := DeleteUser(ctx, h.searchUpdateConfig, id)
	if err != nil {
		switch code {
		case http.StatusForbidden:
			return http.StatusForbidden, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// DeleteUser is a helper method for deleting a user
func DeleteUser(ctx context.Context, searchUpdateConfig *config.SearchUpdateConfig, id uuid.UUID) (int, error) {
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)
	umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)

	user, reg, err := umrs.GetBaseUser(ctx, id, false)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), ucerr.Wrap(err)
	}

	if _, err := shared.ValidateUserOrganizationForRequest(ctx, user.OrganizationID); err != nil {
		return http.StatusForbidden, ucerr.Wrap(err)
	}

	// remove all purposes for all columns via update user mutator

	mutator, err := s.GetLatestMutator(ctx, constants.UpdateUserMutatorID)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	fullUpdateDeletedValue := idp.ValueAndPurposes{Value: nil}
	partialUpdateDeletedValue := idp.ValueAndPurposes{ValueDeletions: idp.MutatorColumnCurrentValue}

	deletedValues := map[string]idp.ValueAndPurposes{}

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for _, columnID := range mutator.ColumnIDs {
		c := cm.GetColumnByID(columnID)
		if c == nil {
			return http.StatusInternalServerError, ucerr.Errorf("mutator column ID is invalid: '%v'", columnID)
		}
		if c.Attributes.Constraints.PartialUpdates {
			deletedValues[c.Name] = partialUpdateDeletedValue
		} else {
			deletedValues[c.Name] = fullUpdateDeletedValue
		}
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	deletedIDs, _, err := ExecuteMutator(
		ctx,
		idp.ExecuteMutatorRequest{
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			SelectorValues: []any{user.ID},
			RowData:        deletedValues,
			Region:         reg,
		},
		ts.ID,
		authzClient,
		searchUpdateConfig,
	)

	if err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if len(deletedIDs) != 1 {
		return http.StatusInternalServerError, ucerr.New("expected exactly one value from mutator")
	}

	// soft delete user

	regDB, ok := ts.UserRegionDbMap[reg]
	if !ok {
		return http.StatusInternalServerError, ucerr.Errorf("db for region %s not found", reg)
	}
	us := storage.NewUserStorage(ctx, regDB, reg, ts.ID)
	if err := us.DeleteUser(ctx, user.ID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil
}
