package updates_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
)

type testFixture struct {
	idptesthelpers.TestFixture

	objectPrefix string

	dataTypesLock sync.Mutex
	dataTypes     map[string]*userstore.ColumnDataType

	columnsLock sync.Mutex
	columns     map[string]*userstore.Column

	purposesLock sync.Mutex
	purposes     map[string]*userstore.Purpose

	transformersLock sync.Mutex
	transformers     map[string]*policy.Transformer

	mutatorsLock sync.Mutex
	mutators     map[string]*userstore.Mutator

	accessorsLock                sync.Mutex
	liveAccessors                map[string]map[string]*userstore.Accessor
	liveCombinedAccessors        map[string]*userstore.Accessor
	softDeletedAccessors         map[string]map[string]*userstore.Accessor
	softDeletedCombinedAccessors map[string]*userstore.Accessor

	// NB: this data doesn't need locked since tests for a single data type are serialized
	addresses         []userstore.Address
	addressesWithIDs  []userstore.Address
	bools             []bool
	composites        []userstore.CompositeValue
	compositesWithIDs []userstore.CompositeValue
	dates             []string
	e164PhoneNumbers  []string
	emails            []string
	ints              []int
	phoneNumbers      []string
	ssns              []string
	strings           []string
	timestamps        []string
	uuids             []uuid.UUID

	testUserID uuid.UUID
}

func newTestFixture(t *testing.T, objectPrefix string) testFixture {
	t.Helper()

	addressesWithIDs := []userstore.Address{
		{ID: "foo", Name: "foo"},
		{ID: "bar", Name: "bar"},
		{ID: "baz", Name: "baz"},
		{ID: "biz", Name: "biz"},
		{ID: "foo", Name: "fee"}, // used to test mutability
		{ID: "bar", Name: "foo"}, // used to test uniqueness
	}

	composites := []userstore.CompositeValue{
		{"b": true, "i": 10, "s": "foo"},
		{"b": false, "i": 20, "s": "bar"},
		{"b": true, "i": 30, "s": "baz"},
		{"b": false, "i": 40, "s": "biz"},
	}

	compositesWithIDs := []userstore.CompositeValue{
		{"id": "foo", "b": true, "i": 10, "s": "foo"},
		{"id": "bar", "b": false, "i": 20, "s": "bar"},
		{"id": "baz", "b": true, "i": 30, "s": "baz"},
		{"id": "biz", "b": false, "i": 40, "s": "biz"},
		{"id": "foo", "b": true, "i": 10, "s": "bar"}, // used to test mutability
		{"id": "bar", "b": true, "i": 10, "s": "foo"}, // used to test uniqueness
	}

	return testFixture{
		TestFixture:                  *idptesthelpers.NewTestFixture(t),
		objectPrefix:                 objectPrefix,
		columns:                      map[string]*userstore.Column{},
		dataTypes:                    map[string]*userstore.ColumnDataType{},
		mutators:                     map[string]*userstore.Mutator{},
		purposes:                     map[string]*userstore.Purpose{},
		transformers:                 map[string]*policy.Transformer{},
		liveAccessors:                map[string]map[string]*userstore.Accessor{},
		liveCombinedAccessors:        map[string]*userstore.Accessor{},
		softDeletedAccessors:         map[string]map[string]*userstore.Accessor{},
		softDeletedCombinedAccessors: map[string]*userstore.Accessor{},
		addresses:                    []userstore.Address{{Name: "foo"}, {Name: "bar"}, {Name: "baz"}, {Name: "biz"}},
		addressesWithIDs:             addressesWithIDs,
		bools:                        []bool{false, true},
		composites:                   composites,
		compositesWithIDs:            compositesWithIDs,
		dates:                        []string{"2023-01-01", "2022-06-01", "2023-05-12", "2023-09-01"},
		emails:                       []string{"foo@bar.com", "foo@bar.org", "biz@bar.com", "baz@bar.com"},
		e164PhoneNumbers:             []string{"+12345678", "+123456789", "+1234567890", "+12345678901"},
		ints:                         []int{42, 23, 5, 12},
		phoneNumbers:                 []string{"123-456-7890", "234-567-8901", "345-678-9012", "456-789-0123"},
		ssns:                         []string{"111-11-1111", "222-22-2222", "333-33-3333", "444-44-4444"},
		strings:                      []string{"foo", "bar", "baz", "biz"},
		timestamps:                   []string{"2023-09-01 12:00:00", "2023-09-01 13:00:00", "2023-09-01 14:00:00", "2023-09-01 09:00:00"},
		uuids:                        []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Nil, uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
	}
}

