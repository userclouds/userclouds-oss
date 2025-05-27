package storage_test

import (
	"encoding/json"
	"testing"
	"time"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
)

// this just exists to ensure that genconstant is working correctly,
// but it's easier to write here than in the cmd/genconstant directory
func TestGenConstant(t *testing.T) {
	t.Parallel()

	var a oidc.ProviderType
	bs, err := json.Marshal(a)
	assert.NoErr(t, err)
	assert.Equal(t, string(bs), `"none"`)

	b := oidc.ProviderTypeGoogle
	assert.IsNil(t, json.Unmarshal(bs, &b))
	assert.Equal(t, b, oidc.ProviderTypeNone)

	b = oidc.ProviderTypeGoogle
	bs, err = json.Marshal(b)
	assert.NoErr(t, err)
	assert.Equal(t, string(bs), `"google"`)

	var c oidc.ProviderType
	assert.IsNil(t, json.Unmarshal(bs, &c))
	assert.Equal(t, c, oidc.ProviderTypeGoogle)
}

func TestInvalidGenConstant(t *testing.T) {
	t.Parallel()

	a := oidc.ProviderType(123)
	assert.Contains(t, a.String(), "unknown ProviderType value '123'")
}

// likewise this is actually testing tenantdb/migrations.go but easier to write here
func TestTombstoning(t *testing.T) {
	t.Parallel()

	fixture := newStorageForTests(t)
	s := storage.NewUserStorage(fixture.ctx, fixture.db, "", fixture.tenant.ID)
	tdb := fixture.db
	ctx := fixture.ctx

	u := &storage.User{
		BaseUser: storage.BaseUser{VersionBaseModel: ucdb.NewVersionBase()},
	}
	assert.IsNil(t, s.SaveUser(ctx, u), assert.Must())
	assert.True(t, u.Alive())

	got, err := s.GetBaseUser(ctx, u.ID, false)
	assert.NoErr(t, err)
	assert.True(t, got.Alive())

	assert.IsNil(t, s.DeleteUser(ctx, u.ID), assert.Must())

	_, err = s.GetBaseUser(ctx, u.ID, false)
	assert.NotNil(t, err, assert.Must())

	var oneUser []storage.User
	assert.IsNil(t, tdb.SelectContext(ctx, "TestTombstoning", &oneUser, "SELECT id, updated, deleted, organization_id FROM users WHERE id=$1; /* lint-deleted */", u.ID), assert.Must())
	assert.Equal(t, len(oneUser), 1)

	// resave the same user (ID) without it being deleted
	u.Deleted = time.Time{}
	assert.IsNil(t, s.SaveUser(ctx, u), assert.Must())

	// now there should be two if we ignore deleted value
	// Note that this test is slightly sensitive to schema but seems better than a
	// storage method ignoring deleted? I guess we could make it private?
	// NB: in case you are tempted to reuse oneUser from above, it turns out that
	// SelectContext() seems to append into that array, so zero it out first or create a new one!
	var twoUsers []storage.User
	assert.IsNil(t, tdb.SelectContext(ctx, "TestTombstoning", &twoUsers, "SELECT id, updated, deleted, organization_id FROM users WHERE id=$1; /* lint-deleted */", u.ID), assert.Must())
	assert.Equal(t, len(twoUsers), 2)
}
