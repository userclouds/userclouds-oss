package updates_test

import (
	"testing"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
)

type fullUpdateTestFixture struct {
	testFixture
}

func initFullUpdateTestFixture(t *testing.T) *fullUpdateTestFixture {
	t.Helper()

	tf := &fullUpdateTestFixture{
		testFixture: newTestFixture(t, "full_update_"),
	}

	tf.initializePurposes()

	tf.initializeDataTypes()

	tf.createSingleValueColumn("address", "canonical_address")
	tf.createSingleValueColumn("birthdate", "birthdate")
	tf.createSingleValueColumn("boolean", "boolean")
	tf.createSingleValueColumn("composite", "composite")
	tf.createSingleValueColumn("date", "date")
	tf.createSingleValueColumn("e164", "e164_phonenumber")
	tf.createSingleValueColumn("email", "email")
	tf.createSingleValueColumn("integer", "integer")
	tf.createSingleValueColumn("phonenumber", "phonenumber")
	tf.createSingleValueColumn("ssn", "ssn")
	tf.createSingleValueColumn("string", "string")
	tf.createSingleValueColumn("timestamp", "timestamp")
	tf.createSingleValueColumn("uuid", "uuid")

	tf.createArrayColumn("addresses", "canonical_address")
	tf.createArrayColumn("birthdates", "birthdate")
	tf.createArrayColumn("booleans", "boolean")
	tf.createArrayColumn("composites", "composite")
	tf.createArrayColumn("dates", "date")
	tf.createArrayColumn("e164s", "e164_phonenumber")
	tf.createArrayColumn("emails", "email")
	tf.createArrayColumn("integers", "integer")
	tf.createArrayColumn("phonenumbers", "phonenumber")
	tf.createArrayColumn("ssns", "ssn")
	tf.createArrayColumn("strings", "string")
	tf.createArrayColumn("timestamps", "timestamp")
	tf.createArrayColumn("uuids", "uuid")

	tf.createUniqueColumn("e164_unique", "e164_phonenumber")
	tf.createUniqueColumn("email_unique", "email")
	tf.createUniqueColumn("integer_unique", "integer")
	tf.createUniqueColumn("phonenumber_unique", "phonenumber")
	tf.createUniqueColumn("ssn_unique", "ssn")
	tf.createUniqueColumn("string_unique", "string")
	tf.createUniqueColumn("uuid_unique", "uuid")

	tf.createImmutableSingleValueColumn("immutable_address", "canonical_address")
	tf.createImmutableArrayColumn("immutable_addresses", "canonical_address")
	tf.createImmutableSingleValueColumn("immutable_composite", "composite_with_id")
	tf.createImmutableArrayColumn("immutable_composites", "composite_with_id")

	tf.createCombinedAccessors()

	tf.initializeTestUser()

	return tf
}

func (tf *fullUpdateTestFixture) createArrayColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *fullUpdateTestFixture) createImmutableArrayColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{UniqueIDRequired: true, ImmutableRequired: true},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *fullUpdateTestFixture) createImmutableSingleValueColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{UniqueIDRequired: true, ImmutableRequired: true},
		false,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *fullUpdateTestFixture) createSingleValueColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{},
		false,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *fullUpdateTestFixture) createUniqueColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{},
		false,
		userstore.ColumnIndexTypeUnique,
	)
}

