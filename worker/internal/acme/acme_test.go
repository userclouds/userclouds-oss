package acme

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/dnsclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func TestCertIssuance(t *testing.T) {
	ctx := context.Background()
	cdbc, ldbc, s := testhelpers.NewTestStorage(t)
	_, ten, tendb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, s, cdbc, ldbc)
	testDNSClient := dnsclient.NewTestClient(&dnsclient.Config{HostAndPort: uuid.Must(uuid.NewV4()).String()})

	tu := &companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  ten.ID,
		TenantURL: "http://new.foo.com",
	}
	assert.NoErr(t, s.SaveTenantURL(ctx, tu), assert.Errorf("failed to save tenant URL"))

	testDNSClient.SetAnswer("new.foo.com", "CNAME", uctest.MustParseURL(ten.TenantURL).Host)
	err := SetupNewTenantURL(ctx, testDNSClient, &acme.Config{}, tendb, ten.ID, tu.TenantURL, s)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to create new ACME order")
	// acme.newClient() fails under this test since cfg.ACME.PrivateKey is not set
	assert.Contains(t, err.Error(), "failed to decode PEM private key")

	// validate that the DNS "resolved" so the tenantURL is marked valid
	tu, err = s.GetTenantURL(ctx, tu.ID)
	assert.NoErr(t, err, assert.Errorf("failed to get tenant URL"))
	assert.True(t, tu.Validated)
	assert.True(t, tu.Active)
}
