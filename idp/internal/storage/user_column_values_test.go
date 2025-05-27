package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uctypes/set"
)

var testReferenceTime = time.Now().UTC()
var testNonExpiredRetention = testReferenceTime.Add(time.Duration(1) * time.Hour)

//var testIndefiniteRetention time.Time
//var testExpiredRetention = testReferenceTime.Add(time.Duration(-1) * time.Hour)

type userstoreTestFixture struct {
	ctx            context.Context
	t              *testing.T
	tdb            *ucdb.DB
	s              *storage.Storage
	us             *storage.UserStorage
	cm             *storage.ColumnManager
	dtm            *storage.DataTypeManager
	organizationID uuid.UUID
}

func newUserstoreTestFixture(t *testing.T) userstoreTestFixture {
	t.Helper()

	stf := newStorageForTests(t)
	us := storage.NewUserStorage(stf.ctx, stf.db, "", stf.tenant.ID)

	cm, err := storage.NewUserstoreColumnManager(stf.ctx, stf.s)
	assert.NoErr(t, err)

	dtm, err := storage.NewDataTypeManager(stf.ctx, stf.s)
	assert.NoErr(t, err)

	return userstoreTestFixture{
		ctx:            stf.ctx,
		t:              t,
		tdb:            stf.db,
		s:              stf.s,
		us:             us,
		cm:             cm,
		dtm:            dtm,
		organizationID: stf.tenant.CompanyID,
	}
}

type userColumnValueTestFixture struct {
	userstoreTestFixture
	user          *storage.User
	purposes      map[string]*storage.Purpose
	systemColumns []storage.Column
}

func (tf *userColumnValueTestFixture) addColumn(cb *columnBuilder, name string) {
	tf.t.Helper()

	c := tf.cm.GetUserColumnByName(name)
	assert.IsNil(tf.t, c)
	cb.setName(name)
	c = cb.build()
	_, err := tf.cm.SaveColumn(tf.ctx, c)
	assert.NoErr(tf.t, err)
}

func (tf *userColumnValueTestFixture) addPurpose(name string, description string) {
	tf.t.Helper()

	_, found := tf.purposes[name]
	assert.False(tf.t, found)
	tf.purposes[name] = &storage.Purpose{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		Description:              description,
	}
}

func (tf *userColumnValueTestFixture) startUser() *userColumnValueBuilder {
	return newUserColumnValueBuilder(tf.user)
}

func newUserColumnValueTestFixture(t *testing.T) *userColumnValueTestFixture {
	t.Helper()

	tf := userColumnValueTestFixture{
		userstoreTestFixture: newUserstoreTestFixture(t),
		purposes:             map[string]*storage.Purpose{},
	}
	for _, c := range tf.cm.GetColumns() {
		if c.Attributes.System {
			tf.systemColumns = append(tf.systemColumns, c)
		}
	}

	cb := newColumnBuilder()

	tf.addColumn(cb.setType(datatype.String.ID).setDefaultValue("default_string").setIndexType(userstore.ColumnIndexTypeNone), "string")
	tf.addColumn(cb.setType(datatype.Boolean.ID).setDefaultValue("true"), "bool")
	tf.addColumn(cb.setType(datatype.Integer.ID).setDefaultValue("42"), "int")
	tf.addColumn(cb.setType(datatype.Timestamp.ID), "timestamp")
	tf.addColumn(cb.setType(datatype.UUID.ID), "uuid")
	tf.addColumn(cb.makeArray().setType(datatype.String.ID), "string_array")
	tf.addColumn(cb.makeArray().setType(datatype.Boolean.ID), "bool_array")
	tf.addColumn(cb.makeArray().setType(datatype.Integer.ID), "int_array")
	tf.addColumn(cb.makeArray().setType(datatype.Timestamp.ID), "timestamp_array")
	tf.addColumn(cb.makeArray().setType(datatype.UUID.ID), "uuid_array")
	tf.addColumn(cb.setType(datatype.String.ID).setIndexType(userstore.ColumnIndexTypeUnique), "string_unique")
	tf.addColumn(cb.setType(datatype.Integer.ID).setIndexType(userstore.ColumnIndexTypeUnique), "int_unique")
	tf.addColumn(cb.setType(datatype.UUID.ID).setIndexType(userstore.ColumnIndexTypeUnique), "uuid_unique")

	tf.addPurpose("marketing", "for marketing")
	tf.addPurpose("operational", "for operations")
	tf.addPurpose("security", "for security")

	tf.user = &storage.User{
		BaseUser: storage.BaseUser{
			VersionBaseModel: ucdb.NewVersionBase(),
			OrganizationID:   tf.organizationID,
		},
	}
	assert.NoErr(tf.t, tf.us.SaveUser(tf.ctx, tf.user))

	return &tf
}

