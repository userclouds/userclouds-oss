package updates_test

import (
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
)

func createBadPartialUpdateColumn(
	tf *idptesthelpers.TestFixture,
	dataType userstore.ResourceID,
	isArray bool,
	constraints userstore.ColumnConstraints,
) {
	tf.T.Helper()

	constraints.PartialUpdates = true
	_, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:        "users",
			Name:         "bad_column",
			DataType:     dataType,
			IsArray:      isArray,
			DefaultValue: "",
			IndexType:    userstore.ColumnIndexTypeIndexed,
			Constraints:  constraints,
		},
		idp.IfNotExists(),
	)
	assert.NotNil(tf.T, err)
}

func TestBadPartialUpdateColumns(t *testing.T) {
	tf := idptesthelpers.NewTestFixture(t)

	compositeDataType, err := tf.IDPClient.CreateDataType(
		tf.Ctx,
		userstore.ColumnDataType{
			Name:        "composite",
			Description: "test composite data type",
			CompositeAttributes: userstore.CompositeAttributes{
				Fields: []userstore.CompositeField{
					{
						Name:     "Foo",
						DataType: datatype.String,
					},
				},
			},
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)

	// not array column
	createBadPartialUpdateColumn(tf, datatype.String, false, userstore.ColumnConstraints{UniqueRequired: true})
	createBadPartialUpdateColumn(tf, datatype.CanonicalAddress, false, userstore.ColumnConstraints{UniqueRequired: true})
	createBadPartialUpdateColumn(tf, userstore.ResourceID{ID: compositeDataType.ID}, false, userstore.ColumnConstraints{UniqueRequired: true})
	// UniqueRequired not enabled
	createBadPartialUpdateColumn(tf, datatype.String, true, userstore.ColumnConstraints{})
	// UniqueRequired and UniqueIDRequired not enabled
	createBadPartialUpdateColumn(tf, datatype.CanonicalAddress, true, userstore.ColumnConstraints{})
	createBadPartialUpdateColumn(tf, userstore.ResourceID{ID: compositeDataType.ID}, true, userstore.ColumnConstraints{})
}

type partialUpdateTestFixture struct {
	testFixture
}

func initPartialUpdateTestFixture(t *testing.T) *partialUpdateTestFixture {
	t.Helper()

	tf := &partialUpdateTestFixture{
		testFixture: newTestFixture(t, "partial_update_"),
	}

	tf.initializePurposes()

	tf.initializeDataTypes()

	// create columns, each an array column with uniqueness and partial updates enabled, and associated accessors and mutators

	tf.createPartialUniqueColumn("addresses", "canonical_address")
	tf.createPartialUniqueColumn("birthdates", "birthdate")
	tf.createPartialUniqueColumn("booleans", "boolean")
	tf.createPartialUniqueColumn("composites", "composite")
	tf.createPartialUniqueColumn("dates", "date")
	tf.createPartialUniqueColumn("e164s", "e164_phonenumber")
	tf.createPartialUniqueColumn("emails", "email")
	tf.createPartialUniqueColumn("integers", "integer")
	tf.createPartialUniqueColumn("phonenumbers", "phonenumber")
	tf.createPartialUniqueColumn("ssns", "ssn")
	tf.createPartialUniqueColumn("strings", "string")
	tf.createPartialUniqueColumn("timestamps", "timestamp")
	tf.createPartialUniqueColumn("uuids", "uuid")

	tf.createPartialUniqueIDColumn("addressesWithIDs", "canonical_address")
	tf.createPartialImmutableUniqueIDColumn("immutableAddressesWithIDs", "canonical_address")
	tf.createPartialUniqueUniqueIDColumn("uniqueAddressesWithIDs", "canonical_address")
	tf.createPartialImmutableUniqueUniqueIDColumn("immutableUniqueAddressesWithIDs", "canonical_address")

	tf.createPartialUniqueIDColumn("compositesWithIDs", "composite_with_id")
	tf.createPartialImmutableUniqueIDColumn("immutableCompositesWithIDs", "composite_with_id")
	tf.createPartialUniqueUniqueIDColumn("uniqueCompositesWithIDs", "composite_with_id")
	tf.createPartialImmutableUniqueUniqueIDColumn("immutableUniqueCompositesWithIDs", "composite_with_id")

	tf.createCombinedAccessors()

	tf.initializeTestUser()

	return tf
}

func (tf *partialUpdateTestFixture) createPartialUniqueColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{
			PartialUpdates: true,
			UniqueRequired: true,
		},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *partialUpdateTestFixture) createPartialUniqueIDColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{
			PartialUpdates:   true,
			UniqueIDRequired: true,
		},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *partialUpdateTestFixture) createPartialImmutableUniqueIDColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{
			ImmutableRequired: true,
			PartialUpdates:    true,
			UniqueIDRequired:  true,
		},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *partialUpdateTestFixture) createPartialUniqueUniqueIDColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{
			PartialUpdates:   true,
			UniqueRequired:   true,
			UniqueIDRequired: true,
		},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *partialUpdateTestFixture) createPartialImmutableUniqueUniqueIDColumn(columnName string, dataTypeName string) {
	tf.T.Helper()
	tf.createConstrainedColumn(
		columnName,
		dataTypeName,
		userstore.ColumnConstraints{
			PartialUpdates:    true,
			ImmutableRequired: true,
			UniqueRequired:    true,
			UniqueIDRequired:  true,
		},
		true,
		userstore.ColumnIndexTypeIndexed,
	)
}

