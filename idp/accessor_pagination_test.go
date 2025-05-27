package idp_test

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uctypes/timestamp"
)

type accessorPaginationTestFixture struct {
	idptesthelpers.TestFixture

	accessorsByName           map[string]*userstore.Accessor
	columnsByName             map[string]*userstore.Column
	userIDs                   []uuid.UUID
	userIDIndices             map[uuid.UUID]int
	userProfilesByAccessor    map[string][]string
	baseTimestamp             time.Time
	strings                   []string
	uniqueStringPrefix        string
	updatedUniqueStringPrefix string
}

func newAccessorPaginationTestFixture(t *testing.T) accessorPaginationTestFixture {
	t.Helper()

	tf := accessorPaginationTestFixture{
		TestFixture:               *idptesthelpers.NewTestFixture(t),
		accessorsByName:           map[string]*userstore.Accessor{},
		columnsByName:             map[string]*userstore.Column{},
		userIDIndices:             map[uuid.UUID]int{},
		baseTimestamp:             timestamp.Normalize(time.Now().UTC()),
		strings:                   []string{"bar", "baz", "biz", "buz", "foo"},
		uniqueStringPrefix:        uniqueName("string"),
		updatedUniqueStringPrefix: uniqueName("string"),
		userProfilesByAccessor:    map[string][]string{},
	}

	tf.initializeColumns()
	tf.initializeAccessors()
	tf.initializeUsers()

	return tf
}

func (tf accessorPaginationTestFixture) accessorBadRequest(name string, options ...idp.Option) {
	tf.T.Helper()
	_, err := tf.executeAccessor(name, options...)
	assert.HTTPError(tf.T, err, http.StatusBadRequest)
}

func (tf accessorPaginationTestFixture) accessorFail(name string, options ...idp.Option) {
	tf.T.Helper()
	_, err := tf.executeAccessor(name, options...)
	assert.NotNil(tf.T, err)
}

func (tf accessorPaginationTestFixture) accessorSucceed(name string, options ...idp.Option) *idp.ExecuteAccessorResponse {
	tf.T.Helper()
	resp, err := tf.executeAccessor(name, options...)
	assert.NoErr(tf.T, err)
	return resp
}

func (tf *accessorPaginationTestFixture) createAccessor(
	name string,
	dlcs userstore.DataLifeCycleState,
	accessPolicyID uuid.UUID,
	columnNames ...string,
) {
	tf.T.Helper()
	tf.createAccessorWithWhereClause(
		name,
		dlcs,
		accessPolicyID,
		"{id} = ANY (?)",
		columnNames...,
	)
}

func (tf *accessorPaginationTestFixture) createAccessorWithWhereClause(
	name string,
	dlcs userstore.DataLifeCycleState,
	accessPolicyID uuid.UUID,
	whereClause string,
	columnNames ...string,
) {
	tf.T.Helper()
	_, found := tf.accessorsByName[name]
	assert.False(tf.T, found)

	accessor := userstore.Accessor{
		Name:               uniqueName(name),
		DataLifeCycleState: dlcs,
		AccessPolicy:       userstore.ResourceID{ID: accessPolicyID},
		SelectorConfig:     userstore.UserSelectorConfig{WhereClause: whereClause},
		Purposes:           []userstore.ResourceID{{ID: constants.OperationalPurposeID}},
	}

	assert.True(tf.T, len(columnNames) > 0)
	for _, columnName := range columnNames {
		accessor.Columns = append(
			accessor.Columns,
			userstore.ColumnOutputConfig{
				Column:      tf.getColumnResource(columnName),
				Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
			},
		)
	}

	createdAccessor, err := tf.IDPClient.CreateAccessor(tf.Ctx, accessor)
	assert.NoErr(tf.T, err)
	tf.accessorsByName[name] = createdAccessor
}

func (tf *accessorPaginationTestFixture) createColumn(name string, dataType userstore.ResourceID, options ...string) {
	tf.T.Helper()
	_, found := tf.columnsByName[name]
	assert.False(tf.T, found)

	isArray := false
	indexType := userstore.ColumnIndexTypeIndexed
	for _, option := range options {
		if option == "is_array" {
			isArray = true
		}
		if option == "unique" {
			indexType = userstore.ColumnIndexTypeUnique
		}
	}

	col := tf.CreateValidColumn(
		uniqueName(name),
		dataType,
		isArray,
		"",
		userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
		userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
		indexType,
	)
	tf.columnsByName[name] = col
}

