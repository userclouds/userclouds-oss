package storage

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/universe"
)

func TestGenerateDataImportPath(t *testing.T) {
	t.Parallel()
	tenantID := uuid.Must(uuid.FromString("09e8bf49-698b-4bb0-af7e-8f72ef5ba460"))
	jobID := uuid.Must(uuid.FromString("f457a560-66e9-4af0-a94b-359d272f2f71"))
	path := GenerateDataImportPath(tenantID, ExecuteMutatorsImportType, jobID)
	assert.NotNil(t, regexp.MustCompile(importPathRegex).FindStringSubmatch(path))
	assert.Equal(t, fmt.Sprintf("%v/tenants/09e8bf49-698b-4bb0-af7e-8f72ef5ba460/executemutator/v1/f457a560-66e9-4af0-a94b-359d272f2f71", universe.Current()), path)
}

func TestParseDataImportPath(t *testing.T) {
	t.Parallel()
	di, err := ParseDataImportPath(fmt.Sprintf("%v/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v1/faf35f1e-6797-449a-ac78-6feb243203dd", universe.Current()))
	assert.NoErr(t, err)
	assert.Equal(t, uuid.FromStringOrNil("726c6277-e77b-43ad-8d55-13c799dbb9ac"), di.TenantID)
	assert.Equal(t, ExecuteMutatorsImportType, di.ImportType)
	assert.Equal(t, 1, di.Version)
	assert.Equal(t, uuid.FromStringOrNil("faf35f1e-6797-449a-ac78-6feb243203dd"), di.JobID)
	assert.Equal(t, universe.Current(), di.Universe)
	assert.True(t, di.Universe.IsTestOrCI())
	type testcase struct {
		path         string
		errorMessage string
	}
	cases := []testcase{
		{
			path:         "test/tenants",
			errorMessage: "does not match expected format",
		},
		{
			path:         "prod/tenants/726c6277-e77b-43ad-8d5513c799dbb9ac/executemutator/v1/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-1zc799dbb9ac/executemutator/v1/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v1/faf35f1e-6797449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator22/v1/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "test/tens/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator22/v1/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "does not match expected format",
		},
		{
			path:         "prod/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v1/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: fmt.Sprintf("Universe prod doesn't match current universe: %v", universe.Current()),
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v112/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "Version must be 1. got 112",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v2/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "Version must be 1. got 2",
		},
		{
			path:         "test/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/executemutator/v0/faf35f1e-6797-449a-ac78-6feb243203dd",
			errorMessage: "Version must be 1. got 0",
		},
		{
			path:         fmt.Sprintf("%v/tenants/726c6277-e77b-43ad-8d55-13c799dbb9ac/jerry/v1/faf35f1e-6797-449a-ac78-6feb243203dd", universe.Current()),
			errorMessage: "ImportType must be 'executemutator'. got jerry",
		},
	}
	for i, c := range cases {
		di, err = ParseDataImportPath(c.path)
		assert.IsNil(t, di, assert.Errorf("case %d", i))
		assert.NotNil(t, err, assert.Errorf("case %d", i))
		assert.Contains(t, err.Error(), c.errorMessage, assert.Errorf("case %d", i))
	}
}