func (tf *partialUpdateTestFixture) addValues(
	columnName string,
	valueAdditions any,
	purposeAdditions []userstore.ResourceID,
) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			ValueAdditions:   valueAdditions,
			PurposeAdditions: purposeAdditions,
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func (tf *partialUpdateTestFixture) badMutation(
	columnName string,
	value idp.ValueAndPurposes,
) {
	tf.T.Helper()
	_, err := tf.executeMutator(columnName, value)
	assert.NotNil(tf.T, err)
}

func (tf *partialUpdateTestFixture) deleteValues(
	columnName string,
	valueDeletions any,
	purposeDeletions []userstore.ResourceID,
) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			ValueDeletions:   valueDeletions,
			PurposeDeletions: purposeDeletions,
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func (tf *partialUpdateTestFixture) executeMutator(
	columnName string,
	vap idp.ValueAndPurposes,
) (*idp.ExecuteMutatorResponse, error) {
	tf.T.Helper()
	tf.mutatorsLock.Lock()
	defer tf.mutatorsLock.Unlock()

	var purposeAdditions []userstore.ResourceID
	for _, pa := range vap.PurposeAdditions {
		purposeAdditions = append(
			purposeAdditions,
			userstore.ResourceID{ID: pa.ID, Name: tf.getPurposeName(pa.Name)},
		)
	}
	vap.PurposeAdditions = purposeAdditions

	var purposeDeletions []userstore.ResourceID
	for _, pa := range vap.PurposeDeletions {
		purposeDeletions = append(
			purposeDeletions,
			userstore.ResourceID{ID: pa.ID, Name: tf.getPurposeName(pa.Name)},
		)
	}
	vap.PurposeDeletions = purposeDeletions

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

func (tf *partialUpdateTestFixture) failAddOperationalFraudValues(
	columnName string,
	valueAdditions any,
) {
	tf.T.Helper()

	_, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			ValueAdditions: valueAdditions,
			PurposeAdditions: []userstore.ResourceID{
				{Name: "operational"},
				{Name: "fraud"},
			},
		},
	)
	assert.NotNil(tf.T, err)
}

