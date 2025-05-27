package storage

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/apiclient"
)

// UserStorage is a storage object for user related tables (users, user_column_pre_delete_values, user_column_post_delete_values, user_cleanup_candidates)
type UserStorage struct {
	db       *ucdb.DB
	r        region.DataRegion
	tenantID uuid.UUID
}

// NewUserStorage returns a UserStorage object
func NewUserStorage(ctx context.Context, db *ucdb.DB, r region.DataRegion, tenantID uuid.UUID) *UserStorage {
	return &UserStorage{db: db, r: r, tenantID: tenantID}
}

// GetRegion returns the region of the UserStorage
func (s *UserStorage) GetRegion() region.DataRegion {
	return s.r
}

// SaveUserWithAsyncAuthz will save a user and create an authz object for that user with async call
func (s *UserStorage) SaveUserWithAsyncAuthz(ctx context.Context, obj *User) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if featureflags.IsEnabledForTenant(ctx, featureflags.AsyncAuthzForUserStore, s.tenantID) {
		go func(ctxInner context.Context) {
			if err := s.preSaveUser(ctxInner, obj); err != nil {
				// TODO clean up - for now we are okay leaving that user in bad AuthZ state since WH doesn't use AuthZ yet
				uclog.Errorf(ctxInner, "Failed to save the AuthZ object for user %v with %v ", obj.ID, err)
			}
		}(context.WithoutCancel(ctx))
	} else {
		if err := s.preSaveUser(ctx, obj); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return ucerr.Wrap(s.saveInnerUser(ctx, obj))
}

