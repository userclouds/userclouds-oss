package secret_test

import (
	"context"
	"encoding/json"
	"testing"

	"sigs.k8s.io/yaml"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/testdb"
)

type testStruct struct {
	Secret secret.String `yaml:"secret" json:"secret" db:"secret"`
}

func TestFieldYAML(t *testing.T) {
	ctx := context.Background()
	y := "secret: dev-literal://not-actually-secret"
	var got testStruct
	assert.NoErr(t, yaml.Unmarshal([]byte(y), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "not-actually-secret")
	assert.Equal(t, got.Secret.String(), "*********************************") // NB: .String() is required for types to match in assert

	y = "secret: dev://Zm9v"
	got = testStruct{}
	assert.NoErr(t, yaml.Unmarshal([]byte(y), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "foo")
}

func TestFieldJSON(t *testing.T) {
	ctx := context.Background()
	j := `{"secret":"dev-literal://testme"}`
	var got testStruct
	assert.NoErr(t, json.Unmarshal([]byte(j), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "testme")
	assert.Equal(t, got.Secret.String(), "********************") // NB: .String() is required for types to match in assert

	j = `{"secret":"dev://Zm9v"}`
	got = testStruct{}
	assert.NoErr(t, json.Unmarshal([]byte(j), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "foo")
	assert.Equal(t, got.Secret.String(), "**********") // NB: .String() is required for types to match in assert

	// make sure we only round trip the location
	bs, err := json.Marshal(got)
	assert.NoErr(t, err)
	assert.Equal(t, string(bs), j)
}

func TestFromLocation(t *testing.T) {
	ctx := context.Background()
	devSecret := secret.FromLocation("dev-literal://festivus")
	sv, err := devSecret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, sv, "festivus")
	assert.Equal(t, devSecret.String(), "**********************") // NB: .String() is required for types to match in assert
}

func TestFieldDB(t *testing.T) {
	ctx := context.Background()

	tdb := testdb.New(t)

	// create a simple test table
	_, err := tdb.ExecContext(ctx, "TestFieldDB", `CREATE TABLE test (secret VARCHAR);`)
	assert.NoErr(t, err)

	savedSecret := "testsecret"
	ts := testStruct{
		Secret: secret.NewTestString(savedSecret),
	}

	_, err = tdb.ExecContext(ctx, "TestFieldDB", `INSERT INTO test (secret) VALUES ($1); /* lint-system-table lint-deleted-ok for test */`, ts.Secret)
	assert.NoErr(t, err)

	var got testStruct
	assert.NoErr(t, tdb.GetContext(ctx, "TestFieldDB", &got, `SELECT * FROM test; /* lint-system-table lint-deleted-ok for test */`))
	gotS, err := got.Secret.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, gotS, savedSecret)
}

func TestResolveInvalidFieldPrefix(t *testing.T) {
	t.Skip("TODO (sgarrity 7/24): remove this when we resolve #4938")
	ctx := context.Background()
	y := "secret: aws://not-actually-secret"
	var got testStruct
	assert.NoErr(t, yaml.Unmarshal([]byte(y), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unknown or missing secret.String prefix: aws://not-actually-secret")
	assert.Equal(t, s, "")
}

func TestValidateInvalidFieldPrefix(t *testing.T) {
	t.Skip("TODO (sgarrity 7/24): remove this when we resolve #4938")
	y := "secret: aws://not-actually-secret"
	var got testStruct
	assert.NoErr(t, yaml.Unmarshal([]byte(y), &got))
	err := got.Secret.Validate()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "secret.String.Validate unrecognized prefix for aws://not-actually-secret")
}

func TestUpdateFromClient(t *testing.T) {
	ctx := context.Background()
	type foo struct {
		Secret secret.String `json:"secret"`
	}

	type req struct {
		Foo foo `json:"foo"`
	}

	j := `{"foo":{"secret":"new-secret"}}`
	var r req
	assert.NoErr(t, json.Unmarshal([]byte(j), &r))
	assert.NoErr(t, r.Foo.Secret.UpdateFromClient(ctx, "service", "name"))
	bs, err := json.Marshal(r)
	assert.NoErr(t, err)
	assert.Equal(t, string(bs), `{"foo":{"secret":"dev://bmV3LXNlY3JldA=="}}`)
}