func (tf *partialUpdateTestFixture) testAddOperationalFraudValues(
	columnName string,
	valueAdditions any,
	expectedLiveValue any,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.addValues(
		columnName,
		valueAdditions,
		[]userstore.ResourceID{
			{Name: "operational"},
			{Name: "fraud"},
		},
	)
	tf.lookupValue(
		columnName,
		[]string{"operational", "fraud"},
		map[string]any{"operational": expectedLiveValue, "fraud": expectedLiveValue},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *partialUpdateTestFixture) testDeleteOperationalFraudValues(
	columnName string,
	valueDeletions any,
	expectedLiveValue any,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.deleteValues(
		columnName,
		valueDeletions,
		[]userstore.ResourceID{
			{Name: "operational"},
			{Name: "fraud"},
		},
	)
	tf.lookupValue(
		columnName,
		[]string{"operational", "fraud"},
		map[string]any{"operational": expectedLiveValue, "fraud": expectedLiveValue},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *partialUpdateTestFixture) testUpdateOperationalFraudValues(
	columnName string,
	valueAdditions any,
	valueDeletions any,
	expectedLiveValue any,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.updateValues(
		columnName,
		valueAdditions,
		[]userstore.ResourceID{
			{Name: "operational"},
			{Name: "fraud"},
		},
		valueDeletions,
		[]userstore.ResourceID{
			{Name: "operational"},
			{Name: "fraud"},
		},
	)
	tf.lookupValue(
		columnName,
		[]string{"operational", "fraud"},
		map[string]any{"operational": expectedLiveValue, "fraud": expectedLiveValue},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *partialUpdateTestFixture) testRemoveAllPurposes(
	columnName string,
	expectedSoftDeletedFraudValue any,
) {
	tf.T.Helper()

	tf.deleteValues(
		columnName,
		idp.MutatorColumnCurrentValue,
		[]userstore.ResourceID{
			{Name: "analytics"},
			{Name: "operational"},
			{Name: "fraud"},
		},
	)
	tf.lookupValue(
		columnName,
		[]string{"analytics", "operational", "fraud"},
		map[string]any{},
		map[string]any{"fraud": expectedSoftDeletedFraudValue},
	)
}

func (tf *partialUpdateTestFixture) testUpdateValues(
	columnName string,
	purpose string,
	valueAdditions any,
	valueDeletions any,
	expectedLiveValue any,
) {
	tf.T.Helper()

	if valueAdditions != nil {
		if valueDeletions != nil {
			tf.updateValues(
				columnName,
				valueAdditions,
				[]userstore.ResourceID{{Name: purpose}},
				valueDeletions,
				[]userstore.ResourceID{{Name: purpose}},
			)
		} else {
			tf.addValues(
				columnName,
				valueAdditions,
				[]userstore.ResourceID{{Name: purpose}},
			)
		}
	} else {
		tf.deleteValues(
			columnName,
			valueDeletions,
			[]userstore.ResourceID{{Name: purpose}},
		)
	}

	tf.lookupValue(
		columnName,
		[]string{purpose},
		map[string]any{purpose: expectedLiveValue},
		map[string]any{},
	)
}

func (tf *partialUpdateTestFixture) updateValues(
	columnName string,
	valueAdditions any,
	purposeAdditions []userstore.ResourceID,
	valueDeletions any,
	purposeDeletions []userstore.ResourceID,
) {
	tf.T.Helper()

	resp, err := tf.executeMutator(
		columnName,
		idp.ValueAndPurposes{
			ValueAdditions:   valueAdditions,
			PurposeAdditions: purposeAdditions,
			ValueDeletions:   valueDeletions,
			PurposeDeletions: purposeDeletions,
		},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.UserIDs), 1)
	assert.Equal(tf.T, resp.UserIDs[0], tf.testUserID)
}

func TestPartialUpdates(t *testing.T) {
	tf := initPartialUpdateTestFixture(t)

	t.Run("test_bad_mutations", func(t *testing.T) {
		badPurpose := []userstore.ResourceID{{Name: "unknown"}}
		duplicatePurposes := []userstore.ResourceID{{Name: "operational"}, {Name: "operational"}}
		onePurpose := []userstore.ResourceID{{Name: "operational"}}

		duplicateValues := []string{"foo", "foo"}
		oneValue := []string{"foo"}

		tf.badMutation("strings", idp.ValueAndPurposes{})
		tf.badMutation("strings", idp.ValueAndPurposes{Value: oneValue, PurposeAdditions: onePurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{PurposeAdditions: onePurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{PurposeDeletions: onePurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueAdditions: oneValue})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueAdditions: oneValue, PurposeAdditions: badPurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueAdditions: oneValue, PurposeAdditions: duplicatePurposes})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueDeletions: oneValue, PurposeDeletions: badPurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueDeletions: oneValue, PurposeDeletions: duplicatePurposes})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueAdditions: duplicateValues, PurposeAdditions: onePurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueDeletions: duplicateValues, PurposeDeletions: onePurpose})
		tf.badMutation("strings", idp.ValueAndPurposes{ValueDeletions: duplicateValues})
	})

	t.Run("test_addresses", func(t *testing.T) {
		tf.testRemoveAllPurposes("addresses", nil)
		tf.testAddOperationalFraudValues("addresses", tf.getAddresses(0, 2, 1), tf.getAddresses(0, 2, 1), nil)
		tf.testUpdateValues("addresses", "analytics", tf.getAddresses(3, 0), nil, tf.getAddresses(0, 3))
		tf.testUpdateValues("addresses", "support", tf.getAddresses(2, 1), nil, tf.getAddresses(2, 1))
		tf.testUpdateValues("addresses", "analytics", tf.getAddresses(1), tf.getAddresses(0), tf.getAddresses(1, 3))
		tf.testUpdateValues("addresses", "support", nil, tf.getAddresses(2), tf.getAddresses(1))
		tf.testUpdateOperationalFraudValues("addresses", tf.getAddresses(3), tf.getAddresses(2), tf.getAddresses(0, 1, 3), tf.getAddresses(2))
		tf.testDeleteOperationalFraudValues("addresses", tf.getAddresses(3), tf.getAddresses(0, 1), tf.getAddresses(2, 3))
		tf.testRemoveAllPurposes("addresses", tf.getAddresses(2, 3, 0, 1))

		tf.testRemoveAllPurposes("addressesWithIDs", nil)
		tf.testAddOperationalFraudValues("addressesWithIDs", tf.getAddressesWithIDs(0, 2, 1), tf.getAddressesWithIDs(0, 2, 1), nil)
		tf.testUpdateValues("addressesWithIDs", "analytics", tf.getAddressesWithIDs(3, 0), nil, tf.getAddressesWithIDs(0, 3))
		tf.testUpdateValues("addressesWithIDs", "support", tf.getAddressesWithIDs(2, 1), nil, tf.getAddressesWithIDs(2, 1))
		tf.testUpdateValues("addressesWithIDs", "analytics", tf.getAddressesWithIDs(1), tf.getAddressesWithIDs(0), tf.getAddressesWithIDs(1, 3))
		tf.testUpdateValues("addressesWithIDs", "support", nil, tf.getAddressesWithIDs(2), tf.getAddressesWithIDs(1))
		tf.testAddOperationalFraudValues("addressesWithIDs", tf.getAddressesWithIDs(4, 5), tf.getAddressesWithIDs(4, 2, 5), tf.getAddressesWithIDs(0, 1))
		tf.testUpdateOperationalFraudValues("addressesWithIDs", tf.getAddressesWithIDs(3), tf.getAddressesWithIDs(2), tf.getAddressesWithIDs(4, 5, 3), tf.getAddressesWithIDs(0, 1, 2))
		tf.testDeleteOperationalFraudValues("addressesWithIDs", tf.getAddressesWithIDs(3), tf.getAddressesWithIDs(4, 5), tf.getAddressesWithIDs(0, 1, 2, 3))
		tf.testRemoveAllPurposes("addressesWithIDs", tf.getAddressesWithIDs(0, 1, 2, 3, 4, 5))

		tf.testAddOperationalFraudValues("immutableAddressesWithIDs", tf.getAddressesWithIDs(0, 5), tf.getAddressesWithIDs(0, 5), nil)
		tf.failAddOperationalFraudValues("immutableAddressesWithIDs", tf.getAddressesWithIDs(4))

		tf.testAddOperationalFraudValues("uniqueAddressesWithIDs", tf.getAddressesWithIDs(0), tf.getAddressesWithIDs(0), nil)
		tf.failAddOperationalFraudValues("uniqueAddressesWithIDs", tf.getAddressesWithIDs(5))
		tf.testAddOperationalFraudValues("uniqueAddressesWithIDs", tf.getAddressesWithIDs(4), tf.getAddressesWithIDs(4), tf.getAddressesWithIDs(0))

		tf.testAddOperationalFraudValues("immutableUniqueAddressesWithIDs", tf.getAddressesWithIDs(0), tf.getAddressesWithIDs(0), nil)
		tf.failAddOperationalFraudValues("immutableUniqueAddressesWithIDs", tf.getAddressesWithIDs(4))
		tf.failAddOperationalFraudValues("immutableUniqueAddressesWithIDs", tf.getAddressesWithIDs(5))
	})

	t.Run("test_booleans", func(t *testing.T) {
		tf.testRemoveAllPurposes("booleans", nil)
		tf.testAddOperationalFraudValues("booleans", tf.getBools(0, 1), tf.getBools(0, 1), nil)
		tf.testUpdateOperationalFraudValues("booleans", tf.getBools(1), tf.getBools(0), tf.getBools(1), tf.getBools(0))
		tf.testAddOperationalFraudValues("booleans", tf.getBools(0), tf.getBools(1, 0), tf.getBools(0))
		tf.testDeleteOperationalFraudValues("booleans", tf.getBools(1), tf.getBools(0), tf.getBools(0, 1))
		tf.testUpdateValues("booleans", "analytics", tf.getBools(1), nil, tf.getBools(1))
		tf.testUpdateValues("booleans", "support", tf.getBools(0), nil, tf.getBools(0))
		tf.testUpdateValues("booleans", "analytics", tf.getBools(0), tf.getBools(1), tf.getBools(0))
		tf.testUpdateValues("booleans", "support", tf.getBools(1), nil, tf.getBools(0, 1))
		tf.testRemoveAllPurposes("booleans", tf.getBools(0, 1, 0))
	})

	t.Run("test_composites", func(t *testing.T) {
		tf.testRemoveAllPurposes("composites", nil)
		tf.testAddOperationalFraudValues("composites", tf.getComposites(0, 2, 1), tf.getComposites(0, 2, 1), nil)
		tf.testUpdateValues("composites", "analytics", tf.getComposites(3, 0), nil, tf.getComposites(0, 3))
		tf.testUpdateValues("composites", "support", tf.getComposites(2, 1), nil, tf.getComposites(2, 1))
		tf.testUpdateValues("composites", "analytics", tf.getComposites(1), tf.getComposites(0), tf.getComposites(1, 3))
		tf.testUpdateValues("composites", "support", nil, tf.getComposites(2), tf.getComposites(1))
		tf.testUpdateOperationalFraudValues("composites", tf.getComposites(3), tf.getComposites(2), tf.getComposites(0, 1, 3), tf.getComposites(2))
		tf.testDeleteOperationalFraudValues("composites", tf.getComposites(3), tf.getComposites(0, 1), tf.getComposites(2, 3))
		tf.testRemoveAllPurposes("composites", tf.getComposites(2, 3, 0, 1))

		tf.testRemoveAllPurposes("compositesWithIDs", nil)
		tf.testAddOperationalFraudValues("compositesWithIDs", tf.getCompositesWithIDs(0, 2, 1), tf.getCompositesWithIDs(0, 2, 1), nil)
		tf.testUpdateValues("compositesWithIDs", "analytics", tf.getCompositesWithIDs(3, 0), nil, tf.getCompositesWithIDs(0, 3))
		tf.testUpdateValues("compositesWithIDs", "support", tf.getCompositesWithIDs(2, 1), nil, tf.getCompositesWithIDs(2, 1))
		tf.testUpdateValues("compositesWithIDs", "analytics", tf.getCompositesWithIDs(1), tf.getCompositesWithIDs(0), tf.getCompositesWithIDs(1, 3))
		tf.testUpdateValues("compositesWithIDs", "support", nil, tf.getCompositesWithIDs(2), tf.getCompositesWithIDs(1))
		tf.testAddOperationalFraudValues("compositesWithIDs", tf.getCompositesWithIDs(4, 5), tf.getCompositesWithIDs(4, 2, 5), tf.getCompositesWithIDs(0, 1))
		tf.testUpdateOperationalFraudValues("compositesWithIDs", tf.getCompositesWithIDs(3), tf.getCompositesWithIDs(2), tf.getCompositesWithIDs(4, 5, 3), tf.getCompositesWithIDs(0, 1, 2))
		tf.testDeleteOperationalFraudValues("compositesWithIDs", tf.getCompositesWithIDs(3), tf.getCompositesWithIDs(4, 5), tf.getCompositesWithIDs(0, 1, 2, 3))
		tf.testRemoveAllPurposes("compositesWithIDs", tf.getCompositesWithIDs(0, 1, 2, 3, 4, 5))

		tf.testAddOperationalFraudValues("immutableCompositesWithIDs", tf.getCompositesWithIDs(0, 5), tf.getCompositesWithIDs(0, 5), nil)
		tf.failAddOperationalFraudValues("immutableCompositesWithIDs", tf.getCompositesWithIDs(4))

		tf.testAddOperationalFraudValues("uniqueCompositesWithIDs", tf.getCompositesWithIDs(0), tf.getCompositesWithIDs(0), nil)
		tf.failAddOperationalFraudValues("uniqueCompositesWithIDs", tf.getCompositesWithIDs(5))
		tf.testAddOperationalFraudValues("uniqueCompositesWithIDs", tf.getCompositesWithIDs(4), tf.getCompositesWithIDs(4), tf.getCompositesWithIDs(0))

		tf.testAddOperationalFraudValues("immutableUniqueCompositesWithIDs", tf.getCompositesWithIDs(0), tf.getCompositesWithIDs(0), nil)
		tf.failAddOperationalFraudValues("immutableUniqueCompositesWithIDs", tf.getCompositesWithIDs(4))
		tf.failAddOperationalFraudValues("immutableUniqueCompositesWithIDs", tf.getCompositesWithIDs(5))
	})

	t.Run("test_dates", func(t *testing.T) {
		tf.testRemoveAllPurposes("dates", nil)
		tf.testAddOperationalFraudValues("dates", tf.getDates(0, 2, 1), tf.getDateStrs(0, 2, 1), nil)
		tf.testUpdateValues("dates", "analytics", tf.getDates(3, 0), nil, tf.getDateStrs(0, 3))
		tf.testUpdateValues("dates", "support", tf.getDates(2, 1), nil, tf.getDateStrs(2, 1))
		tf.testUpdateValues("dates", "analytics", tf.getDates(1), tf.getDates(0), tf.getDateStrs(1, 3))
		tf.testUpdateValues("dates", "support", nil, tf.getDates(2), tf.getDateStrs(1))
		tf.testUpdateOperationalFraudValues("dates", tf.getDates(3), tf.getDates(2), tf.getDateStrs(0, 1, 3), tf.getDateStrs(2))
		tf.testDeleteOperationalFraudValues("dates", tf.getDates(3), tf.getDateStrs(0, 1), tf.getDateStrs(2, 3))
		tf.testRemoveAllPurposes("dates", tf.getDateStrs(2, 3, 0, 1))
	})

	t.Run("test_e164s", func(t *testing.T) {
		tf.testRemoveAllPurposes("e164s", nil)
		tf.testAddOperationalFraudValues("e164s", tf.getE164s(0, 2, 1), tf.getE164s(0, 2, 1), nil)
		tf.testUpdateValues("e164s", "analytics", tf.getE164s(3, 0), nil, tf.getE164s(0, 3))
		tf.testUpdateValues("e164s", "support", tf.getE164s(2, 1), nil, tf.getE164s(2, 1))
		tf.testUpdateValues("e164s", "analytics", tf.getE164s(1), tf.getE164s(0), tf.getE164s(1, 3))
		tf.testUpdateValues("e164s", "support", nil, tf.getE164s(2), tf.getE164s(1))
		tf.testUpdateOperationalFraudValues("e164s", tf.getE164s(3), tf.getE164s(2), tf.getE164s(0, 1, 3), tf.getE164s(2))
		tf.testDeleteOperationalFraudValues("e164s", tf.getE164s(3), tf.getE164s(0, 1), tf.getE164s(2, 3))
		tf.testRemoveAllPurposes("e164s", tf.getE164s(2, 3, 0, 1))
	})

	t.Run("test_emails", func(t *testing.T) {
		tf.testRemoveAllPurposes("emails", nil)
		tf.testAddOperationalFraudValues("emails", tf.getEmails(0, 2, 1), tf.getEmails(0, 2, 1), nil)
		tf.testUpdateValues("emails", "analytics", tf.getEmails(3, 0), nil, tf.getEmails(0, 3))
		tf.testUpdateValues("emails", "support", tf.getEmails(2, 1), nil, tf.getEmails(2, 1))
		tf.testUpdateValues("emails", "analytics", tf.getEmails(1), tf.getEmails(0), tf.getEmails(1, 3))
		tf.testUpdateValues("emails", "support", nil, tf.getEmails(2), tf.getEmails(1))
		tf.testUpdateOperationalFraudValues("emails", tf.getEmails(3), tf.getEmails(2), tf.getEmails(0, 1, 3), tf.getEmails(2))
		tf.testDeleteOperationalFraudValues("emails", tf.getEmails(3), tf.getEmails(0, 1), tf.getEmails(2, 3))
		tf.testRemoveAllPurposes("emails", tf.getEmails(2, 3, 0, 1))
	})

	t.Run("test_integers", func(t *testing.T) {
		tf.testRemoveAllPurposes("integers", nil)
		tf.testAddOperationalFraudValues("integers", tf.getInts(0, 2, 1), tf.getInts(0, 2, 1), nil)
		tf.testUpdateValues("integers", "analytics", tf.getInts(3, 0), nil, tf.getInts(0, 3))
		tf.testUpdateValues("integers", "support", tf.getInts(2, 1), nil, tf.getInts(2, 1))
		tf.testUpdateValues("integers", "analytics", tf.getInts(1), tf.getInts(0), tf.getInts(1, 3))
		tf.testUpdateValues("integers", "support", nil, tf.getInts(2), tf.getInts(1))
		tf.testUpdateOperationalFraudValues("integers", tf.getInts(3), tf.getInts(2), tf.getInts(0, 1, 3), tf.getInts(2))
		tf.testDeleteOperationalFraudValues("integers", tf.getInts(3), tf.getInts(0, 1), tf.getInts(2, 3))
		tf.testRemoveAllPurposes("integers", tf.getInts(2, 3, 0, 1))
	})

	t.Run("test_phonenumbers", func(t *testing.T) {
		tf.testRemoveAllPurposes("phonenumbers", nil)
		tf.testAddOperationalFraudValues("phonenumbers", tf.getPhones(0, 2, 1), tf.getPhones(0, 2, 1), nil)
		tf.testUpdateValues("phonenumbers", "analytics", tf.getPhones(3, 0), nil, tf.getPhones(0, 3))
		tf.testUpdateValues("phonenumbers", "support", tf.getPhones(2, 1), nil, tf.getPhones(2, 1))
		tf.testUpdateValues("phonenumbers", "analytics", tf.getPhones(1), tf.getPhones(0), tf.getPhones(1, 3))
		tf.testUpdateValues("phonenumbers", "support", nil, tf.getPhones(2), tf.getPhones(1))
		tf.testUpdateOperationalFraudValues("phonenumbers", tf.getPhones(3), tf.getPhones(2), tf.getPhones(0, 1, 3), tf.getPhones(2))
		tf.testDeleteOperationalFraudValues("phonenumbers", tf.getPhones(3), tf.getPhones(0, 1), tf.getPhones(2, 3))
		tf.testRemoveAllPurposes("phonenumbers", tf.getPhones(2, 3, 0, 1))
	})

	t.Run("test_ssns", func(t *testing.T) {
		tf.testRemoveAllPurposes("ssns", nil)
		tf.testAddOperationalFraudValues("ssns", tf.getSSNs(0, 2, 1), tf.getSSNs(0, 2, 1), nil)
		tf.testUpdateValues("ssns", "analytics", tf.getSSNs(3, 0), nil, tf.getSSNs(0, 3))
		tf.testUpdateValues("ssns", "support", tf.getSSNs(2, 1), nil, tf.getSSNs(2, 1))
		tf.testUpdateValues("ssns", "analytics", tf.getSSNs(1), tf.getSSNs(0), tf.getSSNs(1, 3))
		tf.testUpdateValues("ssns", "support", nil, tf.getSSNs(2), tf.getSSNs(1))
		tf.testUpdateOperationalFraudValues("ssns", tf.getSSNs(3), tf.getSSNs(2), tf.getSSNs(0, 1, 3), tf.getSSNs(2))
		tf.testDeleteOperationalFraudValues("ssns", tf.getSSNs(3), tf.getSSNs(0, 1), tf.getSSNs(2, 3))
		tf.testRemoveAllPurposes("ssns", tf.getSSNs(2, 3, 0, 1))
	})

	t.Run("test_strings", func(t *testing.T) {
		tf.testRemoveAllPurposes("strings", nil)
		tf.testAddOperationalFraudValues("strings", tf.getStrings(0, 2, 1), tf.getStrings(0, 2, 1), nil)
		tf.testUpdateValues("strings", "analytics", tf.getStrings(3, 0), nil, tf.getStrings(0, 3))
		tf.testUpdateValues("strings", "support", tf.getStrings(2, 1), nil, tf.getStrings(2, 1))
		tf.testUpdateValues("strings", "analytics", tf.getStrings(1), tf.getStrings(0), tf.getStrings(1, 3))
		tf.testUpdateValues("strings", "support", nil, tf.getStrings(2), tf.getStrings(1))
		tf.testUpdateOperationalFraudValues("strings", tf.getStrings(3), tf.getStrings(2), tf.getStrings(0, 1, 3), tf.getStrings(2))
		tf.testDeleteOperationalFraudValues("strings", tf.getStrings(3), tf.getStrings(0, 1), tf.getStrings(2, 3))
		tf.testRemoveAllPurposes("strings", tf.getStrings(2, 3, 0, 1))
	})

	t.Run("test_timestamps", func(t *testing.T) {
		tf.testRemoveAllPurposes("timestamps", nil)
		tf.testAddOperationalFraudValues("timestamps", tf.getTimestamps(0, 2, 1), tf.getTimestamps(0, 2, 1), nil)
		tf.testUpdateValues("timestamps", "analytics", tf.getTimestamps(3, 0), nil, tf.getTimestamps(0, 3))
		tf.testUpdateValues("timestamps", "support", tf.getTimestamps(2, 1), nil, tf.getTimestamps(2, 1))
		tf.testUpdateValues("timestamps", "analytics", tf.getTimestamps(1), tf.getTimestamps(0), tf.getTimestamps(1, 3))
		tf.testUpdateValues("timestamps", "support", nil, tf.getTimestamps(2), tf.getTimestamps(1))
		tf.testUpdateOperationalFraudValues("timestamps", tf.getTimestamps(3), tf.getTimestamps(2), tf.getTimestamps(0, 1, 3), tf.getTimestamps(2))
		tf.testDeleteOperationalFraudValues("timestamps", tf.getTimestamps(3), tf.getTimestamps(0, 1), tf.getTimestamps(2, 3))
		tf.testRemoveAllPurposes("timestamps", tf.getTimestamps(2, 3, 0, 1))
	})

	t.Run("test_uuids", func(t *testing.T) {
		tf.testRemoveAllPurposes("uuids", nil)
		tf.testAddOperationalFraudValues("uuids", tf.getUUIDs(0, 2, 1), tf.getUUIDs(0, 2, 1), nil)
		tf.testUpdateValues("uuids", "analytics", tf.getUUIDs(3, 0), nil, tf.getUUIDs(0, 3))
		tf.testUpdateValues("uuids", "support", tf.getUUIDs(2, 1), nil, tf.getUUIDs(2, 1))
		tf.testUpdateValues("uuids", "analytics", tf.getUUIDs(1), tf.getUUIDs(0), tf.getUUIDs(1, 3))
		tf.testUpdateValues("uuids", "support", nil, tf.getUUIDs(2), tf.getUUIDs(1))
		tf.testUpdateOperationalFraudValues("uuids", tf.getUUIDs(3), tf.getUUIDs(2), tf.getUUIDs(0, 1, 3), tf.getUUIDs(2))
		tf.testDeleteOperationalFraudValues("uuids", tf.getUUIDs(3), tf.getUUIDs(0, 1), tf.getUUIDs(2, 3))
		tf.testRemoveAllPurposes("uuids", tf.getUUIDs(2, 3, 0, 1))
	})

	t.Run("test_partial_and_full_updates", func(t *testing.T) {
		m, err := tf.CreateMutator(
			"partial_update_strings_and_name_mutator",
			policy.AccessPolicyAllowAll.ID,
			[]string{"name", "partial_update_strings"},
			[]uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID},
		)
		assert.NoErr(tf.T, err)

		a, err := tf.CreateLiveAccessor(
			"partial_update_strings_and_name_accessor",
			policy.AccessPolicyAllowAll.ID,
			[]string{"name", "partial_update_strings"},
			[]uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID},
			[]string{"marketing"},
		)
		assert.NoErr(tf.T, err)

		mresp, err := tf.IDPClient.ExecuteMutator(
			tf.Ctx,
			m.ID,
			policy.ClientContext{},
			[]any{tf.testUserID},
			map[string]idp.ValueAndPurposes{
				"name": {
					Value:            "foobar",
					PurposeAdditions: []userstore.ResourceID{{Name: "marketing"}},
				},
				"partial_update_strings": {
					ValueAdditions:   []string{"dopey", "sneezy", "sleepy"},
					PurposeAdditions: []userstore.ResourceID{{Name: "marketing"}},
				},
			},
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(mresp.UserIDs), 1)
		assert.Equal(tf.T, mresp.UserIDs[0], tf.testUserID)

		aresp, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			a.ID,
			policy.ClientContext{},
			[]any{tf.testUserID},
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(aresp.Data), 1)
		assert.Equal(tf.T, `{"name":"foobar","partial_update_strings":"[\"dopey\",\"sneezy\",\"sleepy\"]"}`, aresp.Data[0])
	})
}