func (s *UserStorage) preSaveUser(ctx context.Context, u *User) error {
	authzObj := authz.Object{
		BaseModel:      ucdb.NewBaseWithID(u.ID),
		TypeID:         authz.UserObjectTypeID,
		OrganizationID: u.OrganizationID,
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if _, err := authzClient.CreateObject(ctx, authzObj.ID, authzObj.TypeID, "", authz.OrganizationID(authzObj.OrganizationID), authz.IfNotExists()); err != nil {
		// The object might already exist with a non-nil alias, so we need to check for that
		if obj, err := authzClient.GetObject(ctx, authzObj.ID); err == nil && obj.ID == authzObj.ID && obj.TypeID == authzObj.TypeID {
			return nil
		}
		return ucerr.Friendlyf(err, "failed to create authz object for user")
	}
	return nil
}

// When deleting a user, we need to soft delete any edges to/from that user
// as well as AuthNs associated with that user.
// TODO: emulating foreign keys is potentially error prone; if we add new tables with `user_id` columns, they'll also need to be deleted.
func (s *UserStorage) preDeleteUser(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	// NOTE: this is not transactional, so we can get into a bad state;
	// probably want a reconciler to clean things up, OR use a transaction
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := authzClient.DeleteObject(ctx, id); err != nil {
		if !errors.Is(err, authz.ErrObjectNotFound) {
			return ucerr.Friendlyf(err, "preDeleteUser failed to delete authz object for user")
		}
	}

	// The next set of queries are against tables that currently don't require cache sentinels to be set (no follower reads), so it's okay to call them directly
	const authnOIDCQuery = `UPDATE authns_social SET deleted=NOW() WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';`
	if _, err := s.db.ExecContext(ctx, "preDeleteUser.social", authnOIDCQuery, id); err != nil {
		return ucerr.Wrap(err)
	}

	const authnPasswordQuery = `UPDATE authns_password SET deleted=NOW() WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';`
	if _, err := s.db.ExecContext(ctx, "preDeleteUser.password", authnPasswordQuery, id); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// listUsersForEmail returns a set of base users with a given email
func (s *UserStorage) listUsersForEmail(ctx context.Context, emailColumn *Column, email string) ([]BaseUser, error) {
	const q = `
	/* bypass-known-table-check */
	SELECT u.id, u.created, u.updated, u.deleted, u._version, u.organization_id
	FROM users u
	LEFT JOIN user_column_pre_delete_values ucv ON ucv.user_id = u.id
	WHERE ucv.column_id=$1
	AND ucv.varchar_value=$2
	AND ucv.deleted='0001-01-01 00:00:00'
	AND u.deleted='0001-01-01 00:00:00';`

	var objs []BaseUser
	if err := s.db.SelectContext(ctx, "ListUsersForEmail", &objs, q, emailColumn.ID, email); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return objs, nil
}

// UserMultiRegionStorage is a facade for UserStorage that allows for multi-region access
type UserMultiRegionStorage struct {
	userRegionDbMap map[region.DataRegion]*ucdb.DB
	tenantID        uuid.UUID
}

// NewUserMultiRegionStorage returns a new UserMultiRegionStorage
func NewUserMultiRegionStorage(ctx context.Context, userRegionDbMap map[region.DataRegion]*ucdb.DB, tenantID uuid.UUID) *UserMultiRegionStorage {
	return &UserMultiRegionStorage{
		userRegionDbMap: userRegionDbMap,
		tenantID:        tenantID,
	}
}

type runAcrossRegionsOutput struct {
	mutex *sync.Mutex

	getBaseUserOutput         getBaseUserOutput
	listBaseUsersOutput       listBaseUsersOutput
	getUserOutput             getUserOutput
	getAllUserValuesOutput    getAllUserValuesOutput
	getUsersForSelectorOutput getUsersForSelectorOutput
	deleteUserOutput          deleteUserOutput
	listUsersForEmailOutput   listUsersForEmailOutput
}

func (umrs *UserMultiRegionStorage) runAcrossRegions(ctx context.Context, f func(context.Context, *UserStorage, *runAcrossRegionsOutput) (int, error), out *runAcrossRegionsOutput) (int, error) {
	if len(umrs.userRegionDbMap) == 0 {
		return http.StatusInternalServerError, ucerr.Errorf("No database regions available for tenant %v", umrs.tenantID)
	}

	if len(umrs.userRegionDbMap) == 1 {
		// only one region, so no need to spin up goroutines
		for r, db := range umrs.userRegionDbMap {
			s := NewUserStorage(ctx, db, r, umrs.tenantID)
			return f(ctx, s, out)
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	firstCode := http.StatusOK

	for r, db := range umrs.userRegionDbMap {
		wg.Add(1)
		go func(r region.DataRegion, db *ucdb.DB) {
			defer wg.Done()
			s := NewUserStorage(ctx, db, r, umrs.tenantID)
			code, err := f(ctx, s, out)
			if err != nil {
				uclog.Errorf(ctx, "Error in region %v: %v", r, err)
				mu.Lock()
				defer mu.Unlock()
				if firstErr == nil {
					firstErr = err
					firstCode = code
				}
			}
		}(r, db)
	}

	wg.Wait()

	return firstCode, ucerr.Wrap(firstErr)
}

type getUserOutput struct {
	region region.DataRegion
	user   *User
}

// GetUser returns a user from the first region that has the user (and writes a warning if the user is found in multiple regions)
func (umrs *UserMultiRegionStorage) GetUser(ctx context.Context, cm *ColumnManager, dtm *DataTypeManager, userID uuid.UUID, accessPrimaryDBOnly bool) (*User, region.DataRegion, int, error) {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
	}

	code, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		user, errCode, err := s.GetUser(ctx, cm, dtm, userID, accessPrimaryDBOnly)

		if errCode == http.StatusNotFound {
			// mask not found errors, since the user may be in another region
			return http.StatusOK, nil
		}

		if err != nil || user == nil {
			return errCode, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		if out.getUserOutput.user != nil {
			uclog.Errorf(ctx, "user %v found in multiple regions", userID)
		} else {
			out.getUserOutput.region = s.GetRegion()
			out.getUserOutput.user = user
		}

		return errCode, ucerr.Wrap(err)
	}, &out)

	if err == nil && out.getUserOutput.user == nil {
		code = http.StatusNotFound
		err = ucerr.Friendlyf(nil, "user %v not found", userID)
	}

	return out.getUserOutput.user, out.getUserOutput.region, code, ucerr.Wrap(err)
}

type getAllUserValuesOutput struct {
	region            region.DataRegion
	user              *BaseUser
	liveValues        ColumnConsentedValues
	softDeletedValues ColumnConsentedValues
}

// GetAllUserValues returns all user values from the first region that has the user (and writes a warning if the user is found in multiple regions)
func (umrs *UserMultiRegionStorage) GetAllUserValues(ctx context.Context, cm *ColumnManager, dtm *DataTypeManager, userID uuid.UUID, accessPrimaryDBOnly bool) (
	retUser *BaseUser,
	retLiveValues ColumnConsentedValues,
	retSoftDeletedValues ColumnConsentedValues,
	retRegion region.DataRegion,
	code int,
	err error,
) {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
	}

	code, err = umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		user, liveValues, softDeletedValues, errCode, err := s.GetAllUserValues(ctx, cm, dtm, userID, accessPrimaryDBOnly)

		if errCode == http.StatusNotFound {
			// mask not found errors, since the user may be in another region
			return http.StatusOK, nil
		}

		if err != nil || user == nil {
			return errCode, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		if out.getAllUserValuesOutput.user != nil {
			uclog.Errorf(ctx, "user %v found in multiple regions", userID)
		} else {
			out.getAllUserValuesOutput.region = s.GetRegion()
			out.getAllUserValuesOutput.user = user
			out.getAllUserValuesOutput.liveValues = liveValues
			out.getAllUserValuesOutput.softDeletedValues = softDeletedValues
		}

		return errCode, ucerr.Wrap(err)
	}, &out)

	if err == nil && out.getAllUserValuesOutput.user == nil {
		code = http.StatusNotFound
		err = ucerr.Friendlyf(nil, "user %v not found", userID)
	}

	return out.getAllUserValuesOutput.user, out.getAllUserValuesOutput.liveValues, out.getAllUserValuesOutput.softDeletedValues, out.getAllUserValuesOutput.region, code, ucerr.Wrap(err)
}