func (tf *testFixture) initializeDataTypes() {
	tf.T.Helper()

	compositeFields := []userstore.CompositeField{
		{DataType: datatype.Boolean, Name: "B"},
		{DataType: datatype.Integer, Name: "I"},
		{DataType: datatype.String, Name: "S"},
	}
	compositeAttributes := userstore.CompositeAttributes{Fields: compositeFields}
	compositeWithIDAttributes := userstore.CompositeAttributes{IncludeID: true, Fields: compositeFields}

	// populate data types

	for _, ddt := range defaults.GetDefaultDataTypes() {
		dt, err := tf.IDPClient.GetDataType(tf.Ctx, ddt.ID)
		assert.NoErr(tf.T, err)
		tf.dataTypesLock.Lock()
		tf.dataTypes[dt.Name] = dt
		tf.dataTypesLock.Unlock()
		tf.createTransformer(dt)
	}
	tf.createDataType("composite", "test composite data type", compositeAttributes)
	tf.createDataType("composite_with_id", "test composite data type", compositeWithIDAttributes)
}

func (tf *testFixture) initializePurposes() {
	tf.T.Helper()

	for _, dp := range defaults.GetDefaultPurposes() {
		p, err := tf.IDPClient.GetPurpose(tf.Ctx, dp.ID)
		assert.NoErr(tf.T, err)
		tf.purposesLock.Lock()
		tf.purposes[p.Name] = p
		tf.purposesLock.Unlock()
	}

	tf.createPurpose("fraud", "used for fraud prevention, retained after deletion")
	tf.createPurpose("spam", "used for marketing spam")
}

func (tf *testFixture) initializeTestUser() {
	tf.T.Helper()

	testUserID, err :=
		tf.IDPClient.CreateUser(
			tf.Ctx,
			userstore.Record{"email": "joe@schmo.org"},
			idp.OrganizationID(tf.Company.ID),
		)
	assert.NoErr(tf.T, err)
	tf.testUserID = testUserID
}

func (tf *testFixture) createConstrainedColumn(
	columnName string,
	dataTypeName string,
	constraints userstore.ColumnConstraints,
	isArray bool,
	indexType userstore.ColumnIndexType,
) {
	tf.T.Helper()

	// look up data type
	dt, found := tf.dataTypes[dataTypeName]
	assert.True(tf.T, found)
	dataTypeID := userstore.ResourceID{ID: dt.ID}

	// look up transformer for data type
	trfmr, found := tf.transformers[dt.Name]
	assert.True(tf.T, found)
	trfmrID := userstore.ResourceID{ID: trfmr.ID}

	// create column

	c, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:        "users",
			Name:         tf.getColumnName(columnName),
			DataType:     dataTypeID,
			IsArray:      isArray,
			DefaultValue: "",
			IndexType:    indexType,
			Constraints:  constraints,
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)

	// retain soft-deleted values for the column and fraud purpose

	tf.purposesLock.Lock()
	fraudPurpose, found := tf.purposes["fraud"]
	tf.purposesLock.Unlock()
	assert.True(tf.T, found)

	ucrdr := idp.UpdateColumnRetentionDurationsRequest{
		RetentionDurations: []idp.ColumnRetentionDuration{
			{
				ColumnID:     c.ID,
				PurposeID:    fraudPurpose.ID,
				DurationType: userstore.DataLifeCycleStateSoftDeleted,
				Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
				UseDefault:   false,
			},
		},
	}

	resp, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, c.ID, ucrdr)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.RetentionDurations), 1)
	resp, err = tf.IDPClient.GetColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, c.ID)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.RetentionDurations), len(tf.purposes))

	tf.columnsLock.Lock()
	tf.columns[columnName] = c
	tf.columnsLock.Unlock()

	// create mutator

	m, err := tf.CreateMutator(
		tf.getMutatorName(columnName),
		policy.AccessPolicyAllowAll.ID,
		[]string{c.Name},
		[]uuid.UUID{trfmrID.ID},
	)
	assert.NoErr(tf.T, err)
	tf.mutatorsLock.Lock()
	tf.mutators[columnName] = m
	tf.mutatorsLock.Unlock()

	// create live and soft-deleted accessors for each purpose

	tf.accessorsLock.Lock()
	tf.liveAccessors[columnName] = map[string]*userstore.Accessor{}
	tf.softDeletedAccessors[columnName] = map[string]*userstore.Accessor{}
	tf.accessorsLock.Unlock()

	for purposeName, purpose := range tf.purposes {
		a, err := tf.CreateLiveAccessor(
			tf.getLiveAccessorName(columnName, purposeName),
			policy.AccessPolicyAllowAll.ID,
			[]string{c.Name},
			[]uuid.UUID{trfmrID.ID},
			[]string{purpose.Name},
		)
		assert.NoErr(tf.T, err)
		tf.accessorsLock.Lock()
		tf.liveAccessors[columnName][purposeName] = a
		tf.accessorsLock.Unlock()

		a, err = tf.CreateSoftDeletedAccessor(
			tf.getSoftDeletedAccessorName(columnName, purposeName),
			policy.AccessPolicyAllowAll.ID,
			[]string{c.Name},
			[]uuid.UUID{trfmrID.ID},
			[]string{purpose.Name},
		)
		assert.NoErr(tf.T, err)
		tf.accessorsLock.Lock()
		tf.softDeletedAccessors[columnName][purposeName] = a
		tf.accessorsLock.Unlock()
	}
}