func (tf accessorPaginationTestFixture) executeAccessor(
	name string,
	additionalOptions ...idp.Option,
) (*idp.ExecuteAccessorResponse, error) {
	tf.T.Helper()

	accessor, found := tf.accessorsByName[name]
	assert.True(tf.T, found)

	options := []idp.Option{idp.Pagination(pagination.Limit(10))}
	options = append(options, additionalOptions...)
	resp, err := tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		accessor.ID,
		policy.ClientContext{},
		[]any{tf.userIDs},
		options...,
	)

	return resp, err
}

func (tf accessorPaginationTestFixture) getBoolean(id uuid.UUID) bool {
	tf.T.Helper()

	index, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return (index%2 == 0)
}

func (tf accessorPaginationTestFixture) getColumn(name string) *userstore.Column {
	tf.T.Helper()

	col, found := tf.columnsByName[name]
	assert.True(tf.T, found)
	return col
}

func (tf accessorPaginationTestFixture) getColumnResource(name string) userstore.ResourceID {
	tf.T.Helper()

	return userstore.ResourceID{ID: tf.getColumn(name).ID}
}

func (tf accessorPaginationTestFixture) getSSN(id uuid.UUID) string {
	tf.T.Helper()

	index, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return fmt.Sprintf("333-33-33%02d", index)
}

func (tf accessorPaginationTestFixture) getString(id uuid.UUID) string {
	tf.T.Helper()

	index, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return tf.strings[index%len(tf.strings)]
}

func (tf accessorPaginationTestFixture) getTimestamp(id uuid.UUID) time.Time {
	tf.T.Helper()

	index, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return tf.baseTimestamp.Add(time.Second * time.Duration(index))
}

func (tf accessorPaginationTestFixture) getUniqueString(id uuid.UUID) string {
	tf.T.Helper()

	_, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return fmt.Sprintf("%s_%v", tf.uniqueStringPrefix, id)
}

func (tf accessorPaginationTestFixture) getUpdatedUniqueString(id uuid.UUID) string {
	tf.T.Helper()

	_, found := tf.userIDIndices[id]
	assert.True(tf.T, found)
	return fmt.Sprintf("%s_%v", tf.updatedUniqueStringPrefix, id)
}

func (tf *accessorPaginationTestFixture) initializeAccessors() {
	tf.createAccessor(
		"system",
		userstore.DataLifeCycleStateLive,
		policy.AccessPolicyAllowAll.ID,
		"organization_id",
		"updated",
	)

	tf.createAccessor(
		"systemDenyAll",
		userstore.DataLifeCycleStateLive,
		policy.AccessPolicyDenyAll.ID,
		"organization_id",
		"updated",
	)

	tf.createAccessor(
		"nonSysLive",
		userstore.DataLifeCycleStateLive,
		policy.AccessPolicyAllowAll.ID,
		"organization_id",
		"address",
		"boolean",
		"integer",
		"ssn",
		"string",
		"strings",
		"timestamp",
		"uuid",
		"uniqueInteger",
		"uniqueString",
		"uniqueUUID",
	)

	tf.createAccessor(
		"nonSysLiveDenyAll",
		userstore.DataLifeCycleStateLive,
		policy.AccessPolicyDenyAll.ID,
		"organization_id",
		"address",
		"boolean",
		"integer",
		"ssn",
		"string",
		"strings",
		"timestamp",
		"uuid",
		"uniqueInteger",
		"uniqueString",
		"uniqueUUID",
	)

	tf.createAccessor(
		"nonSysDead",
		userstore.DataLifeCycleStateSoftDeleted,
		policy.AccessPolicyAllowAll.ID,
		"organization_id",
		"address",
		"boolean",
		"integer",
		"ssn",
		"string",
		"strings",
		"timestamp",
		"uuid",
		"uniqueInteger",
		"uniqueString",
		"uniqueUUID",
	)

	tf.createAccessor(
		"nonSysDeadDenyAll",
		userstore.DataLifeCycleStateSoftDeleted,
		policy.AccessPolicyDenyAll.ID,
		"organization_id",
		"address",
		"boolean",
		"integer",
		"ssn",
		"string",
		"strings",
		"timestamp",
		"uuid",
		"uniqueInteger",
		"uniqueString",
		"uniqueUUID",
	)
}