func makeTestID() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}

type columnBuilder struct {
	column *storage.Column
}

func newColumnBuilder() *columnBuilder {
	cb := &columnBuilder{}
	return cb.newColumn()
}

func (cb *columnBuilder) newColumn() *columnBuilder {
	cb.column = &storage.Column{
		BaseModel:            ucdb.NewBase(),
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	}
	return cb
}

func (cb *columnBuilder) build() *storage.Column {
	c := cb.column
	cb.newColumn()
	return c
}

func (cb *columnBuilder) makeArray() *columnBuilder {
	cb.column.IsArray = true
	return cb
}

func (cb *columnBuilder) setDefaultValue(defaultValue string) *columnBuilder {
	cb.column.DefaultValue = defaultValue
	return cb
}

func (cb *columnBuilder) setIndexType(t userstore.ColumnIndexType) *columnBuilder {
	cb.column.IndexType = storage.ColumnIndexTypeFromClient(t)
	return cb
}

func (cb *columnBuilder) setName(name string) *columnBuilder {
	cb.column.Name = name
	return cb
}

func (cb *columnBuilder) setType(dataTypeID uuid.UUID) *columnBuilder {
	cb.column.DataTypeID = dataTypeID
	return cb
}

type userColumnValueBuilder struct {
	liveUCV        *storage.UserColumnLiveValue
	softDeletedUCV *storage.UserColumnSoftDeletedValue
}

func newUserColumnValueBuilder(u *storage.User) *userColumnValueBuilder {
	return &userColumnValueBuilder{
		liveUCV: &storage.UserColumnLiveValue{
			BaseUserColumnValue: storage.BaseUserColumnValue{
				VersionBaseModel: ucdb.NewVersionBase(),
				UserID:           u.ID,
				IsNew:            true,
			},
		},
		softDeletedUCV: &storage.UserColumnSoftDeletedValue{
			BaseUserColumnValue: storage.BaseUserColumnValue{
				VersionBaseModel: ucdb.NewVersionBase(),
				UserID:           u.ID,
				IsNew:            true,
			},
		},
	}
}

func (b *userColumnValueBuilder) liveValue() *storage.UserColumnLiveValue {
	return b.liveUCV
}

func (b *userColumnValueBuilder) softDeletedValue() *storage.UserColumnSoftDeletedValue {
	return b.softDeletedUCV
}

func (b *userColumnValueBuilder) setColumn(c *storage.Column) *userColumnValueBuilder {
	b.liveUCV.ColumnID = c.ID
	b.liveUCV.Column = c
	b.softDeletedUCV.ColumnID = c.ID
	b.softDeletedUCV.Column = c
	return b
}

func (b *userColumnValueBuilder) withPurpose(p *storage.Purpose, retentionTimeout time.Time) *userColumnValueBuilder {
	b.liveUCV.ConsentedPurposeIDs = append(b.liveUCV.ConsentedPurposeIDs, p.ID)
	b.liveUCV.RetentionTimeouts = append(b.liveUCV.RetentionTimeouts, retentionTimeout)
	b.softDeletedUCV.ConsentedPurposeIDs = append(b.softDeletedUCV.ConsentedPurposeIDs, p.ID)
	b.softDeletedUCV.RetentionTimeouts = append(b.softDeletedUCV.RetentionTimeouts, retentionTimeout)
	return b
}

func (b *userColumnValueBuilder) setOrdering(ordering int) *userColumnValueBuilder {
	b.liveUCV.Ordering = ordering
	b.softDeletedUCV.Ordering = ordering
	return b
}

func (b *userColumnValueBuilder) setBool(v bool) *userColumnValueBuilder {
	b.liveUCV.BooleanValue = &v
	b.softDeletedUCV.BooleanValue = &v
	return b
}

func (b *userColumnValueBuilder) setInt(v int) *userColumnValueBuilder {
	b.liveUCV.IntValue = &v
	b.softDeletedUCV.IntValue = &v
	return b
}

func (b *userColumnValueBuilder) setString(v string) *userColumnValueBuilder {
	b.liveUCV.VarcharValue = &v
	b.softDeletedUCV.VarcharValue = &v
	return b
}

func (b *userColumnValueBuilder) setTimestamp(v time.Time) *userColumnValueBuilder {
	b.liveUCV.TimestampValue = &v
	b.softDeletedUCV.TimestampValue = &v
	return b
}

func (b *userColumnValueBuilder) setUniqueInt(v int) *userColumnValueBuilder {
	b.liveUCV.IntUniqueValue = &v
	b.softDeletedUCV.IntValue = &v
	return b
}

