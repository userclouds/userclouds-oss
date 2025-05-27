package idp_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/dataimport"
	"userclouds.com/internal/multitenant"
)

func TestImportDataFromFile(t *testing.T) {

	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)
	ctx := multitenant.SetTenantState(tf.Ctx, tf.TenantState)

	mutator, err := tf.CreateMutator("test_mutator",
		policy.AccessPolicyAllowAll.ID,
		[]string{"email"},
		[]uuid.UUID{policy.TransformerPassthrough.ID})
	assert.NoErr(t, err)

	uid, err := tf.IDPClient.CreateUser(ctx, userstore.Record{}, idp.OrganizationID(tf.Company.ID))
	assert.NoErr(t, err)

	tmpDir, err := os.MkdirTemp(os.TempDir(), "dataimport-test-")
	if err != nil {
		uclog.Fatalf(ctx, "failed to mkdirtmp: %v", err)
	}
	defer func() {
		err := os.RemoveAll(tmpDir)
		assert.NoErr(t, err)
	}()

	importFile := fmt.Sprintf("%s/file", tmpDir)
	f, err := os.Create(importFile)
	assert.NoErr(t, err)

	// write the mutator ID followed by empty client context to the file
	_, err = f.Write(fmt.Appendf(nil, "%s\n{}\x02", mutator.ID))
	assert.NoErr(t, err)

	// write the serialized params for ExecuteMutator to the file
	_, err = f.Write(fmt.Appendf(nil, "[\"%s\"]\x01", uid))
	assert.NoErr(t, err)
	rd := map[string]idp.ValueAndPurposes{
		"email": {Value: "me@userclouds.tools", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
	}
	rdJSON, err := json.Marshal(rd)
	assert.NoErr(t, err)
	_, err = f.Write(rdJSON)
	assert.NoErr(t, err)
	f.Close()

	err = dataimport.ImportDataFromFile(ctx, importFile, tf.TenantState)
	assert.NoErr(t, err)

	// check that the mutator was executed
	user, err := tf.IDPClient.GetUser(ctx, uid)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile["email"], "me@userclouds.tools")
}