type getUsersForSelectorOutput struct {
	allUsers map[region.DataRegion][]User
}

// GetUsersForSelector returns users from all regions that match the selector
func (umrs *UserMultiRegionStorage) GetUsersForSelector(
	ctx context.Context,
	cm *ColumnManager,
	dtm *DataTypeManager,
	minRetentionTime time.Time,
	dlcs column.DataLifeCycleState,
	columns Columns,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues userstore.UserSelectorValues,
	expectedColumnIDs set.Set[uuid.UUID],
	expectedPurposeIDs set.Set[uuid.UUID],
	p *pagination.Paginator,
	accessPrimaryDBOnly bool,
) (map[region.DataRegion][]User, int, error) {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
		getUsersForSelectorOutput: getUsersForSelectorOutput{
			allUsers: map[region.DataRegion][]User{},
		},
	}

	code, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		users, errCode, err := s.GetUsersForSelector(
			ctx,
			cm,
			dtm,
			minRetentionTime,
			dlcs,
			columns,
			selectorConfig,
			selectorValues,
			expectedColumnIDs,
			expectedPurposeIDs,
			p,
			accessPrimaryDBOnly,
		)

		if err != nil || len(users) == 0 {
			return errCode, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		out.getUsersForSelectorOutput.allUsers[s.GetRegion()] = users

		return errCode, ucerr.Wrap(err)
	}, &out)

	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return out.getUsersForSelectorOutput.allUsers, code, nil
}

type getBaseUserOutput struct {
	region region.DataRegion
	user   *BaseUser
}

