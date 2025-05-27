package userstore_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	internalUserstore "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func makeCursor(objIDs []uuid.UUID, index int) pagination.Cursor {
	return pagination.Cursor(fmt.Sprintf("id:%v", objIDs[index]))
}

func TestUserRegion(t *testing.T) {
	ctx := context.Background()
	var testReferenceTime = time.Now().UTC()

	tf := idptesthelpers.NewTestFixture(t)
	_, remoteRegionDB := testhelpers.ProvisionTestTenant(ctx, t, tf.CCS, tf.CompanyDBConfig, tf.LogDBConfig, tf.Company.ID)

	userRegionDbMap := map[region.DataRegion]*ucdb.DB{
		"aws-us-east-1": tf.TenantDB,
		"aws-us-west-2": remoteRegionDB,
	}

	ts := tf.TenantState
	ts.UserRegionDbMap = userRegionDbMap
	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoErr(t, err)
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, tf.TenantURL)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwt))
	ctx = request.SetRequestData(multitenant.SetTenantState(ctx, ts), req, uuid.Nil)

	s := storage.NewFromTenantState(tf.Ctx, ts)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	assert.NoErr(t, err)
	columns := cm.GetColumns()
	dtm, err := storage.NewDataTypeManager(ctx, s)
	assert.NoErr(t, err)

	//
	// Basic creation/deletion test
	//

	// create a user w/ no data in default region (us-east-1)
	user, code, err := internalUserstore.CreateUserHelper(
		ctx,
		nil,
		idp.CreateUserWithMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			RowData:        nil,
			OrganizationID: tf.Company.ID,
		},
		false,
	)
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)
	assert.NotNil(t, user)

	// verify that it is in the default region
	us_east_storage := storage.NewUserStorage(ctx, tf.TenantState.UserRegionDbMap["aws-us-east-1"], "aws-us-east-1", tf.TenantState.ID)
	u, _, err := us_east_storage.GetUser(ctx, cm, dtm, user.ID, false)
	assert.NoErr(t, err)
	assert.NotNil(t, u)
	us_west_storage := storage.NewUserStorage(ctx, tf.TenantState.UserRegionDbMap["aws-us-west-2"], "aws-us-west-2", tf.TenantState.ID)
	_, _, err = us_west_storage.GetUser(ctx, cm, dtm, user.ID, false)
	assert.NotNil(t, err)

	// delete the user
	_, err = internalUserstore.DeleteUser(ctx, nil, user.ID)
	assert.NoErr(t, err)

	// verify that it is gone from the default region
	_, _, err = us_east_storage.GetUser(ctx, cm, dtm, user.ID, false)
	assert.NotNil(t, err)

	// create a user w/ no data in us-east-1
	user1, code, err := internalUserstore.CreateUserHelper(
		ctx,
		nil,
		idp.CreateUserWithMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			RowData:        nil,
			OrganizationID: tf.Company.ID,
			Region:         "aws-us-east-1",
		},
		false,
	)
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)
	assert.NotNil(t, user1)

	// create a user w/ email in us-west-2
	user2Email := "joe@userclouds.com"
	user2, code, err := internalUserstore.CreateUserHelper(
		ctx,
		nil,
		idp.CreateUserWithMutatorRequest{
			ID:        uuid.Must(uuid.NewV4()),
			MutatorID: constants.UpdateUserMutatorID,
			Context:   policy.ClientContext{},
			RowData: map[string]idp.ValueAndPurposes{
				"email_verified": {Value: true, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				"email":          {Value: user2Email, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				"picture":        {Value: "", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				"nickname":       {Value: "", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				"name":           {Value: "Joe", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				"external_alias": {Value: "asdf", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
			},
			OrganizationID: tf.Company.ID,
			Region:         "aws-us-west-2",
		},
		true,
	)
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)
	assert.NotNil(t, user2)

	// create a user in non-supported region
	userFail, code, err := internalUserstore.CreateUserHelper(
		ctx,
		nil,
		idp.CreateUserWithMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			RowData:        nil,
			OrganizationID: tf.Company.ID,
			Region:         "aws-eu-west-1",
		},
		false,
	)
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest)
	assert.IsNil(t, userFail)

	// pass in a gibberish region
	userFail, code, err = internalUserstore.CreateUserHelper(
		ctx,
		nil,
		idp.CreateUserWithMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			RowData:        nil,
			OrganizationID: tf.Company.ID,
			Region:         "gibberish-region",
		},
		false,
	)
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest)
	assert.IsNil(t, userFail)

	selectorConfig := userstore.UserSelectorConfig{WhereClause: "ALL"}
	umrs := storage.NewUserMultiRegionStorage(ctx, userRegionDbMap, tf.TenantState.ID)

	// get all users from all regions
	allUsersMap, code, err := umrs.GetUsersForSelector(
		ctx,
		cm,
		dtm,
		testReferenceTime,
		column.DataLifeCycleStateLive,
		columns,
		selectorConfig,
		userstore.UserSelectorValues{},
		set.NewUUIDSet(),
		set.NewUUIDSet(),
		nil,
		false,
	)
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)
	east1Users := allUsersMap["aws-us-east-1"]
	assert.Equal(t, len(east1Users), 1)
	west2Users := allUsersMap["aws-us-west-2"]
	assert.Equal(t, len(west2Users), 1)
	assert.Equal(t, west2Users[0].ID, user2.ID)

	// get users with email "joe@userclouds.com" from all regions
	selectorConfig = userstore.UserSelectorConfig{WhereClause: "{email} = ?"}
	allUsersMap, code, err = umrs.GetUsersForSelector(
		ctx,
		cm,
		dtm,
		testReferenceTime,
		column.DataLifeCycleStateLive,
		columns,
		selectorConfig,
		userstore.UserSelectorValues{user2Email},
		set.NewUUIDSet(),
		set.NewUUIDSet(),
		nil,
		false,
	)
	assert.NoErr(t, err)
	assert.Equal(t, code, http.StatusOK)
	_, ok := allUsersMap["aws-us-east-1"]
	assert.False(t, ok)
	west2Users = allUsersMap["aws-us-west-2"]
	assert.Equal(t, len(west2Users), 1, assert.Must())
	assert.Equal(t, west2Users[0].ID, user2.ID)

	// delete these users before we do pagination test
	_, err = internalUserstore.DeleteUser(ctx, nil, user1.ID)
	assert.NoErr(t, err)
	_, err = internalUserstore.DeleteUser(ctx, nil, user2.ID)
	assert.NoErr(t, err)

	//
	// Pagination test across regions
	//

	uuids := []uuid.UUID{
		uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
		uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000002")),
		uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000003")),
		uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000004")),
		uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000005")),
	}

	emails := []string{
		"a",
		"a",
		"b",
		"b",
		"c",
	}

	names := []string{
		"Joe1",
		"Joe2",
		"Joe3",
		"Joe4",
		"Joe5",
	}

	ext_aliases := []string{
		"asdf1",
		"asdf2",
		"asdf3",
		"asdf4",
		"asdf5",
	}

	// create users in us-east-1 and us-west-2
	for i := range 5 {
		reg := region.DataRegion("aws-us-east-1")
		if i%2 == 0 {
			reg = region.DataRegion("aws-us-west-2")
		}
		_, code, err = internalUserstore.CreateUserHelper(
			ctx,
			nil,
			idp.CreateUserWithMutatorRequest{
				ID:        uuids[i],
				MutatorID: constants.UpdateUserMutatorID,
				Context:   policy.ClientContext{},
				RowData: map[string]idp.ValueAndPurposes{
					"email_verified": {Value: true, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
					"email":          {Value: emails[i], PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
					"picture":        {Value: "", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
					"nickname":       {Value: "", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
					"name":           {Value: names[i], PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
					"external_alias": {Value: ext_aliases[i], PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
				},
				OrganizationID: tf.Company.ID,
				Region:         reg,
			},
			false,
		)
		assert.NoErr(t, err)
		assert.Equal(t, code, http.StatusOK)
	}

	p, err := storage.NewBaseUserPaginatorFromOptions()
	assert.NoErr(t, err)
	baseUsers, _, err := umrs.ListBaseUsersPaginated(ctx, *p, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(baseUsers), 5)

	p, err = storage.NewBaseUserPaginatorFromOptions(pagination.Limit(2))
	assert.NoErr(t, err)
	baseUsers, respFields, err := umrs.ListBaseUsersPaginated(ctx, *p, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(baseUsers), 2)
	assert.Equal(t, baseUsers[0].ID, uuids[0])
	assert.Equal(t, baseUsers[1].ID, uuids[1])
	assert.Equal(t, respFields.HasNext, true)
	assert.Equal(t, respFields.HasPrev, false)
	assert.Equal(t, respFields.Next, makeCursor(uuids, 1))

	p, err = storage.NewBaseUserPaginatorFromOptions(pagination.Limit(2), pagination.StartingAfter(respFields.Next))
	assert.NoErr(t, err)
	baseUsers, respFields, err = umrs.ListBaseUsersPaginated(ctx, *p, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(baseUsers), 2)
	assert.Equal(t, baseUsers[0].ID, uuids[2])
	assert.Equal(t, baseUsers[1].ID, uuids[3])
	assert.Equal(t, respFields.HasNext, true)
	assert.Equal(t, respFields.HasPrev, true)
	assert.Equal(t, respFields.Next, makeCursor(uuids, 3))
	assert.Equal(t, respFields.Prev, makeCursor(uuids, 2))

	p, err = storage.NewBaseUserPaginatorFromOptions(pagination.Limit(2), pagination.StartingAfter(respFields.Next))
	assert.NoErr(t, err)
	baseUsers, respFields, err = umrs.ListBaseUsersPaginated(ctx, *p, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(baseUsers), 1)
	assert.Equal(t, baseUsers[0].ID, uuids[4])
	assert.Equal(t, respFields.HasNext, false)
	assert.Equal(t, respFields.HasPrev, true)
	assert.Equal(t, respFields.Next, pagination.CursorEnd)
	assert.Equal(t, respFields.Prev, makeCursor(uuids, 4))

	p, err = storage.NewBaseUserPaginatorFromOptions(pagination.Limit(2), pagination.EndingBefore(respFields.Prev))
	assert.NoErr(t, err)
	baseUsers, respFields, err = umrs.ListBaseUsersPaginated(ctx, *p, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(baseUsers), 2)
	assert.Equal(t, baseUsers[0].ID, uuids[2])
	assert.Equal(t, baseUsers[1].ID, uuids[3])
	assert.Equal(t, respFields.HasNext, true)
	assert.Equal(t, respFields.HasPrev, true)
	assert.Equal(t, respFields.Next, makeCursor(uuids, 3))
	assert.Equal(t, respFields.Prev, makeCursor(uuids, 2))
}