func (tf *testFixture) createCombinedAccessors() {
	tf.T.Helper()

	columnNames := []string{}
	transformerIDs := []uuid.UUID{}
	for _, c := range tf.columns {
		columnNames = append(columnNames, c.Name)
		transformerIDs = append(transformerIDs, policy.TransformerPassthrough.ID)
	}

	for purposeName, purpose := range tf.purposes {
		a, err := tf.CreateLiveAccessor(
			tf.objectPrefix+purposeName+"_live_accessor_all",
			policy.AccessPolicyAllowAll.ID,
			columnNames,
			transformerIDs,
			[]string{purpose.Name},
		)
		assert.NoErr(tf.T, err)
		tf.accessorsLock.Lock()
		tf.liveCombinedAccessors[purposeName] = a
		tf.accessorsLock.Unlock()

		a, err = tf.CreateSoftDeletedAccessor(
			tf.objectPrefix+purposeName+"_soft_deleted_accessor_all",
			policy.AccessPolicyAllowAll.ID,
			columnNames,
			transformerIDs,
			[]string{purpose.Name},
		)
		assert.NoErr(tf.T, err)
		tf.accessorsLock.Lock()
		tf.softDeletedCombinedAccessors[purposeName] = a
		tf.accessorsLock.Unlock()
	}
}

func (tf *testFixture) createDataType(
	dataTypeName string,
	description string,
	attributes userstore.CompositeAttributes,
) {
	tf.T.Helper()

	dt, err := tf.IDPClient.CreateDataType(
		tf.Ctx,
		userstore.ColumnDataType{
			Name:                tf.getDataTypeName(dataTypeName),
			Description:         description,
			CompositeAttributes: attributes,
		},
	)
	assert.NoErr(tf.T, err)
	tf.dataTypesLock.Lock()
	tf.dataTypes[dataTypeName] = dt
	tf.dataTypesLock.Unlock()
	tf.createTransformer(dt)
}

func (tf *testFixture) createPurpose(purposeName string, description string) {
	tf.T.Helper()

	p, err := tf.IDPClient.CreatePurpose(
		tf.Ctx,
		userstore.Purpose{
			Name:        tf.getPurposeName(purposeName),
			Description: description,
		},
	)
	assert.NoErr(tf.T, err)

	tf.purposesLock.Lock()
	_, found := tf.purposes[purposeName]
	assert.False(tf.T, found)
	tf.purposes[purposeName] = p
	tf.purposesLock.Unlock()
}

