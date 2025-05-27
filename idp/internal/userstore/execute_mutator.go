package userstore

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

// ExecuteMutator is an internal idp method that executes a mutator
func ExecuteMutator(
	ctx context.Context,
	req idp.ExecuteMutatorRequest,
	tenantID uuid.UUID,
	authzClient *authz.Client,
	searchUpdateConfig *config.SearchUpdateConfig,
) ([]uuid.UUID, int, error) {
	startTime := time.Now().UTC()

	ts := multitenant.MustGetTenantState(ctx)
	configStorage := storage.NewFromTenantState(ctx, ts)

	mutator, err := configStorage.GetLatestMutator(ctx, req.MutatorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusBadRequest, ucerr.Wrap(err)
		}

		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	defer logMutatorDuration(ctx, mutator.ID, mutator.Version, startTime)
	logMutatorCall(ctx, mutator.ID, mutator.Version)

	cm, err := storage.NewUserstoreColumnManager(ctx, configStorage)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}
	columns := cm.GetColumns()

	columnMutations, code, err := getColumnMutations(ctx, configStorage, columns, mutator, req)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	dtm, err := storage.NewDataTypeManager(ctx, configStorage)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	noColumnIDs := set.NewUUIDSet()
	noPurposeIDs := set.NewUUIDSet()

	var usersByRegion map[region.DataRegion][]storage.User
	if req.Region != "" {
		regDB, ok := ts.UserRegionDbMap[req.Region]
		if !ok {
			return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "data region '%s' is not available for tenant", req.Region)
		}
		us := storage.NewUserStorage(ctx, regDB, req.Region, tenantID)
		var users []storage.User
		users, code, err = us.GetUsersForSelector(
			ctx,
			cm,
			dtm,
			startTime,
			column.DataLifeCycleStateLive,
			columns,
			mutator.SelectorConfig,
			req.SelectorValues,
			noColumnIDs,
			noPurposeIDs,
			nil,
			true,
		)
		if err != nil {
			logMutatorNotFoundError(ctx, mutator.ID, mutator.Version)
			return nil, code, ucerr.Wrap(err)
		}
		if len(users) > 0 {
			usersByRegion = map[region.DataRegion][]storage.User{req.Region: users}
		}
	} else {
		umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, tenantID)
		usersByRegion, code, err = umrs.GetUsersForSelector(
			ctx,
			cm,
			dtm,
			startTime,
			column.DataLifeCycleStateLive,
			columns,
			mutator.SelectorConfig,
			req.SelectorValues,
			noColumnIDs,
			noPurposeIDs,
			nil,
			true,
		)
		if err != nil {
			logMutatorNotFoundError(ctx, mutator.ID, mutator.Version)
			return nil, code, ucerr.Wrap(err)
		}
	}

	// look up latest versions of global and mutator access policies
	globalAP, mutatorAP, thresholdAP, err :=
		configStorage.GetAccessPolicies(
			ctx,
			tenantID,
			policy.AccessPolicyGlobalMutatorID,
			mutator.AccessPolicyID,
		)
	if err != nil {
		logMutatorConfigError(ctx, mutator.ID, mutator.Version)
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	clientAPs := []*policy.AccessPolicy{globalAP, mutatorAP}

	// build base context
	baseAPContext := tokenizer.BuildBaseAPContext(ctx, req.Context, policy.ActionExecute)

	// verify rate threshold is not exceeded if specified
	allowed, err := thresholdAP.CheckRateThreshold(ctx, configStorage, baseAPContext, mutator.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if !allowed {
		uclog.Infof(ctx, "mutator '%v' execution failed due to rate limit", mutator.ID)
		if thresholdAP.Thresholds.AnnounceMaxExecutionFailure {
			return nil, http.StatusTooManyRequests, ucerr.Friendlyf(nil, "access policy rate threshold exceeded")
		}
		return nil, http.StatusOK, nil
	}

	// filter out users that do not pass access policies
	approvedUsersByRegion := map[region.DataRegion][]storage.User{}
	for r, users := range usersByRegion {
		approvedUsers := []storage.User{}
		for _, u := range users {
			apContext := baseAPContext
			apContext.User = u.Profile

			allowed := true
			for _, clientAP := range clientAPs {
				allowed, _, err = tokenizer.ExecuteAccessPolicy(ctx, clientAP, apContext, authzClient, configStorage)
				if err != nil {
					logMutatorConfigError(ctx, mutator.ID, mutator.Version)
					return nil, http.StatusInternalServerError, ucerr.Wrap(err)
				}
				if !allowed {
					break
				}
			}

			if allowed {
				approvedUsers = append(approvedUsers, u)

				if allowed = thresholdAP.CheckResultThreshold(len(approvedUsers)); !allowed {
					uclog.Infof(ctx, "mutator '%v' execution failed due to result limit", mutator.ID)
					if thresholdAP.Thresholds.AnnounceMaxResultFailure {
						return nil, http.StatusBadRequest, ucerr.Friendlyf(nil, "access policy result threshold exceeded")
					}
					return nil, http.StatusOK, nil
				}
			}
		}

		if len(approvedUsers) > 0 {
			approvedUsersByRegion[r] = approvedUsers
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	userIDs := []uuid.UUID{}
	var firstErr error
	retCode := http.StatusOK

	for r, users := range approvedUsersByRegion {
		regionDB, ok := ts.UserRegionDbMap[r]
		if !ok {
			return nil, http.StatusInternalServerError, ucerr.Friendlyf(nil, "region '%v' not an available remote region", r)
		}
		userStorage := storage.NewUserStorage(ctx, regionDB, r, tenantID)

		wg.Add(1)
		go func(configStorage *storage.Storage, userStorage *storage.UserStorage, regionUsers []storage.User) {
			defer wg.Done()

			// apply updates for users that pass access policy
			uvu, err := newUserValueUpdater(ctx, configStorage, userStorage, searchUpdateConfig, startTime, tenantID, uuid.Nil)
			if err != nil {
				uclog.Errorf(ctx, "error creating user value updater: %v", err)
				mu.Lock()
				defer mu.Unlock()
				if firstErr == nil {
					firstErr = err
					retCode = http.StatusInternalServerError
				}
				return
			}

			for _, u := range regionUsers {
				uvu.setUser(&u.BaseUser)

				// update user values for each column mutation
				if code, err := uvu.applyMutations(ctx, &u, columnMutations); err != nil {
					if ucdb.IsUniqueViolation(err) {
						uclog.Warningf(ctx, "error applying column mutations: %v", err)
					} else {
						uclog.Errorf(ctx, "error applying column mutations: %v", err)
					}
					mu.Lock()
					defer mu.Unlock()
					if firstErr == nil {
						firstErr = err
						retCode = code
					}
					return
				}

				// save the user changes if there are any
				if uvu.hasChanges() {
					if err := uvu.saveChanges(ctx); err != nil {
						isUniqueViolation := ucdb.IsUniqueViolation(err)
						if isUniqueViolation {
							uclog.Warningf(ctx, "error saving user changes: %v", err)
						} else {
							uclog.Errorf(ctx, "error saving user changes: %v", err)
						}
						mu.Lock()
						defer mu.Unlock()
						if firstErr == nil {
							firstErr = err
							if isUniqueViolation {
								retCode = http.StatusConflict
							} else {
								retCode = http.StatusInternalServerError
							}
						}
						return
					}
				}

				mu.Lock()
				userIDs = append(userIDs, u.ID)
				mu.Unlock()
			}
		}(configStorage, userStorage, users)
	}

	wg.Wait()

	if firstErr == nil {
		logMutatorSuccess(ctx, mutator.ID, mutator.Version)
	}

	return userIDs, retCode, ucerr.Wrap(firstErr)
}

// OpenAPI Summary: Execute Mutator
// OpenAPI Tags: Mutators
// OpenAPI Description: This endpoint executes a specified mutator (custom write API).
func (h *handler) executeMutatorHandler(ctx context.Context, req idp.ExecuteMutatorRequest) (*idp.ExecuteMutatorResponse, int, []auditlog.Entry, error) {

	if err := h.ensureTenantMember(false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	ts := multitenant.MustGetTenantState(ctx)

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	userIDs, code, err := ExecuteMutator(ctx, req, ts.ID, authzClient, h.searchUpdateConfig)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		case http.StatusTooManyRequests:
			return nil, http.StatusTooManyRequests, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return &idp.ExecuteMutatorResponse{UserIDs: userIDs}, http.StatusOK, nil, nil
}