func (b *userColumnValueBuilder) setUniqueString(v string) *userColumnValueBuilder {
	b.liveUCV.VarcharUniqueValue = &v
	b.softDeletedUCV.VarcharValue = &v
	return b
}

func (b *userColumnValueBuilder) setUniqueUUID(v uuid.UUID) *userColumnValueBuilder {
	b.liveUCV.UUIDUniqueValue = &v
	b.softDeletedUCV.UUIDValue = &v
	return b
}

func (b *userColumnValueBuilder) setUUID(v uuid.UUID) *userColumnValueBuilder {
	b.liveUCV.UUIDValue = &v
	b.softDeletedUCV.UUIDValue = &v
	return b
}

func TestUserColumnValues(t *testing.T) {
	t.Parallel()

	tf := newUserColumnValueTestFixture(t)

	testSaveAndGet(tf, tf.startUser().setBool(true), "bool", "'true'", "'true' , 'false'::BOOL")

	testSaveAndGet(tf, tf.startUser().setBool(true), "bool_array")

	testSaveAndGet(tf, tf.startUser().setInt(42), "int", "42", "42 , 40")

	testSaveAndGet(tf, tf.startUser().setInt(42), "int_array")

	testSaveAndGet(tf, tf.startUser().setUniqueInt(42), "int_unique", "42", "'42' , '40'::INT")

	testSaveAndGet(tf, tf.startUser().setString("hello"), "string", "'hello'", "'hello' , 'bar'::VARCHAR")

	testSaveAndGet(tf, tf.startUser().setString("hello"), "string_array")

	testSaveAndGet(tf, tf.startUser().setUniqueString("hello"), "string_unique", "'hello'", "'hello' , 'bar'::VARCHAR")

	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	testSaveAndGet(tf, tf.startUser().setTimestamp(testTime), "timestamp", "'2023-01-01'::TIMESTAMP", "'2023-01-01'::TIMESTAMP , '2022-01-01'::TIMESTAMP")

	testSaveAndGet(tf, tf.startUser().setTimestamp(testReferenceTime), "timestamp_array")

	testID := makeTestID()
	testSaveAndGet(tf, tf.startUser().setUUID(testID), "uuid", fmt.Sprintf("'%s'", testID), fmt.Sprintf("'%s' , '%s'::UUID", testID, uuid.Nil))

	testSaveAndGet(tf, tf.startUser().setUUID(makeTestID()), "uuid_array")

	testID = makeTestID()
	testSaveAndGet(tf, tf.startUser().setUniqueUUID(testID), "uuid_unique", fmt.Sprintf("'%s'", testID), fmt.Sprintf("'%s' , '%s'::UUID", testID, uuid.Nil))
}

func testSaveAndGet(tf *userColumnValueTestFixture, ucvb *userColumnValueBuilder, columnName string, testValues ...string) {
	tf.t.Helper()

	c := tf.cm.GetUserColumnByName(columnName)
	assert.NotNil(tf.t, c)
	ucvb.setColumn(c).setOrdering(1)

	purposeIDs := set.NewUUIDSet()
	for _, name := range []string{"marketing", "operational", "security"} {
		purpose, found := tf.purposes[name]
		assert.True(tf.t, found)
		purposeIDs.Insert(purpose.ID)
		ucvb.withPurpose(purpose, testNonExpiredRetention)
	}

	cm, err := storage.NewUserstoreColumnManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)
	sim, err := storage.NewSearchIndexManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)

	liveVal := ucvb.liveValue()
	assert.NoErr(tf.t, liveVal.Validate())
	liveVals := []storage.UserColumnLiveValue{*liveVal}
	assert.NoErr(tf.t, tf.us.InsertUserColumnLiveValues(tf.ctx, cm, sim, nil, liveVals))

	softDeletedVal := ucvb.softDeletedValue()
	assert.NoErr(tf.t, softDeletedVal.Validate())
	softDeletedVals := []storage.UserColumnSoftDeletedValue{*softDeletedVal}
	assert.NoErr(tf.t, tf.us.InsertUserColumnSoftDeletedValues(tf.ctx, softDeletedVals))

	validateGetUsers(tf, column.DataLifeCycleStateLive, c, purposeIDs)
	validateGetUsers(tf, column.DataLifeCycleStateSoftDeleted, c, purposeIDs)

	validateSelectByValue(tf, c, purposeIDs, testValues...)
}