func (tf *testFixture) createTransformer(
	cdt *userstore.ColumnDataType,
) {
	tf.T.Helper()

	trfmrName := tf.objectPrefix + "transformer_" + cdt.Name
	function := `function transform(data, params) { return data; } // ` + trfmrName

	trfmr, err := tf.TokenizerClient.CreateTransformer(
		tf.Ctx,
		policy.Transformer{
			Name:           trfmrName,
			Description:    "test transformer",
			InputDataType:  userstore.ResourceID{ID: cdt.ID},
			OutputDataType: userstore.ResourceID{ID: cdt.ID},
			TransformType:  policy.TransformTypeTransform,
			Function:       function,
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)

	tf.transformersLock.Lock()
	tf.transformers[cdt.Name] = trfmr
	tf.transformersLock.Unlock()
}

func (tf *testFixture) getColumnName(columnName string) string {
	if c, found := tf.columns[columnName]; found {
		return c.Name
	}
	return tf.objectPrefix + columnName
}

func (tf *testFixture) getDataTypeName(dataTypeName string) string {
	if dt, found := tf.dataTypes[dataTypeName]; found {
		return dt.Name
	}
	return tf.objectPrefix + dataTypeName
}

func (tf *testFixture) getLiveAccessorName(
	columnName string,
	purposeName string,
) string {
	if accessors, found := tf.liveAccessors[columnName]; found {
		if accessor, found := accessors[purposeName]; found {
			return accessor.Name
		}
	}
	return tf.objectPrefix + purposeName + "_live_accessor_" + columnName
}

func (tf *testFixture) getMutatorName(columnName string) string {
	if m, found := tf.mutators[columnName]; found {
		return m.Name
	}
	return tf.objectPrefix + "mutator_" + columnName
}

func (tf *testFixture) getPurposeName(purposeName string) string {
	if p, found := tf.purposes[purposeName]; found {
		return p.Name
	}
	return tf.objectPrefix + purposeName
}

func (tf *testFixture) getSoftDeletedAccessorName(
	columnName string,
	purposeName string,
) string {
	if accessors, found := tf.softDeletedAccessors[columnName]; found {
		if accessor, found := accessors[purposeName]; found {
			return accessor.Name
		}
	}
	return tf.objectPrefix + purposeName + "_soft_deleted_accessor_" + columnName
}

func (tf *testFixture) getAddress(i int) userstore.Address {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.addresses))
	return tf.addresses[i]
}

func (tf *testFixture) getAddressWithID(i int) userstore.Address {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.addressesWithIDs))
	return tf.addressesWithIDs[i]
}

func (tf *testFixture) getAddresses(indexes ...int) []userstore.Address {
	tf.T.Helper()
	addresses := []userstore.Address{}
	for _, i := range indexes {
		addresses = append(addresses, tf.getAddress(i))
	}
	return addresses
}

func (tf *testFixture) getAddressesWithIDs(indexes ...int) []userstore.Address {
	tf.T.Helper()
	addresses := []userstore.Address{}
	for _, i := range indexes {
		addresses = append(addresses, tf.getAddressWithID(i))
	}
	return addresses
}

func (tf *testFixture) getBool(i int) bool {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.bools))
	return tf.bools[i]
}

func (tf *testFixture) getBools(indexes ...int) []bool {
	tf.T.Helper()
	bools := []bool{}
	for _, i := range indexes {
		bools = append(bools, tf.getBool(i))
	}
	return bools
}

func (tf *testFixture) getComposite(i int) userstore.CompositeValue {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.composites))
	return tf.composites[i]
}

func (tf *testFixture) getCompositeWithID(i int) userstore.CompositeValue {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.compositesWithIDs))
	return tf.compositesWithIDs[i]
}

func (tf *testFixture) getComposites(indexes ...int) []userstore.CompositeValue {
	tf.T.Helper()
	composites := []userstore.CompositeValue{}
	for _, i := range indexes {
		composites = append(composites, tf.getComposite(i))
	}
	return composites
}

func (tf *testFixture) getCompositesWithIDs(indexes ...int) []userstore.CompositeValue {
	tf.T.Helper()
	composites := []userstore.CompositeValue{}
	for _, i := range indexes {
		composites = append(composites, tf.getCompositeWithID(i))
	}
	return composites
}

func (tf *testFixture) getDate(i int) time.Time {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.dates))
	d, err := time.Parse(time.DateOnly, tf.dates[i])
	assert.NoErr(tf.T, err)
	return d
}

func (tf *testFixture) getDateStr(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.dates))
	return tf.dates[i]
}

func (tf *testFixture) getDates(indexes ...int) []time.Time {
	tf.T.Helper()
	dates := []time.Time{}
	for _, i := range indexes {
		dates = append(dates, tf.getDate(i))
	}
	return dates
}

func (tf *testFixture) getDateStrs(indexes ...int) []string {
	tf.T.Helper()
	dates := []string{}
	for _, i := range indexes {
		dates = append(dates, tf.getDateStr(i))
	}
	return dates
}

func (tf *testFixture) getEmail(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.emails))
	return tf.emails[i]
}

func (tf *testFixture) getEmails(indexes ...int) []string {
	tf.T.Helper()
	emails := []string{}
	for _, i := range indexes {
		emails = append(emails, tf.getEmail(i))
	}
	return emails
}

func (tf *testFixture) getE164(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.e164PhoneNumbers))
	return tf.e164PhoneNumbers[i]
}