func (tf *fullUpdateTestFixture) executeMutator(
	columnName string,
	vap idp.ValueAndPurposes,
) (*idp.ExecuteMutatorResponse, error) {
	tf.T.Helper()
	tf.mutatorsLock.Lock()
	defer tf.mutatorsLock.Unlock()

	resp, err := tf.IDPClient.ExecuteMutator(
		tf.Ctx,
		tf.mutators[columnName].ID,
		policy.ClientContext{},
		[]any{tf.testUserID},
		map[string]idp.ValueAndPurposes{tf.getColumnName(columnName): vap},
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return resp, nil
}

func (tf *fullUpdateTestFixture) addSpamPurpose(columnName string) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			Value: idp.MutatorColumnCurrentValue,
			PurposeAdditions: []userstore.ResourceID{
				{Name: tf.getPurposeName("spam")},
			},
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func (tf *fullUpdateTestFixture) removeAllPurposes(columnName string) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			Value: idp.MutatorColumnCurrentValue,
			PurposeDeletions: []userstore.ResourceID{
				{Name: tf.getPurposeName("spam")},
				{Name: tf.getPurposeName("operational")},
				{Name: tf.getPurposeName("fraud")},
			},
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func (tf *fullUpdateTestFixture) failOperationalFraudValueUpdate(columnName string, value any) {
	tf.T.Helper()

	_, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			Value: value,
			PurposeAdditions: []userstore.ResourceID{
				{Name: tf.getPurposeName("operational")},
				{Name: tf.getPurposeName("fraud")},
			},
		},
	)
	assert.NotNil(tf.T, err)
}

