package idp_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
)

func assertHttpStatus(t *testing.T, resp any, err error, status int) {
	t.Helper()

	t.Logf("got response: %+v, error: %+v", resp, err)
	var e jsonclient.Error
	assert.True(t, errors.As(err, &e))
	assert.Equal(t, e.Code(), status)
}

func doBasicTestForColumns(t *testing.T, tf *idptesthelpers.TestFixture, dlc userstore.DataLifeCycleState) {
	t.Helper()

	col := tf.CreateValidColumn(uniqueName("test_column"), datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

	// Create
	createResp, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, dlc, col.ID, idp.UpdateColumnRetentionDurationsRequest{
		RetentionDurations: []idp.ColumnRetentionDuration{{
			ColumnID:     col.ID,
			PurposeID:    constants.OperationalPurposeID,
			DurationType: dlc,
			Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
		}},
	})
	assert.NoErr(t, err)
	t.Logf("created setting: %+v", createResp.RetentionDurations[0])
	assert.NotNil(t, createResp.RetentionDurations[0].ID)
	assert.False(t, createResp.RetentionDurations[0].UseDefault)
	assert.Equal(t, createResp.RetentionDurations[0].ColumnID, col.ID)
	assert.Equal(t, createResp.RetentionDurations[0].PurposeID, constants.OperationalPurposeID)
	assert.Equal(t, createResp.RetentionDurations[0].DurationType, dlc)
	assert.Equal(t, createResp.RetentionDurations[0].Duration, idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1})

	// List
	listResp, err := tf.IDPClient.GetColumnRetentionDurationsForColumn(tf.Ctx, dlc, col.ID)
	assert.NoErr(t, err)
	for _, crrd := range createResp.RetentionDurations {
		if crrd.PurposeID == constants.OperationalPurposeID {
			for _, lrrd := range listResp.RetentionDurations {
				if lrrd.PurposeID == constants.OperationalPurposeID {
					assert.Equal(t, crrd, lrrd)
				}
			}
		}
		// TODO: should check the rest are defaults
	}

	// Read
	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, createResp.RetentionDurations[0].ID)
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDurations[0], readResp.RetentionDuration)

	// Update via POST
	setting := createResp.RetentionDurations[0]
	setting.Duration.Duration += 1
	postResp, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, dlc, col.ID, idp.UpdateColumnRetentionDurationsRequest{
		RetentionDurations: []idp.ColumnRetentionDuration{setting},
	})
	assert.NoErr(t, err)
	setting.Version += 1
	assert.Equal(t, postResp.RetentionDurations[0], setting)
	readRespAfterPost, err := tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, setting.ID)
	assert.NoErr(t, err)
	assert.Equal(t, readRespAfterPost.RetentionDuration, setting)
	t.Logf("updated setting: %+v", setting)

	// Update via PUT
	setting.Duration.Duration += 1
	putResp, err := tf.IDPClient.UpdateSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, setting.ID, setting)
	assert.NoErr(t, err)
	setting.Version += 1
	assert.Equal(t, putResp.RetentionDuration, setting)
	readRespAfterPut, err := tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, setting.ID)
	assert.NoErr(t, err)
	assert.Equal(t, readRespAfterPut.RetentionDuration, setting)

	// Delete
	err = tf.IDPClient.DeleteColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, createResp.RetentionDurations[0].ID)
	assert.NoErr(t, err)
	listResp2, err := tf.IDPClient.GetColumnRetentionDurationsForColumn(tf.Ctx, dlc, col.ID)
	assert.NoErr(t, err)
	assert.True(t, listResp2.RetentionDurations[0].ID.IsNil())
	assert.True(t, listResp2.RetentionDurations[0].UseDefault)

	// GET should now give 404
	readResp2, err := tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, createResp.RetentionDurations[0].ID)
	assertHttpStatus(t, readResp2, err, http.StatusNotFound)
}

