package storage_test

import (
	"fmt"
	"reflect"
	"testing"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
)

func TestGetLatestAccessPolicyTemplateWithCache(t *testing.T) {
	t.Parallel()

	tf := newTestFixtureForClientCache(t, "ACCESSPOLICYTEMPLATE")
	apt := &storage.AccessPolicyTemplate{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     "Newman",
		Function:                 "function jerry() {}",
	}
	expectedNumFields := tf.getNumFields(reflect.ValueOf(*apt))

	expectedDBCalls := tf.getTotalDBCalls()
	assert.IsNil(t, tf.storage.SaveAccessPolicyTemplate(tf.ctx, apt), assert.Must())
	expectedKey := fmt.Sprintf("%s_%v", tf.keyPrefix, apt.ID)
	expectedDBCalls++
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tf.cacheGetJSON(expectedKey)), expectedNumFields)

	// Load from cache
	loadedApt, err := tf.storage.GetLatestAccessPolicyTemplate(tf.ctx, apt.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedApt, apt)
	tf.assertDBCallCount(expectedDBCalls)

	// Load from DB and populate cache
	tf.deleteKey(expectedKey)
	expectedDBCalls++
	loadedApt, err = tf.storage.GetLatestAccessPolicyTemplate(tf.ctx, apt.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedApt, apt)
	tf.assertDBCallCount(expectedDBCalls)

	apt.Name = "Jerry"
	expectedDBCalls++
	assert.IsNil(t, tf.storage.SaveAccessPolicyTemplate(tf.ctx, apt), assert.Must())
	tf.assertDBCallCount(expectedDBCalls)
	// Loaded from cache
	loadedApt, err = tf.storage.GetLatestAccessPolicyTemplate(tf.ctx, apt.ID)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, loadedApt, apt)
	assert.Equal(t, loadedApt.Name, "Jerry")
}

func TestGetLatestAccessPolicyWithCache(t *testing.T) {
	t.Parallel()

	tf := newTestFixtureForClientCache(t, "ACCESSPOLICY")
	ap := &storage.AccessPolicy{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     "Festivus",
		Description:              "Festivus for the rest of us",
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	}
	expectedNumFields := tf.getNumFields(reflect.ValueOf(*ap))

	expectedDBCalls := tf.getTotalDBCalls()
	assert.IsNil(t, tf.storage.SaveAccessPolicy(tf.ctx, ap), assert.Must())
	expectedDBCalls++
	tf.assertDBCallCount(expectedDBCalls)

	expectedKey := fmt.Sprintf("%s_%v", tf.keyPrefix, ap.ID)
	assert.Equal(t, len(tf.cacheGetJSON(expectedKey)), expectedNumFields)
	loadedAP, err := tf.storage.GetLatestAccessPolicy(tf.ctx, ap.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedAP, ap)
	tf.assertDBCallCount(expectedDBCalls) // no DB calls, since it was loaded from cache
	tf.deleteKey(expectedKey)

	// Load from DB and populate cache
	loadedAP, err = tf.storage.GetLatestAccessPolicy(tf.ctx, ap.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedAP, ap)
	expectedDBCalls++
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tf.cacheGetJSON(expectedKey)), expectedNumFields)

	// Load from cache, no additional DB calls
	loadedAP, err = tf.storage.GetLatestAccessPolicy(tf.ctx, ap.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedAP, ap)
	tf.assertDBCallCount(expectedDBCalls)

	ap.Name = "Jerry"
	expectedDBCalls++
	assert.IsNil(t, tf.storage.SaveAccessPolicy(tf.ctx, ap), assert.Must())
	tf.assertDBCallCount(expectedDBCalls)
	loadedAP, err = tf.storage.GetLatestAccessPolicy(tf.ctx, ap.ID)
	tf.assertDBCallCount(expectedDBCalls) // Loaded from cache
	assert.NoErr(t, err)
	assert.Equal(t, loadedAP, ap)
	assert.Equal(t, loadedAP.Name, "Jerry")
}