func (tf *fullUpdateTestFixture) updateOperationalFraudValue(columnName string, value any) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			Value: value,
			PurposeAdditions: []userstore.ResourceID{
				{Name: tf.getPurposeName("operational")},
				{Name: tf.getPurposeName("fraud")},
			},
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func (tf *fullUpdateTestFixture) testAddSpamPurpose(
	columnName string,
	expectedLiveValue any,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.addSpamPurpose(columnName)
	tf.lookupValue(
		columnName,
		[]string{"spam", "operational", "fraud"},
		map[string]any{"spam": expectedLiveValue, "operational": expectedLiveValue, "fraud": expectedLiveValue},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *fullUpdateTestFixture) testRemoveAllPurposes(columnName string, expectedSoftDeletedFraudValue any) {
	tf.T.Helper()

	tf.removeAllPurposes(columnName)
	tf.lookupValue(
		columnName,
		[]string{"spam", "operational", "fraud"},
		map[string]any{},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *fullUpdateTestFixture) testUpdateOperationalFraudValue(
	columnName string,
	updateValue any,
	expectedLiveValue any,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.updateOperationalFraudValue(columnName, updateValue)
	tf.lookupValue(
		columnName,
		[]string{"operational", "fraud"},
		map[string]any{"operational": expectedLiveValue, "fraud": expectedLiveValue},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func TestFullUpdates(t *testing.T) {
	tf := initFullUpdateTestFixture(t)
	// single-value non-unique columns

	t.Run("test_address", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("address", tf.getAddress(0), tf.getAddress(0), nil)
		tf.testUpdateOperationalFraudValue("address", tf.getAddress(0), tf.getAddress(0), nil)
		tf.testUpdateOperationalFraudValue("address", tf.getAddress(1), tf.getAddress(1), tf.getAddresses(0))
		tf.testUpdateOperationalFraudValue("address", tf.getAddress(3), tf.getAddress(3), tf.getAddresses(0, 1))
		tf.testAddSpamPurpose("address", tf.getAddress(3), tf.getAddresses(0, 1))
		tf.testRemoveAllPurposes("address", tf.getAddresses(0, 1, 3))

		tf.testUpdateOperationalFraudValue("addresses", tf.getAddresses(0, 1), tf.getAddresses(0, 1), nil)
		tf.testUpdateOperationalFraudValue("addresses", tf.getAddresses(0, 1), tf.getAddresses(0, 1), nil)
		tf.testUpdateOperationalFraudValue("addresses", tf.getAddresses(2, 0), tf.getAddresses(2, 0), tf.getAddresses(1))
		tf.testUpdateOperationalFraudValue("addresses", tf.getAddresses(1, 0, 2, 0), tf.getAddresses(1, 0, 2, 0), tf.getAddresses(1))
		tf.testUpdateOperationalFraudValue("addresses", tf.getAddresses(1, 2, 0), tf.getAddresses(1, 2, 0), tf.getAddresses(1, 0))
		tf.testAddSpamPurpose("addresses", tf.getAddresses(1, 2, 0), tf.getAddresses(1, 0))
		tf.testRemoveAllPurposes("addresses", tf.getAddresses(1, 0, 1, 2, 0))

		// test that immutability constraint cannot be violated
		tf.testUpdateOperationalFraudValue("immutable_address", tf.getAddressWithID(0), tf.getAddressWithID(0), nil)
		tf.failOperationalFraudValueUpdate("immutable_address", tf.getAddressWithID(4))
		tf.testUpdateOperationalFraudValue("immutable_addresses", tf.getAddressesWithIDs(0), tf.getAddressesWithIDs(0), nil)
		tf.testUpdateOperationalFraudValue("immutable_addresses", tf.getAddressesWithIDs(0), tf.getAddressesWithIDs(0), nil)
		tf.failOperationalFraudValueUpdate("immutable_addresses", tf.getAddressesWithIDs(4))
	})

	t.Run("test_birthdate", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("birthdate", tf.getDate(0), tf.getDateStr(0), nil)
		tf.testUpdateOperationalFraudValue("birthdate", tf.getDate(0), tf.getDateStr(0), nil)
		tf.testUpdateOperationalFraudValue("birthdate", tf.getDate(1), tf.getDateStr(1), tf.getDateStrs(0))
		tf.testAddSpamPurpose("birthdate", tf.getDateStr(1), tf.getDateStrs(0))
		tf.testRemoveAllPurposes("birthdate", tf.getDateStrs(0, 1))

		tf.testUpdateOperationalFraudValue("birthdates", tf.getDates(0, 1), tf.getDateStrs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("birthdates", tf.getDates(0, 1), tf.getDateStrs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("birthdates", tf.getDates(1, 2, 3), tf.getDateStrs(1, 2, 3), tf.getDateStrs(0))
		tf.testUpdateOperationalFraudValue("birthdates", tf.getDates(3, 1, 2), tf.getDateStrs(3, 1, 2), tf.getDateStrs(0))
		tf.testAddSpamPurpose("birthdates", tf.getDateStrs(3, 1, 2), tf.getDateStrs(0))
		tf.testRemoveAllPurposes("birthdates", tf.getDateStrs(0, 3, 1, 2))
	})

	t.Run("test_boolean", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("boolean", tf.getBool(1), tf.getBool(1), nil)
		tf.testUpdateOperationalFraudValue("boolean", tf.getBool(1), tf.getBool(1), nil)
		tf.testUpdateOperationalFraudValue("boolean", tf.getBool(0), tf.getBool(0), tf.getBools(1))
		tf.testAddSpamPurpose("boolean", tf.getBool(0), tf.getBools(1))
		tf.testRemoveAllPurposes("boolean", tf.getBools(1, 0))

		tf.testUpdateOperationalFraudValue("booleans", tf.getBools(1, 0), tf.getBools(1, 0), nil)
		tf.testUpdateOperationalFraudValue("booleans", tf.getBools(1, 0), tf.getBools(1, 0), nil)
		tf.testUpdateOperationalFraudValue("booleans", tf.getBools(0), tf.getBools(0), tf.getBools(1))
		tf.testUpdateOperationalFraudValue("booleans", tf.getBools(1, 1, 1, 1, 1), tf.getBools(1, 1, 1, 1, 1), tf.getBools(1, 0))
		tf.testUpdateOperationalFraudValue("booleans", tf.getBools(1, 1), tf.getBools(1, 1), tf.getBools(1, 0, 1, 1, 1))
		tf.testAddSpamPurpose("booleans", tf.getBools(1, 1), tf.getBools(1, 0, 1, 1, 1))
		tf.testRemoveAllPurposes("booleans", tf.getBools(1, 0, 1, 1, 1, 1, 1))
	})

	t.Run("test_composite", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("composite", tf.getComposite(0), tf.getComposite(0), nil)
		tf.testUpdateOperationalFraudValue("composite", tf.getComposite(0), tf.getComposite(0), nil)
		tf.testUpdateOperationalFraudValue("composite", tf.getComposite(1), tf.getComposite(1), tf.getComposites(0))
		tf.testAddSpamPurpose("composite", tf.getComposite(1), tf.getComposites(0))
		tf.testRemoveAllPurposes("composite", tf.getComposites(0, 1))

		tf.testUpdateOperationalFraudValue("composites", tf.getComposites(0, 1), tf.getComposites(0, 1), nil)
		tf.testUpdateOperationalFraudValue("composites", tf.getComposites(0, 1), tf.getComposites(0, 1), nil)
		tf.testUpdateOperationalFraudValue("composites", tf.getComposites(1, 2, 3), tf.getComposites(1, 2, 3), tf.getComposites(0))
		tf.testUpdateOperationalFraudValue("composites", tf.getComposites(3, 1, 2), tf.getComposites(3, 1, 2), tf.getComposites(0))
		tf.testAddSpamPurpose("composites", tf.getComposites(3, 1, 2), tf.getComposites(0))
		tf.testRemoveAllPurposes("composites", tf.getComposites(0, 3, 1, 2))

		// test that immutability constraint cannot be violated
		tf.testUpdateOperationalFraudValue("immutable_composite", tf.getCompositeWithID(0), tf.getCompositeWithID(0), nil)
		tf.failOperationalFraudValueUpdate("immutable_composite", tf.getCompositeWithID(4))
		tf.testUpdateOperationalFraudValue("immutable_composites", tf.getCompositesWithIDs(0), tf.getCompositesWithIDs(0), nil)
		tf.testUpdateOperationalFraudValue("immutable_composites", tf.getCompositesWithIDs(0), tf.getCompositesWithIDs(0), nil)
		tf.failOperationalFraudValueUpdate("immutable_composites", tf.getCompositesWithIDs(4))
	})

	t.Run("test_date", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("date", tf.getDate(0), tf.getDateStr(0), nil)
		tf.testUpdateOperationalFraudValue("date", tf.getDate(0), tf.getDateStr(0), nil)
		tf.testUpdateOperationalFraudValue("date", tf.getDate(1), tf.getDateStr(1), tf.getDateStrs(0))
		tf.testAddSpamPurpose("date", tf.getDateStr(1), tf.getDateStrs(0))
		tf.testRemoveAllPurposes("date", tf.getDateStrs(0, 1))

		tf.testUpdateOperationalFraudValue("dates", tf.getDates(0, 1), tf.getDateStrs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("dates", tf.getDates(0, 1), tf.getDateStrs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("dates", tf.getDates(1, 2, 3), tf.getDateStrs(1, 2, 3), tf.getDateStrs(0))
		tf.testUpdateOperationalFraudValue("dates", tf.getDates(3, 1, 2), tf.getDateStrs(3, 1, 2), tf.getDateStrs(0))
		tf.testAddSpamPurpose("dates", tf.getDateStrs(3, 1, 2), tf.getDateStrs(0))
		tf.testRemoveAllPurposes("dates", tf.getDateStrs(0, 3, 1, 2))
	})

	t.Run("test_email", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("email", tf.getEmail(1), tf.getEmail(1), nil)
		tf.testUpdateOperationalFraudValue("email", tf.getEmail(1), tf.getEmail(1), nil)
		tf.testUpdateOperationalFraudValue("email", tf.getEmail(0), tf.getEmail(0), tf.getEmails(1))
		tf.testAddSpamPurpose("email", tf.getEmail(0), tf.getEmails(1))
		tf.testRemoveAllPurposes("email", tf.getEmails(1, 0))

		tf.testUpdateOperationalFraudValue("emails", tf.getEmails(0, 1), tf.getEmails(0, 1), nil)
		tf.testUpdateOperationalFraudValue("emails", tf.getEmails(0, 1), tf.getEmails(0, 1), nil)
		tf.testUpdateOperationalFraudValue("emails", tf.getEmails(1, 2, 3), tf.getEmails(1, 2, 3), tf.getEmails(0))
		tf.testUpdateOperationalFraudValue("emails", tf.getEmails(3, 1, 2), tf.getEmails(3, 1, 2), tf.getEmails(0))
		tf.testAddSpamPurpose("emails", tf.getEmails(3, 1, 2), tf.getEmails(0))
		tf.testRemoveAllPurposes("emails", tf.getEmails(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("email_unique", tf.getEmail(0), tf.getEmail(0), nil)
		tf.testUpdateOperationalFraudValue("email_unique", tf.getEmail(0), tf.getEmail(0), nil)
		tf.testUpdateOperationalFraudValue("email_unique", tf.getEmail(2), tf.getEmail(2), tf.getEmails(0))
		tf.testUpdateOperationalFraudValue("email_unique", tf.getEmail(1), tf.getEmail(1), tf.getEmails(0, 2))
		tf.testUpdateOperationalFraudValue("email_unique", tf.getEmail(2), tf.getEmail(2), tf.getEmails(0, 2, 1))
		tf.testAddSpamPurpose("email_unique", tf.getEmail(2), tf.getEmails(0, 2, 1))
		tf.testRemoveAllPurposes("email_unique", tf.getEmails(0, 2, 1, 2))
	})

	t.Run("test_e164", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("e164", tf.getE164(1), tf.getE164(1), nil)
		tf.testUpdateOperationalFraudValue("e164", tf.getE164(1), tf.getE164(1), nil)
		tf.testUpdateOperationalFraudValue("e164", tf.getE164(0), tf.getE164(0), tf.getE164s(1))
		tf.testAddSpamPurpose("e164", tf.getE164(0), tf.getE164s(1))
		tf.testRemoveAllPurposes("e164", tf.getE164s(1, 0))

		tf.testUpdateOperationalFraudValue("e164s", tf.getE164s(0, 1), tf.getE164s(0, 1), nil)
		tf.testUpdateOperationalFraudValue("e164s", tf.getE164s(0, 1), tf.getE164s(0, 1), nil)
		tf.testUpdateOperationalFraudValue("e164s", tf.getE164s(1, 2, 3), tf.getE164s(1, 2, 3), tf.getE164s(0))
		tf.testUpdateOperationalFraudValue("e164s", tf.getE164s(3, 1, 2), tf.getE164s(3, 1, 2), tf.getE164s(0))
		tf.testAddSpamPurpose("e164s", tf.getE164s(3, 1, 2), tf.getE164s(0))
		tf.testRemoveAllPurposes("e164s", tf.getE164s(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("e164_unique", tf.getE164(0), tf.getE164(0), nil)
		tf.testUpdateOperationalFraudValue("e164_unique", tf.getE164(0), tf.getE164(0), nil)
		tf.testUpdateOperationalFraudValue("e164_unique", tf.getE164(2), tf.getE164(2), tf.getE164s(0))
		tf.testUpdateOperationalFraudValue("e164_unique", tf.getE164(1), tf.getE164(1), tf.getE164s(0, 2))
		tf.testUpdateOperationalFraudValue("e164_unique", tf.getE164(2), tf.getE164(2), tf.getE164s(0, 2, 1))
		tf.testAddSpamPurpose("e164_unique", tf.getE164(2), tf.getE164s(0, 2, 1))
		tf.testRemoveAllPurposes("e164_unique", tf.getE164s(0, 2, 1, 2))
	})

	t.Run("test_integer", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("integer", tf.getIntStr(0), tf.getInt(0), nil)
		tf.testUpdateOperationalFraudValue("integer", tf.getIntStr(0), tf.getInt(0), nil)
		tf.testUpdateOperationalFraudValue("integer", tf.getIntStr(1), tf.getInt(1), tf.getInts(0))
		tf.testAddSpamPurpose("integer", tf.getInt(1), tf.getInts(0))
		tf.testRemoveAllPurposes("integer", tf.getInts(0, 1))

		tf.testUpdateOperationalFraudValue("integers", tf.getInts(0, 1), tf.getInts(0, 1), nil)
		tf.testUpdateOperationalFraudValue("integers", tf.getInts(0, 1), tf.getInts(0, 1), nil)
		tf.testUpdateOperationalFraudValue("integers", tf.getInts(1, 2, 3), tf.getInts(1, 2, 3), tf.getInts(0))
		tf.testUpdateOperationalFraudValue("integers", tf.getInts(3, 1, 2), tf.getInts(3, 1, 2), tf.getInts(0))
		tf.testAddSpamPurpose("integers", tf.getInts(3, 1, 2), tf.getInts(0))
		tf.testRemoveAllPurposes("integers", tf.getInts(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("integer_unique", tf.getIntStr(0), tf.getInt(0), nil)
		tf.testUpdateOperationalFraudValue("integer_unique", tf.getIntStr(0), tf.getInt(0), nil)
		tf.testUpdateOperationalFraudValue("integer_unique", tf.getIntStr(3), tf.getInt(3), tf.getInts(0))
		tf.testUpdateOperationalFraudValue("integer_unique", tf.getIntStr(2), tf.getInt(2), tf.getInts(0, 3))
		tf.testAddSpamPurpose("integer_unique", tf.getInt(2), tf.getInts(0, 3))
		tf.testRemoveAllPurposes("integer_unique", tf.getInts(0, 3, 2))
	})

	t.Run("test_phonenumber", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("phonenumber", tf.getPhone(1), tf.getPhone(1), nil)
		tf.testUpdateOperationalFraudValue("phonenumber", tf.getPhone(1), tf.getPhone(1), nil)
		tf.testUpdateOperationalFraudValue("phonenumber", tf.getPhone(0), tf.getPhone(0), tf.getPhones(1))
		tf.testAddSpamPurpose("phonenumber", tf.getPhone(0), tf.getPhones(1))
		tf.testRemoveAllPurposes("phonenumber", tf.getPhones(1, 0))

		tf.testUpdateOperationalFraudValue("phonenumbers", tf.getPhones(0, 1), tf.getPhones(0, 1), nil)
		tf.testUpdateOperationalFraudValue("phonenumbers", tf.getPhones(0, 1), tf.getPhones(0, 1), nil)
		tf.testUpdateOperationalFraudValue("phonenumbers", tf.getPhones(1, 2, 3), tf.getPhones(1, 2, 3), tf.getPhones(0))
		tf.testUpdateOperationalFraudValue("phonenumbers", tf.getPhones(3, 1, 2), tf.getPhones(3, 1, 2), tf.getPhones(0))
		tf.testAddSpamPurpose("phonenumbers", tf.getPhones(3, 1, 2), tf.getPhones(0))
		tf.testRemoveAllPurposes("phonenumbers", tf.getPhones(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("phonenumber_unique", tf.getPhone(0), tf.getPhone(0), nil)
		tf.testUpdateOperationalFraudValue("phonenumber_unique", tf.getPhone(0), tf.getPhone(0), nil)
		tf.testUpdateOperationalFraudValue("phonenumber_unique", tf.getPhone(2), tf.getPhone(2), tf.getPhones(0))
		tf.testUpdateOperationalFraudValue("phonenumber_unique", tf.getPhone(1), tf.getPhone(1), tf.getPhones(0, 2))
		tf.testUpdateOperationalFraudValue("phonenumber_unique", tf.getPhone(2), tf.getPhone(2), tf.getPhones(0, 2, 1))
		tf.testAddSpamPurpose("phonenumber_unique", tf.getPhone(2), tf.getPhones(0, 2, 1))
		tf.testRemoveAllPurposes("phonenumber_unique", tf.getPhones(0, 2, 1, 2))
	})

	t.Run("test_ssn", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("ssn", tf.getSSN(1), tf.getSSN(1), nil)
		tf.testUpdateOperationalFraudValue("ssn", tf.getSSN(1), tf.getSSN(1), nil)
		tf.testUpdateOperationalFraudValue("ssn", tf.getSSN(0), tf.getSSN(0), tf.getSSNs(1))
		tf.testAddSpamPurpose("ssn", tf.getSSN(0), tf.getSSNs(1))
		tf.testRemoveAllPurposes("ssn", tf.getSSNs(1, 0))

		tf.testUpdateOperationalFraudValue("ssns", tf.getSSNs(0, 1), tf.getSSNs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("ssns", tf.getSSNs(0, 1), tf.getSSNs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("ssns", tf.getSSNs(1, 2, 3), tf.getSSNs(1, 2, 3), tf.getSSNs(0))
		tf.testUpdateOperationalFraudValue("ssns", tf.getSSNs(3, 1, 2), tf.getSSNs(3, 1, 2), tf.getSSNs(0))
		tf.testAddSpamPurpose("ssns", tf.getSSNs(3, 1, 2), tf.getSSNs(0))
		tf.testRemoveAllPurposes("ssns", tf.getSSNs(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("ssn_unique", tf.getSSN(0), tf.getSSN(0), nil)
		tf.testUpdateOperationalFraudValue("ssn_unique", tf.getSSN(0), tf.getSSN(0), nil)
		tf.testUpdateOperationalFraudValue("ssn_unique", tf.getSSN(2), tf.getSSN(2), tf.getSSNs(0))
		tf.testUpdateOperationalFraudValue("ssn_unique", tf.getSSN(1), tf.getSSN(1), tf.getSSNs(0, 2))
		tf.testUpdateOperationalFraudValue("ssn_unique", tf.getSSN(2), tf.getSSN(2), tf.getSSNs(0, 2, 1))
		tf.testAddSpamPurpose("ssn_unique", tf.getSSN(2), tf.getSSNs(0, 2, 1))
		tf.testRemoveAllPurposes("ssn_unique", tf.getSSNs(0, 2, 1, 2))
	})

	t.Run("test_string", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("string", tf.getString(0), tf.getString(0), nil)
		tf.testUpdateOperationalFraudValue("string", tf.getString(0), tf.getString(0), nil)
		tf.testUpdateOperationalFraudValue("string", tf.getString(1), tf.getString(1), tf.getStrings(0))
		tf.testUpdateOperationalFraudValue("string", tf.getString(0), tf.getString(0), tf.getStrings(0, 1))
		tf.testUpdateOperationalFraudValue("string", tf.getString(3), tf.getString(3), tf.getStrings(0, 1, 0))
		tf.testAddSpamPurpose("string", tf.getString(3), tf.getStrings(0, 1, 0))
		tf.testRemoveAllPurposes("string", tf.getStrings(0, 1, 0, 3))

		tf.testUpdateOperationalFraudValue("strings", tf.getStrings(0, 1), tf.getStrings(0, 1), nil)
		tf.testUpdateOperationalFraudValue("strings", tf.getStrings(0, 1), tf.getStrings(0, 1), nil)
		tf.testUpdateOperationalFraudValue("strings", tf.getStrings(1, 2, 3), tf.getStrings(1, 2, 3), tf.getStrings(0))
		tf.testUpdateOperationalFraudValue("strings", tf.getStrings(3, 1, 2), tf.getStrings(3, 1, 2), tf.getStrings(0))
		tf.testAddSpamPurpose("strings", tf.getStrings(3, 1, 2), tf.getStrings(0))
		tf.testRemoveAllPurposes("strings", tf.getStrings(0, 3, 1, 2))

		tf.testUpdateOperationalFraudValue("string_unique", tf.getString(0), tf.getString(0), nil)
		tf.testUpdateOperationalFraudValue("string_unique", tf.getString(0), tf.getString(0), nil)
		tf.testUpdateOperationalFraudValue("string_unique", tf.getString(2), tf.getString(2), tf.getStrings(0))
		tf.testUpdateOperationalFraudValue("string_unique", tf.getString(1), tf.getString(1), tf.getStrings(0, 2))
		tf.testUpdateOperationalFraudValue("string_unique", tf.getString(2), tf.getString(2), tf.getStrings(0, 2, 1))
		tf.testAddSpamPurpose("string_unique", tf.getString(2), tf.getStrings(0, 2, 1))
		tf.testRemoveAllPurposes("string_unique", tf.getStrings(0, 2, 1, 2))
	})

	t.Run("test_timestamp", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("timestamp", tf.getTimestamp(0), tf.getTimestamp(0), nil)
		tf.testUpdateOperationalFraudValue("timestamp", tf.getTimestamp(0), tf.getTimestamp(0), nil)
		tf.testUpdateOperationalFraudValue("timestamp", tf.getTimestamp(1), tf.getTimestamp(1), tf.getTimestamps(0))
		tf.testAddSpamPurpose("timestamp", tf.getTimestamp(1), tf.getTimestamps(0))
		tf.testRemoveAllPurposes("timestamp", tf.getTimestamps(0, 1))

		tf.testUpdateOperationalFraudValue("timestamps", tf.getTimestamps(0, 1), tf.getTimestamps(0, 1), nil)
		tf.testUpdateOperationalFraudValue("timestamps", tf.getTimestamps(0, 1), tf.getTimestamps(0, 1), nil)
		tf.testUpdateOperationalFraudValue("timestamps", tf.getTimestamps(1, 2, 3), tf.getTimestamps(1, 2, 3), tf.getTimestamps(0))
		tf.testUpdateOperationalFraudValue("timestamps", tf.getTimestamps(3, 1, 2), tf.getTimestamps(3, 1, 2), tf.getTimestamps(0))
		tf.testAddSpamPurpose("timestamps", tf.getTimestamps(3, 1, 2), tf.getTimestamps(0))
		tf.testRemoveAllPurposes("timestamps", tf.getTimestamps(0, 3, 1, 2))
	})

	t.Run("test_uuid", func(t *testing.T) {
		tf.testUpdateOperationalFraudValue("uuid", tf.getUUID(0), tf.getUUID(0), nil)
		tf.testUpdateOperationalFraudValue("uuid", tf.getUUID(0), tf.getUUID(0), nil)
		tf.testUpdateOperationalFraudValue("uuid", tf.getUUID(1), tf.getUUID(1), tf.getUUIDs(0))
		tf.testUpdateOperationalFraudValue("uuid", tf.getUUID(2), tf.getUUID(2), tf.getUUIDs(0, 1))
		tf.testAddSpamPurpose("uuid", tf.getUUID(2), tf.getUUIDs(0, 1))
		tf.testRemoveAllPurposes("uuid", tf.getUUIDs(0, 1, 2))

		tf.testUpdateOperationalFraudValue("uuids", tf.getUUIDs(0, 1), tf.getUUIDs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("uuids", tf.getUUIDs(0, 1), tf.getUUIDs(0, 1), nil)
		tf.testUpdateOperationalFraudValue("uuids", tf.getUUIDs(1, 2), tf.getUUIDs(1, 2), tf.getUUIDs(0))
		tf.testUpdateOperationalFraudValue("uuids", tf.getUUIDs(0, 1, 0, 2, 0), tf.getUUIDs(0, 1, 0, 2, 0), tf.getUUIDs(0))
		tf.testUpdateOperationalFraudValue("uuids", tf.getUUIDs(0, 2, 1, 0), tf.getUUIDs(0, 2, 1, 0), tf.getUUIDs(0, 0))
		tf.testAddSpamPurpose("uuids", tf.getUUIDs(0, 2, 1, 0), tf.getUUIDs(0, 0))
		tf.testRemoveAllPurposes("uuids", tf.getUUIDs(0, 0, 0, 2, 1, 0))

		tf.testUpdateOperationalFraudValue("uuid_unique", tf.getUUID(0), tf.getUUID(0), nil)
		tf.testUpdateOperationalFraudValue("uuid_unique", tf.getUUID(0), tf.getUUID(0), nil)
		tf.testUpdateOperationalFraudValue("uuid_unique", tf.getUUID(1), tf.getUUID(1), tf.getUUIDs(0))
		tf.testUpdateOperationalFraudValue("uuid_unique", tf.getUUID(3), tf.getUUID(3), tf.getUUIDs(0, 1))
		tf.testUpdateOperationalFraudValue("uuid_unique", tf.getUUID(0), tf.getUUID(0), tf.getUUIDs(0, 1, 3))
		tf.testAddSpamPurpose("uuid_unique", tf.getUUID(0), tf.getUUIDs(0, 1, 3))
		tf.testRemoveAllPurposes("uuid_unique", tf.getUUIDs(0, 1, 3, 0))
	})

	// TODO: test combined accessor, combination of purpose changes and value changes
}