func doBasicTestForPurposes(t *testing.T, tf *idptesthelpers.TestFixture, dlc userstore.DataLifeCycleState) {
	t.Helper()

	dlcStr, err := dlc.MarshalText()
	assert.NoErr(t, err)
	purpose, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
		Name: uniqueName("test_purpose") + string(dlcStr),
	})
	assert.NoErr(t, err)

	// Create
	createResp, err := tf.IDPClient.CreateColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, idp.ColumnRetentionDuration{
		PurposeID:    purpose.ID,
		DurationType: dlc,
		Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
	})
	assert.NoErr(t, err)
	t.Logf("created setting: %+v", createResp.RetentionDuration)
	assert.NotNil(t, createResp.RetentionDuration.ID)
	assert.False(t, createResp.RetentionDuration.UseDefault)
	assert.True(t, createResp.RetentionDuration.ColumnID.IsNil())
	assert.Equal(t, createResp.RetentionDuration.PurposeID, purpose.ID)
	assert.Equal(t, createResp.RetentionDuration.DurationType, dlc)
	assert.Equal(t, createResp.RetentionDuration.Duration, idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1})

	// List
	listResp, err := tf.IDPClient.GetColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID)
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration, listResp.RetentionDuration)

	// Read
	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, createResp.RetentionDuration.ID)
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration, readResp.RetentionDuration)

	// Update via PUT
	setting := createResp.RetentionDuration
	setting.Duration.Duration += 1
	putResp, err := tf.IDPClient.UpdateSpecificColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, setting.ID, setting)
	assert.NoErr(t, err)
	setting.Version += 1
	assert.Equal(t, putResp.RetentionDuration, setting)
	readRespAfterPut, err := tf.IDPClient.GetSpecificColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, setting.ID)
	assert.NoErr(t, err)
	assert.Equal(t, readRespAfterPut.RetentionDuration, setting)

	// Delete
	err = tf.IDPClient.DeleteColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, createResp.RetentionDuration.ID)
	assert.NoErr(t, err)
	listResp2, err := tf.IDPClient.GetColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID)
	assert.NoErr(t, err)
	assert.True(t, listResp2.RetentionDuration.ID.IsNil())
	assert.True(t, listResp2.RetentionDuration.UseDefault)

	// GET should now give 404
	readResp2, err := tf.IDPClient.GetSpecificColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, createResp.RetentionDuration.ID)
	assertHttpStatus(t, readResp2, err, http.StatusNotFound)
}

func doBasicTestForTenants(t *testing.T, tf *idptesthelpers.TestFixture, dlc userstore.DataLifeCycleState) {
	// Create
	createResp, err := tf.IDPClient.CreateColumnRetentionDurationForTenant(tf.Ctx, dlc, idp.ColumnRetentionDuration{
		DurationType: dlc,
		Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
	})
	assert.NoErr(t, err)
	t.Logf("created setting: %+v", createResp.RetentionDuration)
	assert.NotNil(t, createResp.RetentionDuration.ID)
	assert.False(t, createResp.RetentionDuration.UseDefault)
	assert.True(t, createResp.RetentionDuration.ColumnID.IsNil())
	assert.True(t, createResp.RetentionDuration.PurposeID.IsNil())
	assert.Equal(t, createResp.RetentionDuration.DurationType, dlc)
	assert.Equal(t, createResp.RetentionDuration.Duration, idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1})

	// List
	listResp, err := tf.IDPClient.GetColumnRetentionDurationForTenant(tf.Ctx, dlc)
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration, listResp.RetentionDuration)

	// Read
	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForTenant(tf.Ctx, dlc, createResp.RetentionDuration.ID)
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration, readResp.RetentionDuration)

	// Update via PUT
	setting := createResp.RetentionDuration
	setting.Duration.Duration += 1
	putResp, err := tf.IDPClient.UpdateSpecificColumnRetentionDurationForTenant(tf.Ctx, dlc, setting.ID, setting)
	assert.NoErr(t, err)
	setting.Version += 1
	assert.Equal(t, putResp.RetentionDuration, setting)
	readRespAfterPut, err := tf.IDPClient.GetSpecificColumnRetentionDurationForTenant(tf.Ctx, dlc, setting.ID)
	assert.NoErr(t, err)
	assert.Equal(t, readRespAfterPut.RetentionDuration, setting)

	// Delete
	err = tf.IDPClient.DeleteColumnRetentionDurationForTenant(tf.Ctx, dlc, createResp.RetentionDuration.ID)
	assert.NoErr(t, err)
	listResp2, err := tf.IDPClient.GetColumnRetentionDurationForTenant(tf.Ctx, dlc)
	assert.NoErr(t, err)
	assert.True(t, listResp2.RetentionDuration.ID.IsNil())
	assert.True(t, listResp2.RetentionDuration.UseDefault)

	// GET should now give 404
	readResp2, err := tf.IDPClient.GetSpecificColumnRetentionDurationForTenant(tf.Ctx, dlc, createResp.RetentionDuration.ID)
	assertHttpStatus(t, readResp2, err, http.StatusNotFound)
}

