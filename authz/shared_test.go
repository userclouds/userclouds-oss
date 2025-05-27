package authz_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

type testFixture struct {
	t          *testing.T
	jwt        string
	client     *authz.Client
	rbacClient *authz.RBACClient
	aLClient   *auditlog.Client
	tenant     *companyconfig.Tenant
	tenantDB   *ucdb.DB
}

func newTestFixture(t *testing.T, opts ...testhelpers.TestProvisionOption) testFixture {
	ctx := context.Background()
	_, tenant, _, tenantDB, _, _ := testhelpers.CreateTestServer(ctx, t, opts...)
	host, err := tenantmap.GetHostFromTenantURL(tenant.TenantURL)
	assert.NoErr(t, err)
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{OrganizationID: tenant.CompanyID.String()}, tenant.TenantURL)
	client, err := authz.NewClient(
		tenant.TenantURL,
		authz.JSONClient(jsonclient.HeaderHost(host),
			jsonclient.HeaderAuthBearer(jwt)))
	assert.NoErr(t, err)
	rbacClient := authz.NewRBACClient(client)
	auditLogClient, err := auditlog.NewClient(tenant.TenantURL, jsonclient.HeaderHost(host), jsonclient.HeaderAuthBearer(jwt))
	assert.NoErr(t, err)
	return testFixture{
		t:          t,
		jwt:        jwt,
		client:     client,
		rbacClient: rbacClient,
		aLClient:   auditLogClient,
		tenant:     tenant,
		tenantDB:   tenantDB,
	}
}

func (tf *testFixture) newTestUser() authz.User {
	tf.t.Helper()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	idptesthelpers.CreateUser(tf.t, tf.tenantDB, userID, uuid.Nil, uuid.Nil, tf.tenant.TenantURL)
	user, err := tf.rbacClient.GetUser(ctx, userID)
	assert.NoErr(tf.t, err)
	return *user
}