func validateSelectByValue(tf *userColumnValueTestFixture, c *storage.Column, purposeIDs set.Set[uuid.UUID], testValues ...string) {
	tf.t.Helper()

	totalTestValues := len(testValues)
	assert.True(tf.t, totalTestValues == 0 || totalTestValues == 2)
	if totalTestValues == 0 {
		return
	}

	selectorVariations := []string{`ANY(ARRAY[%s])`, `ANY ( ARRAY [ %s ] )`}

	selectorClauses := []string{fmt.Sprintf("{%s} = %s", c.Name, testValues[0])}
	var selectorValues userstore.UserSelectorValues
	var columns []storage.Column
	columns = append(columns, *c)

	for _, sv := range selectorVariations {
		selectorFormat := fmt.Sprintf("{%s} = %s", c.Name, sv)
		selectorClauses = append(selectorClauses, fmt.Sprintf(selectorFormat, testValues[1]))
	}

	cm, err := storage.NewUserstoreColumnManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)
	dtm, err := storage.NewDataTypeManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)

	for _, selectorClause := range selectorClauses {
		selectorConfig := userstore.UserSelectorConfig{WhereClause: selectorClause}
		users, code, err := tf.us.GetUsersForSelector(tf.ctx, cm, dtm, testReferenceTime, column.DataLifeCycleStateLive, columns, selectorConfig, selectorValues, set.NewUUIDSet(), purposeIDs, nil, false)
		assert.NoErr(tf.t, err)
		assert.Equal(tf.t, code, http.StatusOK)
		assert.Equal(tf.t, len(users), 1)
	}
}

func validateGetUsers(tf *userColumnValueTestFixture, dlcs column.DataLifeCycleState, column *storage.Column, purposeIDs set.Set[uuid.UUID]) {
	tf.t.Helper()

	selectorConfig := userstore.UserSelectorConfig{WhereClause: "ALL"}
	var selectorValues userstore.UserSelectorValues
	var columns []storage.Column
	columns = append(columns, tf.systemColumns...)

	cm, err := storage.NewUserstoreColumnManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)
	dtm, err := storage.NewDataTypeManager(tf.ctx, tf.s)
	assert.NoErr(tf.t, err)

	systemUsers, code, err := tf.us.GetUsersForSelector(tf.ctx, cm, dtm, testReferenceTime, dlcs, columns, selectorConfig, selectorValues, set.NewUUIDSet(), set.NewUUIDSet(), nil, false)
	assert.NoErr(tf.t, err)
	assert.Equal(tf.t, code, http.StatusOK)
	assert.Equal(tf.t, len(systemUsers), 1)
	assert.NoErr(tf.t, systemUsers[0].Validate())
	assert.Equal(tf.t, tf.user.ID, systemUsers[0].ID)
	assert.Equal(tf.t, len(systemUsers[0].Profile), len(columns))
	assert.Equal(tf.t, len(systemUsers[0].ColumnValues), len(columns))
	for _, c := range columns {
		assert.NotNil(tf.t, systemUsers[0].Profile[c.Name])
		assert.NotNil(tf.t, systemUsers[0].ColumnValues[c.Name])
		assert.Equal(tf.t, len(systemUsers[0].ColumnValues[c.Name]), 1)
	}
	assert.Equal(tf.t, len(systemUsers[0].ProfileConsentedPurposeIDs), 0)

	columns = append(columns, *column)
	nonSystemUsers, code, err := tf.us.GetUsersForSelector(tf.ctx, cm, dtm, testReferenceTime, dlcs, columns, selectorConfig, selectorValues, set.NewUUIDSet(), purposeIDs, nil, false)
	assert.NoErr(tf.t, err)
	assert.Equal(tf.t, code, http.StatusOK)
	assert.Equal(tf.t, len(nonSystemUsers), 1)
	assert.NoErr(tf.t, nonSystemUsers[0].Validate())
	assert.Equal(tf.t, tf.user.ID, nonSystemUsers[0].ID)
	assert.Equal(tf.t, len(nonSystemUsers[0].Profile), len(columns))
	assert.Equal(tf.t, len(nonSystemUsers[0].ColumnValues), len(columns))
	for _, c := range columns {
		assert.NotNil(tf.t, nonSystemUsers[0].Profile[c.Name])
		assert.NotNil(tf.t, nonSystemUsers[0].ColumnValues[c.Name])
		assert.Equal(tf.t, len(nonSystemUsers[0].ColumnValues[c.Name]), 1)
	}
	assert.Equal(tf.t, len(nonSystemUsers[0].ProfileConsentedPurposeIDs), 1)
	assert.NotNil(tf.t, nonSystemUsers[0].ProfileConsentedPurposeIDs[column.Name])
	assert.Equal(tf.t, len(nonSystemUsers[0].ProfileConsentedPurposeIDs[column.Name]), 1)
	assert.Equal(tf.t, len(nonSystemUsers[0].ProfileConsentedPurposeIDs[column.Name][0]), purposeIDs.Size())
	assert.Equal(tf.t, set.NewUUIDSet(nonSystemUsers[0].ProfileConsentedPurposeIDs[column.Name][0]...), purposeIDs)
}