func (tf *accessorPaginationTestFixture) initializeColumns() {
	tf.T.Helper()

	for _, dc := range defaults.GetDefaultColumns() {
		c, err := tf.IDPClient.GetColumn(tf.Ctx, dc.ID)
		assert.NoErr(tf.T, err)
		tf.columnsByName[c.Name] = c
	}

	tf.createColumn("address", datatype.CanonicalAddress)
	tf.createColumn("boolean", datatype.Boolean)
	tf.createColumn("integer", datatype.Integer)
	tf.createColumn("ssn", datatype.SSN)
	tf.createColumn("string", datatype.String)
	tf.createColumn("strings", datatype.String, "is_array")
	tf.createColumn("timestamp", datatype.Timestamp)
	tf.createColumn("uniqueInteger", datatype.Integer, "unique")
	tf.createColumn("uniqueString", datatype.String, "unique")
	tf.createColumn("uniqueUUID", datatype.UUID, "unique")
	tf.createColumn("uuid", datatype.UUID)

	_, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(
		tf.Ctx,
		userstore.DataLifeCycleStateSoftDeleted,
		tf.getColumn("uniqueInteger").ID,
		idp.UpdateColumnRetentionDurationsRequest{
			RetentionDurations: []idp.ColumnRetentionDuration{
				{
					ColumnID:     tf.getColumn("uniqueInteger").ID,
					PurposeID:    constants.OperationalPurposeID,
					DurationType: userstore.DataLifeCycleStateSoftDeleted,
					Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
				},
			},
		},
	)
	assert.NoErr(tf.T, err)

	_, err = tf.IDPClient.UpdateColumnRetentionDurationsForColumn(
		tf.Ctx,
		userstore.DataLifeCycleStateSoftDeleted,
		tf.getColumn("uniqueString").ID,
		idp.UpdateColumnRetentionDurationsRequest{
			RetentionDurations: []idp.ColumnRetentionDuration{
				{
					ColumnID:     tf.getColumn("uniqueString").ID,
					PurposeID:    constants.OperationalPurposeID,
					DurationType: userstore.DataLifeCycleStateSoftDeleted,
					Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
				},
			},
		},
	)
	assert.NoErr(tf.T, err)
}

func (tf *accessorPaginationTestFixture) initializeUsers() {
	tf.T.Helper()

	userIDs := make([]uuid.UUID, 0, 35)
	for range 35 {
		userID := uuid.Must(uuid.NewV4())
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i int, j int) bool {
		return userIDs[i].String() < userIDs[j].String()
	})

	tf.userIDs = userIDs
	for i, userID := range tf.userIDs {
		tf.userIDIndices[userID] = i

		// create user

		profile := userstore.Record{
			"external_alias":                   tf.getUniqueString(userID),
			tf.getColumn("boolean").Name:       tf.getBoolean(userID),
			tf.getColumn("strings").Name:       tf.strings,
			tf.getColumn("string").Name:        tf.getString(userID),
			tf.getColumn("uniqueInteger").Name: i,
			tf.getColumn("uniqueString").Name:  tf.getUniqueString(userID),
			tf.getColumn("uuid").Name:          userID,
		}

		if i%15 != 0 {
			profile[tf.getColumn("integer").Name] = i
			profile[tf.getColumn("ssn").Name] = tf.getSSN(userID)
			profile[tf.getColumn("timestamp").Name] = tf.getTimestamp(userID)
			profile[tf.getColumn("uniqueUUID").Name] = userID
		}

		createdUserID, err := tf.IDPClient.CreateUser(
			tf.Ctx,
			profile,
			idp.UserID(userID),
			idp.OrganizationID(tf.Company.ID),
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, createdUserID, userID)

		// update user after changing a few values for retained columns

		profile[tf.getColumn("uniqueString").Name] = tf.getUpdatedUniqueString(userID)
		profile[tf.getColumn("uniqueInteger").Name] = i + 50

		resp, err := tf.IDPClient.UpdateUser(
			tf.Ctx,
			userID,
			idp.UpdateUserRequest{Profile: profile},
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, resp.ID, userID)

		// record expected values

		for accessorName, accessor := range tf.accessorsByName {
			if accessor.AccessPolicy.ID == policy.AccessPolicyDenyAll.ID {
				continue
			}

			resp, err := tf.IDPClient.ExecuteAccessor(
				tf.Ctx,
				accessor.ID,
				policy.ClientContext{},
				[]any{[]uuid.UUID{userID}},
			)
			assert.NoErr(tf.T, err)
			assert.Equal(tf.T, len(resp.Data), 1)
			userProfiles := tf.userProfilesByAccessor[accessorName]
			userProfiles = append(userProfiles, resp.Data[0])
			tf.userProfilesByAccessor[accessorName] = userProfiles
		}
	}
}