// GetBaseUser returns a base user from the first region that has the user (and writes a warning if the user is found in multiple regions)
func (umrs *UserMultiRegionStorage) GetBaseUser(ctx context.Context, id uuid.UUID, accessPrimaryDBOnly bool) (*BaseUser, region.DataRegion, error) {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
	}

	_, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		user, err := s.GetBaseUser(ctx, id, accessPrimaryDBOnly)

		if errors.Is(err, sql.ErrNoRows) {
			// mask not found errors, since the user may be in another region
			return http.StatusOK, nil
		}

		if err != nil || user == nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		if out.getBaseUserOutput.user != nil {
			uclog.Errorf(ctx, "user %v found in multiple regions", id)
		} else {
			out.getBaseUserOutput.region = s.GetRegion()
			out.getBaseUserOutput.user = user
		}

		return http.StatusOK, ucerr.Wrap(err)
	}, &out)

	if err == nil && out.getBaseUserOutput.user == nil {
		err = ucerr.Friendlyf(nil, "user %v not found", id)
	}

	return out.getBaseUserOutput.user, out.getBaseUserOutput.region, ucerr.Wrap(err)
}

// GetBaseUserFromRegion returns a base user from a specific region
func (umrs *UserMultiRegionStorage) GetBaseUserFromRegion(ctx context.Context, r region.DataRegion, id uuid.UUID, accessPrimaryDBOnly bool) (*BaseUser, error) {
	s := NewUserStorage(ctx, umrs.userRegionDbMap[r], r, umrs.tenantID)
	return s.GetBaseUser(ctx, id, accessPrimaryDBOnly)
}

type deleteUserOutput struct {
	deleted bool
}

// DeleteUser deletes a user from the first region that has the user (and writes a warning if the user is found in multiple regions)
func (umrs *UserMultiRegionStorage) DeleteUser(ctx context.Context, id uuid.UUID) error {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
	}

	_, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		if err := s.DeleteUser(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// mask not found errors, since the user may be in another region
				return http.StatusOK, nil
			}

			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()

		if out.deleteUserOutput.deleted {
			uclog.Errorf(ctx, "user %v found in multiple regions", id)
		} else {
			out.deleteUserOutput.deleted = true
		}

		return http.StatusOK, nil
	}, &out)

	if err == nil && !out.deleteUserOutput.deleted {
		err = ucerr.Friendlyf(nil, "user %v not found", id)
	}

	return ucerr.Wrap(err)
}

// CleanupUsers cleans up users from all regions
func (umrs *UserMultiRegionStorage) CleanupUsers(ctx context.Context, cm *ColumnManager, maxCandidates int, dryRun bool) (int, error) {
	code, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		return s.CleanupUsers(ctx, cm, maxCandidates, dryRun)
	}, &runAcrossRegionsOutput{})

	return code, ucerr.Wrap(err)
}

type listBaseUsersOutput struct {
	baseUsers      map[region.DataRegion][]BaseUser
	responseFields map[region.DataRegion]*pagination.ResponseFields
}