func TestRetentionDurations(t *testing.T) {

	tf := idptesthelpers.NewTestFixture(t)

	t.Run("TestBasicColumnRetentionDurationsForColumns", func(t *testing.T) {
		t.Parallel()
		doBasicTestForColumns(t, tf, userstore.DataLifeCycleStateLive)
		doBasicTestForColumns(t, tf, userstore.DataLifeCycleStateSoftDeleted)
	})

	t.Run("TestBasicColumnRetentionDurationsForPurposes", func(t *testing.T) {
		t.Parallel()
		doBasicTestForPurposes(t, tf, userstore.DataLifeCycleStateLive)
		doBasicTestForPurposes(t, tf, userstore.DataLifeCycleStateSoftDeleted)
	})

	t.Run("TestBasicColumnRetentionDurationsForTenants(", func(t *testing.T) {
		//t.Parallel()
		doBasicTestForTenants(t, tf, userstore.DataLifeCycleStateLive)
		doBasicTestForTenants(t, tf, userstore.DataLifeCycleStateSoftDeleted)
	})

	t.Run("TestColumnRetentionDuration404sForBadColumnID(", func(t *testing.T) {
		t.Parallel()
		// Set up a bogus setting to send
		bogusUUID := uuid.Must(uuid.FromString("f3b14c7b-2b78-4e10-bf97-1f1e58d977e1"))
		bogusSetting := idp.ColumnRetentionDuration{
			ColumnID:     bogusUUID,
			PurposeID:    constants.OperationalPurposeID,
			DurationType: userstore.DataLifeCycleStateSoftDeleted,
			Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
		}

		// Try GET collection
		var resp any
		resp, err := tf.IDPClient.GetColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try POST collection
		resp, err = tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, idp.UpdateColumnRetentionDurationsRequest{
			RetentionDurations: []idp.ColumnRetentionDuration{bogusSetting},
		})
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try GET resource
		resp, err = tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try PUT resource
		bogusSetting.ID = bogusUUID
		resp, err = tf.IDPClient.UpdateSpecificColumnRetentionDurationForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID, bogusSetting)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try DELETE resource
		err = tf.IDPClient.DeleteColumnRetentionDurationForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID)
		assertHttpStatus(t, nil, err, http.StatusNotFound)
	})

	t.Run("TestColumnRetentionDuration404sForBadPurposeID(", func(t *testing.T) {
		t.Parallel()
		// Set up a bogus setting to send
		bogusUUID := uuid.Must(uuid.FromString("f3b14c7b-2b78-4e10-bf97-1f1e58d977e1"))
		bogusSetting := idp.ColumnRetentionDuration{
			PurposeID:    bogusUUID,
			DurationType: userstore.DataLifeCycleStateSoftDeleted,
			Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
		}

		// Try GET collection
		var resp any
		resp, err := tf.IDPClient.GetColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try GET resource
		resp, err = tf.IDPClient.GetSpecificColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try PUT resource
		bogusSetting.ID = bogusUUID
		resp, err = tf.IDPClient.UpdateSpecificColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID, bogusSetting)
		assertHttpStatus(t, resp, err, http.StatusNotFound)

		// Try DELETE resource
		err = tf.IDPClient.DeleteColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, bogusUUID, bogusUUID)
		assertHttpStatus(t, nil, err, http.StatusNotFound)
	})
	t.Run("TestCreatingDuplicativeSettings(", func(t *testing.T) {
		t.Parallel()
		col := tf.CreateValidColumn(uniqueName("test_column"), datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		purpose, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name: uniqueName("test_purpose"),
		})
		assert.NoErr(t, err)

		colReq := idp.UpdateColumnRetentionDurationsRequest{
			RetentionDurations: []idp.ColumnRetentionDuration{{
				ColumnID:     col.ID,
				PurposeID:    constants.OperationalPurposeID,
				DurationType: userstore.DataLifeCycleStateSoftDeleted,
				Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
			}},
		}
		// Create good column setting
		_, err = tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, col.ID, colReq)
		assert.NoErr(t, err)
		// Create duplicative column setting
		dupColResp, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, col.ID, colReq)
		assertHttpStatus(t, dupColResp, err, http.StatusConflict)
		assert.Contains(t, err.Error(), "A retention duration setting already exists")

		purpReq := idp.ColumnRetentionDuration{
			PurposeID:    purpose.ID,
			DurationType: userstore.DataLifeCycleStateSoftDeleted,
			Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
		}
		// Create good purpose setting
		_, err = tf.IDPClient.CreateColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, purpose.ID, purpReq)
		assert.NoErr(t, err)
		// Create duplicative purpose setting
		dupPurpResp, err := tf.IDPClient.CreateColumnRetentionDurationForPurpose(tf.Ctx, userstore.DataLifeCycleStateSoftDeleted, purpose.ID, purpReq)
		assertHttpStatus(t, dupPurpResp, err, http.StatusConflict)
		assert.Contains(t, err.Error(), "A retention duration setting already exists")
	})
	t.Run("TestCreatingRetentionDurationSettingForColumnWithFixedID(", func(t *testing.T) {
		t.Parallel()
		col := tf.CreateValidColumn(uniqueName("test_column"), datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		doFixedIdTestForColumns(t, tf, col, uuid.Must(uuid.FromString("2ae6ec32-90f8-4855-8fbd-87510f04ad2f")), userstore.DataLifeCycleStateLive)
		doFixedIdTestForColumns(t, tf, col, uuid.Must(uuid.FromString("2f4fcec5-2ea7-49d0-aa38-982b4ccac977")), userstore.DataLifeCycleStateSoftDeleted)
	})

	t.Run("TestCreatingRetentionDurationSettingForPurposeWithFixedID(", func(t *testing.T) {
		t.Parallel()
		purpose, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name: uniqueName("test_purpose"),
		})
		assert.NoErr(t, err)

		doFixedIdTestForPurposes(t, tf, purpose, uuid.Must(uuid.FromString("bcab9d0a-aba4-4663-8efc-c8ae807cd81b")), userstore.DataLifeCycleStateLive)
		doFixedIdTestForPurposes(t, tf, purpose, uuid.Must(uuid.FromString("a8ac6918-572d-4370-8411-1b1b3b815bfc")), userstore.DataLifeCycleStateSoftDeleted)
	})

	/*t.Run("TestCreatingRetentionDurationSettingForTenantWithFixedID(", func(t *testing.T) {
		//t.Parallel()
		doFixedIdTestForTenants(t, tf, uuid.Must(uuid.FromString("fc5be647-80bf-47c2-bb78-8f72fd9e1170")), userstore.DataLifeCycleStateLive)
		doFixedIdTestForTenants(t, tf, uuid.Must(uuid.FromString("a29ea0b9-97f0-42a1-b6e2-1ffb8cbe4491")), userstore.DataLifeCycleStateSoftDeleted)
	})*/
}