func (tf accessorPaginationTestFixture) sortBy(sortKeys ...string) idp.Option {
	joinedSortKey := ""

	if len(sortKeys) > 0 {
		var sortColumnNames []string
		for _, sortKey := range sortKeys {
			colName := sortKey
			if col, found := tf.columnsByName[sortKey]; found {
				colName = col.Name
			}
			sortColumnNames = append(sortColumnNames, colName)
		}
		joinedSortKey = strings.Join(sortColumnNames, ",")
	}

	return idp.Pagination(pagination.SortKey(pagination.Key(joinedSortKey)))
}

func (tf accessorPaginationTestFixture) validateResults(
	accessorName string,
	resp *idp.ExecuteAccessorResponse,
	indices ...int,
) {
	tf.T.Helper()

	userProfiles, found := tf.userProfilesByAccessor[accessorName]
	assert.True(tf.T, found)
	assert.Equal(tf.T, len(resp.Data), len(indices))
	for i, index := range indices {
		assert.True(tf.T, index < len(userProfiles))
		assert.Equal(tf.T, resp.Data[i], userProfiles[index])
	}
}

func TestAccessorPagination(t *testing.T) {
	t.Parallel()
	tf := newAccessorPaginationTestFixture(t)

	t.Run("test_accessor_non_unique_like_hint", func(t *testing.T) {
		tf.createAccessorWithWhereClause(
			"like_non_unique_string",
			userstore.DataLifeCycleStateLive,
			policy.AccessPolicyAllowAll.ID,
			fmt.Sprintf("{%s} LIKE ?", tf.columnsByName["string"].Name),
			"uuid",
			"string",
		)

		_, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.accessorsByName["like_non_unique_string"].ID,
			policy.ClientContext{},
			[]any{"%foo%"},
		)
		assert.NoErr(tf.T, err)
	})

	t.Run("test_accessor_non_unique_ilike_hint", func(t *testing.T) {
		tf.createAccessorWithWhereClause(
			"ilike_non_unique_string",
			userstore.DataLifeCycleStateLive,
			policy.AccessPolicyAllowAll.ID,
			fmt.Sprintf("{%s} ILIKE ?", tf.columnsByName["string"].Name),
			"uuid",
			"string",
		)

		_, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.accessorsByName["ilike_non_unique_string"].ID,
			policy.ClientContext{},
			[]any{"%foo%"},
		)
		assert.NoErr(tf.T, err)
	})

	t.Run("test_accessor_unique_like_hint", func(t *testing.T) {
		tf.createAccessorWithWhereClause(
			"like_unique_string",
			userstore.DataLifeCycleStateLive,
			policy.AccessPolicyAllowAll.ID,
			fmt.Sprintf("{%s} LIKE ?", tf.columnsByName["uniqueString"].Name),
			"uuid",
			"uniqueString",
		)

		_, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.accessorsByName["like_unique_string"].ID,
			policy.ClientContext{},
			[]any{"%foo%"},
		)
		assert.NoErr(tf.T, err)
	})

	t.Run("test_accessor_unique_ilike_hint", func(t *testing.T) {
		tf.createAccessorWithWhereClause(
			"ilike_unique_string",
			userstore.DataLifeCycleStateLive,
			policy.AccessPolicyAllowAll.ID,
			fmt.Sprintf("{%s} ILIKE ?", tf.columnsByName["uniqueString"].Name),
			"uuid",
			"uniqueString",
		)

		_, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.accessorsByName["ilike_unique_string"].ID,
			policy.ClientContext{},
			[]any{"%foo%"},
		)
		assert.NoErr(tf.T, err)
	})

	t.Run("test_sort_key_combinations", func(t *testing.T) {
		//// failures

		// empty sort key
		tf.accessorFail("system", tf.sortBy(""))
		tf.accessorFail("nonSysLive", tf.sortBy(""))
		tf.accessorFail("nonSysDead", tf.sortBy(""))

		// id not included
		tf.accessorBadRequest("system", tf.sortBy("organization_id"))
		tf.accessorBadRequest("nonSysLive", tf.sortBy("string", "integer"))
		tf.accessorBadRequest("nonSysDead", tf.sortBy("organization_id"))

		// id in wrong location
		tf.accessorBadRequest("system", tf.sortBy("id", "string", "integer"))
		tf.accessorBadRequest("nonSysLive", tf.sortBy("id", "string", "integer"))
		tf.accessorBadRequest("nonSysDead", tf.sortBy("id", "organization_id"))

		// unrecognized sort key
		tf.accessorBadRequest("system", tf.sortBy("foo", "id"))
		tf.accessorBadRequest("nonSysLive", tf.sortBy("foo", "id"))
		tf.accessorBadRequest("nonSysDead", tf.sortBy("foo", "id"))

		// supported sort key type not returned by accessor
		tf.accessorBadRequest("system", tf.sortBy("version", "id"))
		tf.accessorBadRequest("nonSysLive", tf.sortBy("name", "id"))
		tf.accessorBadRequest("nonSysDead", tf.sortBy("version", "id"))

		// soft-deleted accessor does not support non-system sort keys
		tf.accessorBadRequest("nonSysDead", tf.sortBy("ssn", "integer", "id"))

		// non-system sort keys must be of supported type
		tf.accessorBadRequest("nonSysLive", tf.sortBy("address", "id"))
		tf.accessorBadRequest("nonSysLive", tf.sortBy("strings", "id"))

		//// successes

		// issue request including only system sort keys
		tf.accessorSucceed("system", tf.sortBy("created", "organization_id", "id"))
		tf.accessorSucceed("nonSysLive", tf.sortBy("created", "organization_id", "id"))
		tf.accessorSucceed("nonSysDead", tf.sortBy("created", "organization_id", "id"))

		// issue live accessor request including non-system sort keys
		tf.accessorSucceed("nonSysLive", tf.sortBy("string", "uuid", "id"))

		// issue live accessor request including mixture of system and non-system sort keys
		tf.accessorSucceed("nonSysLive", tf.sortBy("updated", "uniqueString", "id"))
	})

	t.Run("test_single_key_non_sys_live_ascending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed("nonSysLive")
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

		resp = tf.accessorSucceed(
			"nonSysLive",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)

		resp = tf.accessorSucceed(
			"nonSysLive",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29)

		resp = tf.accessorSucceed(
			"nonSysLive",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 30, 31, 32, 33, 34)
	})

	t.Run("test_multi_key_non_sys_live_ascending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 15, 5, 25, 0, 30, 10, 20, 1, 11, 21)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 31, 6, 16, 26, 7, 17, 27, 2, 12, 22)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 32, 3, 13, 23, 33, 8, 18, 28, 9, 19)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 31, 6, 16, 26, 7, 17, 27, 2, 12, 22)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 32, 3, 13, 23, 33, 8, 18, 28, 9, 19)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "integer", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 29, 4, 14, 24, 34)
	})

	t.Run("test_multi_key_non_sys_live_ascending_from_end", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "timestamp", "id"),
			idp.Pagination(pagination.EndingBefore(pagination.CursorEnd)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 8, 18, 28, 9, 19, 29, 4, 14, 24, 34)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "timestamp", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 17, 27, 2, 12, 22, 32, 3, 13, 23, 33)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "timestamp", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 10, 20, 1, 11, 21, 31, 6, 16, 26, 7)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "timestamp", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 15, 5, 25, 0, 30)
	})

	t.Run("test_multi_key_non_sys_live_descending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "uniqueUUID", "id"),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 34, 24, 14, 4, 29, 19, 9, 28, 18, 8)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "uniqueUUID", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 33, 23, 13, 3, 32, 22, 12, 2, 27, 17)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "uniqueUUID", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 7, 26, 16, 6, 31, 21, 11, 1, 20, 10)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "uniqueUUID", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 30, 0, 25, 5, 15)
	})

	t.Run("test_multi_key_non_sys_live_descending_from_end", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "ssn", "id"),
			idp.Pagination(pagination.EndingBefore(pagination.CursorEnd)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 21, 11, 1, 20, 10, 30, 0, 25, 5, 15)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "ssn", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 22, 12, 2, 27, 17, 7, 26, 16, 6, 31)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "ssn", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 19, 9, 28, 18, 8, 33, 23, 13, 3, 32)

		resp = tf.accessorSucceed(
			"nonSysLive",
			tf.sortBy("string", "boolean", "ssn", "id"),
			idp.Pagination(pagination.EndingBefore(resp.Prev)),
			idp.Pagination(pagination.SortOrder(pagination.OrderDescending)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysLive", resp, 34, 24, 14, 4, 29)
	})

	t.Run("test_multi_key_sys_ascending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"system",
			tf.sortBy("organization_id", "id"),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("system", resp, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

		resp = tf.accessorSucceed(
			"system",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("system", resp, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)

		resp = tf.accessorSucceed(
			"system",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("system", resp, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29)

		resp = tf.accessorSucceed(
			"system",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("system", resp, 30, 31, 32, 33, 34)
	})

	t.Run("test_single_key_non_sys_dead_ascending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysDead",
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

		resp = tf.accessorSucceed(
			"nonSysDead",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)

		resp = tf.accessorSucceed(
			"nonSysDead",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29)

		resp = tf.accessorSucceed(
			"nonSysDead",
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 30, 31, 32, 33, 34)
	})

	t.Run("test_multi_key_non_sys_dead_ascending_from_start", func(t *testing.T) {
		resp := tf.accessorSucceed(
			"nonSysDead",
			tf.sortBy("organization_id", "id"),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.False(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

		resp = tf.accessorSucceed(
			"nonSysDead",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)

		resp = tf.accessorSucceed(
			"nonSysDead",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 10)
		assert.True(tf.T, resp.HasPrev)
		assert.True(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29)

		resp = tf.accessorSucceed(
			"nonSysDead",
			tf.sortBy("organization_id", "id"),
			idp.Pagination(pagination.StartingAfter(resp.Next)),
		)
		assert.Equal(tf.T, len(resp.Data), 5)
		assert.True(tf.T, resp.HasPrev)
		assert.False(tf.T, resp.HasNext)
		tf.validateResults("nonSysDead", resp, 30, 31, 32, 33, 34)
	})

	t.Run("test_deny_all_access_policy", func(t *testing.T) {
		for _, accessorName := range []string{"systemDenyAll", "nonSysLiveDenyAll", "nonSysDeadDenyAll"} {
			resp := tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.False(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[9]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.StartingAfter(resp.Next)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.True(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Prev), fmt.Sprintf("id:%v", tf.userIDs[10]))
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[19]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.StartingAfter(resp.Next)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.True(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Prev), fmt.Sprintf("id:%v", tf.userIDs[20]))
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[29]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.StartingAfter(resp.Next)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.True(tf.T, resp.HasPrev)
			assert.False(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Prev), fmt.Sprintf("id:%v", tf.userIDs[30]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.EndingBefore(resp.Prev)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.True(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Prev), fmt.Sprintf("id:%v", tf.userIDs[20]))
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[29]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.EndingBefore(resp.Prev)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.True(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Prev), fmt.Sprintf("id:%v", tf.userIDs[10]))
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[19]))

			resp = tf.accessorSucceed(
				accessorName,
				idp.Pagination(pagination.Limit(1)),
				idp.Pagination(pagination.EndingBefore(resp.Prev)),
			)
			assert.Equal(tf.T, len(resp.Data), 0)
			assert.False(tf.T, resp.HasPrev)
			assert.True(tf.T, resp.HasNext)
			assert.Equal(tf.T, string(resp.Next), fmt.Sprintf("id:%v", tf.userIDs[9]))
		}
	})
}