// ListBaseUsersPaginated returns base users from all regions, respecting pagination
func (umrs *UserMultiRegionStorage) ListBaseUsersPaginated(ctx context.Context, p pagination.Paginator, accessPrimaryDBOnly bool) ([]BaseUser, *pagination.ResponseFields, error) {
	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
		listBaseUsersOutput: listBaseUsersOutput{
			baseUsers:      map[region.DataRegion][]BaseUser{},
			responseFields: map[region.DataRegion]*pagination.ResponseFields{},
		},
	}

	_, err := umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		baseUsers, respFields, err := s.ListBaseUsersPaginated(ctx, p, accessPrimaryDBOnly)

		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		out.listBaseUsersOutput.baseUsers[s.GetRegion()] = baseUsers
		out.listBaseUsersOutput.responseFields[s.GetRegion()] = respFields

		return http.StatusOK, ucerr.Wrap(err)
	}, &out)

	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	mergedUsers := make([]BaseUser, 0)
	var respFields pagination.ResponseFields
	numRegionsWithUsers := 0
	for r, users := range out.listBaseUsersOutput.baseUsers {
		mergedUsers = append(mergedUsers, users...)

		if numRegionsWithUsers == 0 {
			respFields = *out.listBaseUsersOutput.responseFields[r]
		}

		if len(users) > 0 {
			numRegionsWithUsers++
			respFields = *out.listBaseUsersOutput.responseFields[r]
		}
	}

	// if we only got users in zero or one region, then we can return the user list and response fields as is
	if numRegionsWithUsers <= 1 {
		return mergedUsers, &respFields, nil
	}

	// otherwise, we need to sort and limit the users and redo the response fields
	sortKeys := strings.Split(string(p.GetSortKey()), ",")
	for _, s := range sortKeys {
		if s != "id" && s != "organization_id" {
			// we only support sorting by id and organization_id
			return nil, nil, ucerr.Errorf("Cannot sort by key %s", s)
		}
	}
	sort.Slice(mergedUsers, func(i, j int) bool {
		for _, s := range sortKeys {
			switch s {
			case "id":
				if mergedUsers[i].ID != mergedUsers[j].ID {
					return mergedUsers[i].ID.String() < mergedUsers[j].ID.String()
				}
			case "organization_id":
				if mergedUsers[i].OrganizationID != mergedUsers[j].OrganizationID {
					return mergedUsers[i].OrganizationID.String() < mergedUsers[j].OrganizationID.String()
				}
			}
		}
		return mergedUsers[i].ID.String() < mergedUsers[j].ID.String()
	})

	if p.GetSortOrder() == pagination.OrderDescending {
		slices.Reverse(mergedUsers)
	}

	retUsers, respFields := pagination.ProcessResults(mergedUsers, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())

	return retUsers, &respFields, ucerr.Wrap(err)
}

type listUsersForEmailOutput struct {
	users map[region.DataRegion][]BaseUser
}

// ListUsersForEmail returns users from all regions that match the email
func (umrs *UserMultiRegionStorage) ListUsersForEmail(ctx context.Context, configStorage *Storage, email string) ([]BaseUser, error) {
	// TODO: This should be purpose and retention duration aware, right?
	//       Since for now pre-delete retention timeouts are always indefinite,
	//       it is equivalent to the old behavior to just join on the ucv col,
	//       since there will only be a col if there is a valid purpose.
	col, err := configStorage.GetUserColumnByName(ctx, "email")
	if err != nil {
		return nil, ucerr.Friendlyf(err, "failed to get email column")
	}

	dataType, err := configStorage.GetDataType(ctx, col.DataTypeID)
	if err != nil {
		return nil, ucerr.Friendlyf(err, "failed to get email column data type")
	}

	if dataType.ConcreteDataTypeID != datatype.String.ID {
		return nil, ucerr.Friendlyf(nil, "email column is not a string")
	}

	out := runAcrossRegionsOutput{
		mutex: &sync.Mutex{},
		listUsersForEmailOutput: listUsersForEmailOutput{
			users: map[region.DataRegion][]BaseUser{},
		},
	}

	_, err = umrs.runAcrossRegions(ctx, func(ctx context.Context, s *UserStorage, out *runAcrossRegionsOutput) (int, error) {
		users, err := s.listUsersForEmail(ctx, col, email)

		if err != nil {
			return http.StatusInternalServerError, ucerr.Wrap(err)
		}

		if len(users) == 0 {
			return http.StatusOK, nil
		}

		out.mutex.Lock()
		defer out.mutex.Unlock()
		out.listUsersForEmailOutput.users[s.GetRegion()] = users

		return http.StatusOK, nil
	}, &out)

	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	mergedUsers := []BaseUser{}
	for _, users := range out.listUsersForEmailOutput.users {
		mergedUsers = append(mergedUsers, users...)
	}

	return mergedUsers, nil
}