func (tf *testFixture) getE164s(indexes ...int) []string {
	tf.T.Helper()
	e164PhoneNumbers := []string{}
	for _, i := range indexes {
		e164PhoneNumbers = append(e164PhoneNumbers, tf.getE164(i))
	}
	return e164PhoneNumbers
}

func (tf *testFixture) getInt(i int) int {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.ints))
	return tf.ints[i]
}

func (tf *testFixture) getIntStr(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.ints))
	return fmt.Sprintf("%v", tf.ints[i])
}

func (tf *testFixture) getInts(indexes ...int) []int {
	tf.T.Helper()
	ints := []int{}
	for _, i := range indexes {
		ints = append(ints, tf.getInt(i))
	}
	return ints
}

func (tf *testFixture) getPhone(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.phoneNumbers))
	return tf.phoneNumbers[i]
}

func (tf *testFixture) getPhones(indexes ...int) []string {
	tf.T.Helper()
	phoneNumbers := []string{}
	for _, i := range indexes {
		phoneNumbers = append(phoneNumbers, tf.getPhone(i))
	}
	return phoneNumbers
}

func (tf *testFixture) getSSN(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.ssns))
	return tf.ssns[i]
}

func (tf *testFixture) getSSNs(indexes ...int) []string {
	tf.T.Helper()
	ssns := []string{}
	for _, i := range indexes {
		ssns = append(ssns, tf.getSSN(i))
	}
	return ssns
}

func (tf *testFixture) getString(i int) string {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.strings))
	return tf.strings[i]
}

func (tf *testFixture) getStrings(indexes ...int) []string {
	tf.T.Helper()
	strings := []string{}
	for _, i := range indexes {
		strings = append(strings, tf.getString(i))
	}
	return strings
}

func (tf *testFixture) getTimestamp(i int) time.Time {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.timestamps))
	t, err := time.Parse(time.DateTime, tf.timestamps[i])
	assert.NoErr(tf.T, err)
	return t
}

func (tf *testFixture) getTimestamps(indexes ...int) []time.Time {
	tf.T.Helper()
	timestamps := []time.Time{}
	for _, i := range indexes {
		timestamps = append(timestamps, tf.getTimestamp(i))
	}
	return timestamps
}

func (tf *testFixture) getUUID(i int) uuid.UUID {
	tf.T.Helper()
	assert.True(tf.T, i < len(tf.uuids))
	return tf.uuids[i]
}

func (tf *testFixture) getUUIDs(indexes ...int) []uuid.UUID {
	tf.T.Helper()
	uuids := []uuid.UUID{}
	for _, i := range indexes {
		uuids = append(uuids, tf.getUUID(i))
	}
	return uuids
}

func (tf *testFixture) lookupValue(
	columnName string,
	purposeNames []string,
	expectedLiveValues map[string]any,
	expectedSoftDeletedValues map[string]any,
) {
	tf.T.Helper()
	tf.accessorsLock.Lock()
	defer tf.accessorsLock.Unlock()

	for _, purposeName := range purposeNames {
		ret, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.liveAccessors[columnName][purposeName].ID,
			policy.ClientContext{},
			[]any{tf.testUserID},
		)
		assert.NoErr(tf.T, err)

		expectedLiveValue := expectedLiveValues[purposeName]
		if expectedLiveValue == nil {
			assert.Equal(tf.T, len(ret.Data), 0)
		} else {
			assert.Equal(tf.T, len(ret.Data), 1)
			tf.validateResult(ret.Data[0], columnName, expectedLiveValue)
		}

		ret, err = tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.softDeletedAccessors[columnName][purposeName].ID,
			policy.ClientContext{},
			[]any{tf.testUserID},
		)
		assert.NoErr(tf.T, err)

		expectedSoftDeletedValue := expectedSoftDeletedValues[purposeName]
		if expectedSoftDeletedValue == nil {
			assert.Equal(tf.T, len(ret.Data), 0)
		} else {
			assert.Equal(tf.T, len(ret.Data), 1)
			tf.validateResult(ret.Data[0], columnName, expectedSoftDeletedValue)
		}
	}
}

func (tf *testFixture) validateResult(value string, columnName string, expected any) {
	tf.T.Helper()

	bytes, err := json.Marshal(expected)
	assert.NoErr(tf.T, err)
	expectedString := string(bytes)
	if value == fmt.Sprintf(`{"%s":%#v}`, tf.getColumnName(columnName), expectedString) {
		return
	}

	assert.Equal(tf.T, value, fmt.Sprintf(`{"%s":%v}`, tf.getColumnName(columnName), expectedString))
}
