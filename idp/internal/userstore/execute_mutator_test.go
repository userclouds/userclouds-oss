package userstore_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/constants"
	internalUserstore "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

// this test ensures that we know what column name was expected (see 2023-10-27 postmortem)
func TestColumnNameMismatch(t *testing.T) {
	ctx := context.Background()

	cc, lc, ccs := testhelpers.NewTestStorage(t)
	company, tenant, tenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cc, lc)
	ts := tenantmap.NewTenantState(tenant, company, uctest.MustParseURL(tenant.TenantURL), tenantDB, nil, nil, "", ccs, false, nil, nil)
	ctx = multitenant.SetTenantState(ctx, ts)

	rd := map[string]idp.ValueAndPurposes{
		"email_verified": {Value: true},
		"email":          {Value: "me@userclouds.tools"},
		"picture":        {Value: "https://twitpic.com/1234"},
		"nickname":       {Value: "Little Bobby Tables"},
		"name":           {Value: "DROP TABLE users;"},
	}

	req := idp.ExecuteMutatorRequest{
		MutatorID:      constants.UpdateUserMutatorID,
		SelectorValues: userstore.UserSelectorValues{uuid.Must(uuid.NewV4())},
		RowData:        rd,
	}
	_, code, err := internalUserstore.ExecuteMutator(ctx, req, tenant.ID, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest)
	assert.Equal(t, ucerr.UserFriendlyMessage(err), "request RowData map missing column 2ee3d57d-9756-464e-a5e9-04244936cb9e (external_alias): got keys [email email_verified name nickname picture]")

	rd["external_alias"] = idp.ValueAndPurposes{Value: "foo"}
	rd["foo"] = idp.ValueAndPurposes{Value: "bar"}
	req.RowData = rd
	_, code, err = internalUserstore.ExecuteMutator(ctx, req, tenant.ID, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, code, http.StatusBadRequest)
	assert.Equal(t, ucerr.UserFriendlyMessage(err), "request RowData has the wrong number of columns (7) for mutator (6)")
}