func doFixedIdTestForColumns(t *testing.T, tf *idptesthelpers.TestFixture, col *userstore.Column, targetID uuid.UUID, dlc userstore.DataLifeCycleState) {
	createResp, err := tf.IDPClient.UpdateColumnRetentionDurationsForColumn(tf.Ctx, dlc, col.ID, idp.UpdateColumnRetentionDurationsRequest{
		RetentionDurations: []idp.ColumnRetentionDuration{{
			ID:           targetID,
			ColumnID:     col.ID,
			PurposeID:    constants.OperationalPurposeID,
			DurationType: dlc,
			Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
		}},
	})
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDurations[0].ID, targetID)

	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForColumn(tf.Ctx, dlc, col.ID, targetID)
	assert.NoErr(t, err)
	assert.Equal(t, readResp.RetentionDuration.ID, targetID)
}

func doFixedIdTestForPurposes(t *testing.T, tf *idptesthelpers.TestFixture, purpose *userstore.Purpose, targetID uuid.UUID, dlc userstore.DataLifeCycleState) {
	createResp, err := tf.IDPClient.CreateColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, idp.ColumnRetentionDuration{
		ID:           targetID,
		PurposeID:    purpose.ID,
		DurationType: dlc,
		Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
	})
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration.ID, targetID)

	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForPurpose(tf.Ctx, dlc, purpose.ID, targetID)
	assert.NoErr(t, err)
	assert.Equal(t, readResp.RetentionDuration.ID, targetID)
}

func doFixedIdTestForTenants(t *testing.T, tf *idptesthelpers.TestFixture, targetID uuid.UUID, dlc userstore.DataLifeCycleState) {
	createResp, err := tf.IDPClient.CreateColumnRetentionDurationForTenant(tf.Ctx, dlc, idp.ColumnRetentionDuration{
		ID:           targetID,
		DurationType: dlc,
		Duration:     idp.RetentionDuration{Unit: idp.DurationUnitMonth, Duration: 1},
	})
	assert.NoErr(t, err)
	assert.Equal(t, createResp.RetentionDuration.ID, targetID)

	readResp, err := tf.IDPClient.GetSpecificColumnRetentionDurationForTenant(tf.Ctx, dlc, targetID)
	assert.NoErr(t, err)
	assert.Equal(t, readResp.RetentionDuration.ID, targetID)
}

func TestCreatingRetentionDurationSettingForTenantWithFixedID(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)
	doFixedIdTestForTenants(t, tf, uuid.Must(uuid.FromString("2ae6ec32-90f8-4855-8fbd-87510f04ad2f")), userstore.DataLifeCycleStateLive)
	doFixedIdTestForTenants(t, tf, uuid.Must(uuid.FromString("2f4fcec5-2ea7-49d0-aa38-982b4ccac977")), userstore.DataLifeCycleStateSoftDeleted)
}
