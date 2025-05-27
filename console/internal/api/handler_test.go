package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/internal/api"
	consoletesthelpers "userclouds.com/console/testhelpers"
	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/manager"
)

func getWithoutRedirect(url string) (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	resp, err := client.Get(url)
	return resp, ucerr.Wrap(err)
}

func provisionUser(t *testing.T, consoleTenantDB *ucdb.DB, orgID uuid.UUID, tenantURL string) (uuid.UUID, string) {
	userID := uuid.Must(uuid.NewV4())
	_, email := idptesthelpers.CreateUser(t, consoleTenantDB, userID, orgID, uuid.Nil, tenantURL)
	return userID, email
}

func TestHandler(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf := consoletesthelpers.NewTestFixture(t)

	// Provision UC Company admin
	ucCompanyOwnerUserID, cookie, adminEmail := tf.MakeUCAdmin(ctx)

	apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))
	column1, column2, apID, tID := setupForAccessorsAndMutators(t, tf, ctx, apiClient)

	// we need this to sequence eg. company provisioning in these parallel tests because
	// otherwise we race on PlexConfig versions
	var provLock sync.Mutex

	t.Run("TestListAndGetAccessors", func(t *testing.T) {

		accessorOneID := uuid.Must(uuid.NewV4())
		acReq := api.SaveAccessorRequest{
			ID:                 accessorOneID,
			Name:               "Accessor_1",
			Description:        "About accessor 1",
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []api.ConsoleAccessorColumn{{
				Name:          column1.Name,
				TransformerID: tID,
			}},
			ComposedAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
			ComposedTokenAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			Purposes:       []userstore.ResourceID{{Name: "operational"}},
		}
		var acResp userstore.Accessor
		err := apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/accessors", tf.ConsoleTenantID), &acReq, &acResp)
		assert.NoErr(t, err)

		acReq = api.SaveAccessorRequest{
			ID:                 uuid.Must(uuid.NewV4()),
			Name:               "Accessor_2",
			Description:        "",
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []api.ConsoleAccessorColumn{{
				Name:          column2.Name,
				TransformerID: tID,
			}},
			ComposedAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
			ComposedTokenAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			Purposes:       []userstore.ResourceID{{Name: "operational"}},
		}
		err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/accessors", tf.ConsoleTenantID), &acReq, &acResp)
		assert.NoErr(t, err)

		var listResp api.ListConsoleAccessorResponse
		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/userstore/accessors", tf.ConsoleTenantID), &listResp)
		assert.NoErr(t, err)

		foundOne := false
		for _, accessor := range listResp.Data {
			if accessor.Name == "Accessor_1" {
				foundOne = true
				assert.Equal(t, accessor.Version, 0)
				assert.Equal(t, accessor.Description, "About accessor 1")
				assert.Equal(t, accessor.DataLifeCycleState, userstore.DataLifeCycleStateLive)
				assert.Equal(t, len(accessor.Columns), 1)
				assert.Equal(t, accessor.Columns[0].Name, column1.Name)
				assert.Equal(t, accessor.Columns[0].TransformerID, tID)
				assert.False(t, accessor.AccessPolicy.ID.IsNil())
			}
		}
		assert.True(t, foundOne)

		foundTwo := false
		for _, accessor := range listResp.Data {
			if accessor.Name == "Accessor_2" {
				foundTwo = true
				assert.Equal(t, accessor.Version, 0)
				assert.Equal(t, accessor.Description, "")
				assert.Equal(t, accessor.DataLifeCycleState, userstore.DataLifeCycleStateLive)
				assert.Equal(t, len(accessor.Columns), 1)
				assert.Equal(t, accessor.Columns[0].Name, column2.Name)
				assert.Equal(t, accessor.Columns[0].TransformerID, tID)
				assert.False(t, accessor.AccessPolicy.ID.IsNil())
			}
		}
		assert.True(t, foundTwo)

		// GetOne call should include columns and policies
		// TODO: test versioning
		var getResp api.AccessorResponse
		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/userstore/accessors/%s", tf.ConsoleTenantID, accessorOneID), &getResp)
		assert.NoErr(t, err)
		assert.Equal(t, getResp.ID, accessorOneID)
		assert.Equal(t, getResp.Name, "Accessor_1")
		assert.Equal(t, getResp.Version, 0)
		assert.Equal(t, getResp.Description, "About accessor 1")
		assert.Equal(t, getResp.DataLifeCycleState, userstore.DataLifeCycleStateLive)
		assert.Equal(t, getResp.Columns[0].Name, column1.Name)
		assert.Equal(t, getResp.SelectorConfig, userstore.UserSelectorConfig{WhereClause: "{id} = ?"})
	})

	t.Run("TestListMutators", func(t *testing.T) {
		t.Parallel()

		acReq := api.SaveMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Mutator_1",
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			Columns: []api.ConsoleMutatorColumn{{
				Name:         column1.Name,
				NormalizerID: policy.TransformerPassthrough.ID,
			}},
			ComposedAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
		}
		var acResp userstore.Mutator
		err := apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/mutators", tf.ConsoleTenantID), &acReq, &acResp)
		assert.NoErr(t, err)

		acReq = api.SaveMutatorRequest{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Mutator_2",
			SelectorConfig: userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
			Columns: []api.ConsoleMutatorColumn{{
				Name:         column2.Name,
				NormalizerID: policy.TransformerPassthrough.ID,
			}},
			ComposedAccessPolicy: &policy.AccessPolicy{
				PolicyType: policy.PolicyTypeCompositeAnd,
				Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: apID}}},
			},
		}
		err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/mutators", tf.ConsoleTenantID), &acReq, &acResp)
		assert.NoErr(t, err)

		var listResp api.ListConsoleMutatorResponse
		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/userstore/mutators", tf.ConsoleTenantID), &listResp)
		assert.NoErr(t, err)

		foundOne := false
		for _, mutator := range listResp.Data {
			if mutator.Name == "Mutator_1" {
				foundOne = true
				assert.Equal(t, mutator.Version, 0)
				assert.Equal(t, len(mutator.Columns), 1)
				assert.Equal(t, mutator.Columns[0].Name, column1.Name)
				assert.Equal(t, mutator.Columns[0].NormalizerID, policy.TransformerPassthrough.ID)
				assert.False(t, mutator.AccessPolicy.ID.IsNil())
			}
		}
		assert.True(t, foundOne)

		foundTwo := false
		for _, mutator := range listResp.Data {
			if mutator.Name == "Mutator_2" {
				foundTwo = true
				assert.Equal(t, mutator.Version, 0)
				assert.Equal(t, len(mutator.Columns), 1)
				assert.Equal(t, mutator.Columns[0].Name, column2.Name)
				assert.Equal(t, mutator.Columns[0].NormalizerID, policy.TransformerPassthrough.ID)
				assert.False(t, mutator.AccessPolicy.ID.IsNil())
			}
		}
		assert.True(t, foundTwo)
	})

	t.Run("TestListUsers", func(t *testing.T) {
		t.Parallel()

		// Create a new "customer" company
		newCompany := companyconfig.NewCompany("NewListUsersCompany", companyconfig.CompanyTypeCustomer)

		provLock.Lock()
		testhelpers.ProvisionTestCompany(ctx, t, tf.CompanyConfigStorage, &newCompany, tf.ConsoleTenantDB, tf.ConsoleTenantID, tf.ConsoleTenantCompanyID, provisioning.Owner(ucCompanyOwnerUserID))
		provLock.Unlock()

		// Remove the UC Company Admin from the new company
		ucAdminUser, err := tf.RBACClient.GetUser(ctx, ucCompanyOwnerUserID)
		assert.NoErr(t, err)
		group, err := tf.RBACClient.GetGroup(ctx, newCompany.ID)
		assert.NoErr(t, err)
		err = group.RemoveUser(ctx, *ucAdminUser)
		assert.NoErr(t, err)
		// Provision a regular Console user (i.e. a normal customer/developer account)
		developerUserID, _ := provisionUser(t, tf.ConsoleTenantDB, newCompany.ID, tf.TenantServerURL)
		user, err := tf.RBACClient.GetUser(ctx, developerUserID)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
		assert.NoErr(t, err)

		// Make sure UC Company Admin can list all users in Console tenant.
		// This also ensures that IDP tokens work for the Console tenant.
		ucAdminAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))
		var resp idp.ListUsersResponse
		err = ucAdminAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users", tf.ConsoleTenantID), &resp)
		assert.NoErr(t, err)
		users := resp.Data
		// can't check number of users but can check that both are in the list
		var haveDeveloper, haveOwner bool
		for _, user := range users {
			if user.ID == developerUserID {
				haveDeveloper = true
			}
			if user.ID == ucCompanyOwnerUserID {
				haveOwner = true
			}
		}
		assert.True(t, haveDeveloper)
		assert.True(t, haveOwner)

		newTenant, tenantDB := testhelpers.ProvisionTestTenant(ctx, t, tf.CompanyConfigStorage, tf.CompanyConfigDBCfg, tf.LogDBCfg, newCompany.ID)
		assert.NoErr(t, err)

		// Start a new server for the new tenant just because we need a unique URL for each tenant
		// otherwise tenant resolution doesn't work (plus it violates a unique constraint)
		newTenantServer := httptest.NewServer(tf.TenantServer.Config.Handler)
		t.Cleanup(newTenantServer.Close)
		newTenantServerURL := testhelpers.UpdateTenantURLForTestTenant(t, newTenant.TenantURL, newTenantServer.URL)
		testhelpers.FixupTenantURL(t, tf.CompanyConfigStorage, newTenant, newTenantServerURL, tenantDB)

		// Create an "end user" in the developer's tenant
		endUserID := uuid.Must(uuid.NewV4())
		_, email := idptesthelpers.CreateUser(t, tenantDB, endUserID, uuid.Nil, newTenant.ID, newTenantServerURL)

		// List users in the developer's new tenant (using the developer's auth).
		developerCookie, err := tf.GetUserCookie(developerUserID, email)
		assert.NoErr(t, err)
		developerAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*developerCookie))
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users", newTenant.ID), &resp)
		assert.NoErr(t, err)
		users = resp.Data
		assert.Equal(t, len(users), 1, assert.Must())
		assert.Equal(t, users[0].ID, endUserID)

		// Make sure developer can't list users in the Console tenant
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users", tf.ConsoleTenantID), &resp)
		assert.NotNil(t, err, assert.Must())
		var clientError jsonclient.Error
		assert.True(t, errors.As(err, &clientError), assert.Must())
		assert.Equal(t, clientError.StatusCode, http.StatusForbidden)

		// Make sure UC admin can't list users in the developer's tenant
		err = ucAdminAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users", newTenant.ID), &resp)
		assert.NotNil(t, err, assert.Must())
		assert.True(t, errors.As(err, &clientError), assert.Must())
		assert.Equal(t, clientError.StatusCode, http.StatusForbidden)

		// TODO (sgarrity 8/24): I can't figure out how to make this reasonably parallelized
		// and it seems low risk so commenting it out for now :/
		// Remove Console as a valid fallback provider for JWT verification and ensure that talking
		// to non-Console tenants will fail.
		// tf.OIDCProviderMap.ClearFallbackProvider()
		// err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users", newTenant.ID), &resp)
		// assert.NotNil(t, err, assert.Must())
	})

	t.Run("TestSqlShimProxies", func(t *testing.T) {
		t.Parallel()

		// Create a new "customer" company
		newCompany := companyconfig.NewCompany("NewCompanySqlShim", companyconfig.CompanyTypeCustomer)
		provLock.Lock()
		testhelpers.ProvisionTestCompany(ctx, t, tf.CompanyConfigStorage, &newCompany, tf.ConsoleTenantDB, tf.ConsoleTenantID, tf.ConsoleTenantCompanyID, provisioning.Owner(ucCompanyOwnerUserID))
		provLock.Unlock()

		// Provision a regular Console user (i.e. a normal customer/developer account)
		developerUserID, developerEmail := provisionUser(t, tf.ConsoleTenantDB, newCompany.ID, tf.TenantServerURL)
		user, err := tf.RBACClient.GetUser(ctx, developerUserID)
		assert.NoErr(t, err)
		group, err := tf.RBACClient.GetGroup(ctx, newCompany.ID)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
		assert.NoErr(t, err)
		developerCookie, err := tf.GetUserCookie(developerUserID, developerEmail)
		assert.NoErr(t, err)
		developerAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*developerCookie))

		// Provision a new tenant for the company
		newTenant, tenantDB := testhelpers.ProvisionTestTenant(ctx, t, tf.CompanyConfigStorage, tf.CompanyConfigDBCfg, tf.LogDBCfg, newCompany.ID, testhelpers.EmployeeIDs([]uuid.UUID{developerUserID, ucCompanyOwnerUserID}), testhelpers.UseOrganizations())
		newTenantServer := httptest.NewServer(tf.TenantServer.Config.Handler)
		t.Cleanup(newTenantServer.Close)
		newTenantServerURL := testhelpers.UpdateTenantURLForTestTenant(t, newTenant.TenantURL, newTenantServer.URL)
		testhelpers.FixupTenantURL(t, tf.CompanyConfigStorage, newTenant, newTenantServerURL, tenantDB)

		// Test setting the sqlshim database and proxy using the developer's auth
		dbIn := api.DatabaseWithProxy{
			SQLShimDatabase: userstore.SQLShimDatabase{
				Name:     "testdb",
				Type:     "postgres",
				Host:     "localhost",
				Port:     5432,
				Username: "testuser",
				Password: "testpassword",
			},
			ProxyHost: "localhost",
			ProxyPort: 5433,
		}
		err = developerAPIClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/databases", newTenant.ID), dbIn, nil)
		assert.NoErr(t, err)

		// Verify that the database was created but the proxy host and port were not set
		var dbList api.ListDatabaseWithProxyResponse
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/userstore/databases", newTenant.ID), &dbList)

		assert.NoErr(t, err)

		foundOne := false
		var dbID uuid.UUID
		for _, database := range dbList.Data {
			if database.Name == "testdb" {
				foundOne = true
				dbID = database.ID
				assert.True(t, dbIn.SQLShimDatabase.EqualsIgnoringNilIDSchemasAndPassword(database.SQLShimDatabase))
				assert.Equal(t, database.ProxyHost, "")
				assert.Equal(t, database.ProxyPort, 0)
			}
		}
		assert.True(t, foundOne)

		// Now do the same with the UC Admin's auth
		ucAdminCookie, err := tf.GetUserCookie(ucCompanyOwnerUserID, adminEmail)
		assert.NoErr(t, err)
		ucAdminAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*ucAdminCookie))

		dbIn.ID = dbID
		err = ucAdminAPIClient.Put(ctx, fmt.Sprintf("/api/tenants/%s/userstore/databases/%s", newTenant.ID, dbID), dbIn, nil)
		assert.NoErr(t, err)

		// Verify that everything, including proxy host and port are set
		var dbOne api.DatabaseWithProxy
		err = ucAdminAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/userstore/databases/%s", newTenant.ID, dbID), &dbOne)
		assert.NoErr(t, err)
		assert.True(t, dbIn.SQLShimDatabase.EqualsIgnoringNilIDSchemasAndPassword(dbOne.SQLShimDatabase))
		assert.Equal(t, dbOne.ProxyHost, "localhost")
		assert.Equal(t, dbOne.ProxyPort, 5433)
	})

	t.Run("TestLoginApps", func(t *testing.T) {
		apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))

		var loginApps []tenantplex.App
		err := apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/loginapps", tf.ConsoleTenantID), &loginApps)
		assert.NoErr(t, err)
		assert.True(t, len(loginApps) > 0)

		var loginApp tenantplex.App
		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/loginapps/%s", tf.ConsoleTenantID, loginApps[0].ID), &loginApp)
		assert.NoErr(t, err)
		var found bool
		for _, app := range loginApps {
			if app.ID == loginApp.ID {
				found = true
				break
			}
		}
		assert.True(t, found)

		loginApp.ClientID = "new-client-id"
		req := api.UpdateLoginAppRequest{
			App: loginApp,
		}
		err = apiClient.Put(ctx, fmt.Sprintf("/api/tenants/%s/loginapps/%s", tf.ConsoleTenantID, loginApp.ID), req, nil)
		assert.NotNil(t, err)

		loginApp.ClientID = loginApps[0].ClientID
		loginApp.Description = "new description"
		req.App = loginApp
		err = apiClient.Put(ctx, fmt.Sprintf("/api/tenants/%s/loginapps/%s", tf.ConsoleTenantID, loginApp.ID), req, nil)
		assert.NoErr(t, err)

		addReq := api.AddLoginAppRequest{
			AppID:        uuid.Must(uuid.NewV4()),
			Name:         "new app",
			ClientID:     "new-client-id",
			ClientSecret: "new-client-secret",
		}
		err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/loginapps", tf.ConsoleTenantID), addReq, nil)
		assert.NoErr(t, err)

		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/loginapps", tf.ConsoleTenantID), &loginApps)
		assert.NoErr(t, err)
		assert.True(t, len(loginApps) >= 2)
		var found1, found2 bool
		for _, app := range loginApps {
			if app.ID == loginApp.ID {
				found1 = true
			}
			if app.ID == addReq.AppID {
				found2 = true
			}
		}
		assert.True(t, found1)
		assert.True(t, found2)

		err = apiClient.Delete(ctx, fmt.Sprintf("/api/tenants/%s/loginapps/%s", tf.ConsoleTenantID, addReq.AppID), nil)
		assert.NoErr(t, err)

		err = apiClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/loginapps", tf.ConsoleTenantID), &loginApps)
		assert.NoErr(t, err)
		assert.True(t, len(loginApps) > 0)
		found = false
		var notFound bool // should stay false unless delete failed
		for _, app := range loginApps {
			if app.ID == addReq.AppID {
				notFound = true
			}
			if app.ID == loginApp.ID {
				found = true
			}
		}
		assert.True(t, found)
		assert.False(t, notFound)
	})

	t.Run("TestOrganizations", func(t *testing.T) {
		t.Parallel()

		// Create a new "customer" company
		newCompany := companyconfig.NewCompany("NewCompanyOrgs", companyconfig.CompanyTypeCustomer)
		provLock.Lock()
		testhelpers.ProvisionTestCompany(ctx, t, tf.CompanyConfigStorage, &newCompany, tf.ConsoleTenantDB, tf.ConsoleTenantID, tf.ConsoleTenantCompanyID, provisioning.Owner(ucCompanyOwnerUserID))
		provLock.Unlock()

		// Remove the UC Company Admin from the new company
		ucAdminUser, err := tf.RBACClient.GetUser(ctx, ucCompanyOwnerUserID)
		assert.NoErr(t, err)
		group, err := tf.RBACClient.GetGroup(ctx, newCompany.ID)
		assert.NoErr(t, err)
		err = group.RemoveUser(ctx, *ucAdminUser)
		assert.NoErr(t, err)

		// Provision a regular Console user (i.e. a normal customer/developer account)
		developerUserID, developerEmail := provisionUser(t, tf.ConsoleTenantDB, newCompany.ID, tf.TenantServerURL)
		user, err := tf.RBACClient.GetUser(ctx, developerUserID)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
		assert.NoErr(t, err)
		developerCookie, err := tf.GetUserCookie(developerUserID, developerEmail)
		assert.NoErr(t, err)
		developerAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*developerCookie))

		// Provision a new tenant for the company
		newTenant, tenantDB := testhelpers.ProvisionTestTenant(ctx, t, tf.CompanyConfigStorage, tf.CompanyConfigDBCfg, tf.LogDBCfg, newCompany.ID, testhelpers.EmployeeIDs([]uuid.UUID{developerUserID}), testhelpers.UseOrganizations())
		newTenantServer := httptest.NewServer(tf.TenantServer.Config.Handler)
		assert.NoErr(t, err)
		newTenantServerURL := testhelpers.UpdateTenantURLForTestTenant(t, newTenant.TenantURL, newTenantServer.URL)
		testhelpers.FixupTenantURL(t, tf.CompanyConfigStorage, newTenant, newTenantServerURL, tenantDB)

		// create a new organization within the new tenant
		var org authz.Organization
		err = developerAPIClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/organizations", newTenant.ID), &api.CreateOrganizationRequest{ID: uuid.Must(uuid.NewV4()), Name: "NewOrg"}, &org)
		assert.NoErr(t, err)

		// find the login app for that organization
		var loginApps []tenantplex.App
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/loginapps?organization_id=%s", newTenant.ID, org.ID), &loginApps)
		assert.NoErr(t, err)
		assert.Equal(t, len(loginApps), 1)

		// create a object in the new organization
		var objReq api.CreateObjectRequest
		objReq.Object.TypeID = authz.GroupObjectTypeID
		objReq.Object.Alias = "NewObject"
		objReq.Object.OrganizationID = org.ID
		var obj authz.Object
		err = developerAPIClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/authz/objects", newTenant.ID), &objReq, &obj)
		assert.NoErr(t, err)
		// Make sure we can find the object by alias/typeID/org
		var objs authz.ListObjectsResponse
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/objects?type_id=%v&name=%v&organization_id=%v", newTenant.ID, obj.TypeID, *obj.Alias, org.ID), &objs)
		assert.NoErr(t, err)
		found := false
		for _, o := range objs.Data {
			if obj.ID == o.ID {
				found = true
			}
		}
		assert.True(t, found)
		// Make sure we can't find the object by alias/typeID/ and wrong org
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/objects?type_id=%v&name=%v&organization_id=%v", newTenant.ID, obj.TypeID, *obj.Alias, uuid.Must(uuid.NewV4())), &objs)
		assert.NoErr(t, err)
		found = false
		for _, o := range objs.Data {
			if obj.ID == o.ID {
				found = true
			}
		}
		assert.False(t, found)
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/objects?organization_id=%s", newTenant.ID, org.ID), &objs)
		assert.NoErr(t, err)
		found = false
		for _, o := range objs.Data {
			if obj.ID == o.ID {
				found = true
			}
		}
		assert.True(t, found)

		// create an edge type in the new organization
		var edgeReq api.CreateEdgeTypeRequest
		edgeReq.EdgeType.SourceObjectTypeID = authz.GroupObjectTypeID
		edgeReq.EdgeType.TargetObjectTypeID = authz.UserObjectTypeID
		edgeReq.EdgeType.TypeName = "NewEdgeType"
		edgeReq.EdgeType.OrganizationID = org.ID
		var edgeType authz.EdgeType
		err = developerAPIClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/authz/edgetypes", newTenant.ID), &edgeReq, &edgeType)
		assert.NoErr(t, err)
		var edgeTypes authz.ListEdgeTypesResponse
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/edgetypes?organization_id=%s", newTenant.ID, org.ID), &edgeTypes)
		assert.NoErr(t, err)
		assert.Equal(t, len(edgeTypes.Data), 1)

		// create a user in the new organization
		newUser, _ := provisionUser(t, tenantDB, org.ID, newTenantServerURL)
		var resp idp.ListUsersResponse
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users?organization_id=%s", newTenant.ID, org.ID), &resp)
		assert.NoErr(t, err)
		assert.Equal(t, len(resp.Data), 1)
		assert.Equal(t, resp.Data[0].ID, newUser)

		// create another organization within the new tenant
		var org2 authz.Organization
		err = developerAPIClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/organizations", newTenant.ID), &api.CreateOrganizationRequest{ID: uuid.Must(uuid.NewV4()), Name: "NewOrg2"}, &org2)
		assert.NoErr(t, err)

		// verify that the object, edge type, and user are not visible in the new organization
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/objects?organization_id=%s", newTenant.ID, org2.ID), &objs)
		assert.NoErr(t, err)
		found = false
		for _, o := range objs.Data {
			if obj.ID == o.ID {
				found = true
			}
		}
		assert.False(t, found)
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/authz/edgetypes?organization_id=%s", newTenant.ID, org2.ID), &edgeTypes)
		assert.NoErr(t, err)
		assert.Equal(t, len(edgeTypes.Data), 0)
		err = developerAPIClient.Get(ctx, fmt.Sprintf("/api/tenants/%s/users?organization_id=%s", newTenant.ID, org2.ID), &resp)
		assert.NoErr(t, err)
		assert.Equal(t, len(resp.Data), 0)
	})

	t.Run("TestInvite", func(t *testing.T) {
		t.Parallel()
		cacheCfg := cachetesthelpers.NewCacheConfig()
		// User creates a new account
		email, password := genEmailAndPassword()
		userID, err := createUser(tf, email, password)
		assert.NoErr(t, err)

		// set no signups for this test
		mgr := manager.NewFromDB(tf.ConsoleTenantDB, cacheCfg)
		pc, err := mgr.GetTenantPlex(ctx, tf.ConsoleTenantID)
		assert.NoErr(t, err)
		pc.PlexConfig.DisableSignUps = true
		assert.IsNil(t, mgr.SaveTenantPlex(ctx, pc), assert.Must())

		// User is auth'd with Console but can't create an company - should get 403 response
		_, err = createCompany(tf, userID, email, "testcompany")
		assert.NotNil(t, err, assert.Must())
		var jsonClientErr jsonclient.Error
		assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
		assert.Equal(t, jsonClientErr.StatusCode, http.StatusForbidden)

		// Let UC Admin create a company now
		newCompanyID, err := createCompany(tf, ucCompanyOwnerUserID, "admin@uc.com", "testcompany")
		assert.NoErr(t, err)

		// There should be 1 user in the ACL associated with the new company,
		// which is the user who created it (admin).
		companyACL, err := tf.RBACClient.GetGroup(ctx, newCompanyID)
		assert.NoErr(t, err)
		companyMemberships, err := companyACL.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(companyMemberships), 1, assert.Must())
		assert.Equal(t, companyMemberships[0].User.ID, ucCompanyOwnerUserID)
		assert.Equal(t, companyMemberships[0].Role, ucauthz.AdminRole)

		loginApps, err := mgr.GetLoginApps(ctx, tf.ConsoleTenantID, newCompanyID)
		assert.NoErr(t, err)
		assert.Equal(t, len(loginApps), 1, assert.Must())
		companyApp := loginApps[0]
		time.Sleep(15 * time.Second) // need to sleep so that plex's cache is invalidated

		// Creator of new company invites the teammate, who follows through with the create user flow
		teammateEmail, teammatePassword := genEmailAndPassword()
		inviteURL, err := inviteUserToCompany(tf, ucCompanyOwnerUserID, email, teammateEmail, newCompanyID)
		assert.NoErr(t, err)
		teammateUserID, postLoginRedirectURL, err := createUserWithInvite(tf, inviteURL, companyApp.ClientID, teammateEmail, teammateEmail, teammatePassword)
		assert.NoErr(t, err)
		res, err := getWithoutRedirect(postLoginRedirectURL)
		assert.NoErr(t, err)
		assert.Equal(t, res.StatusCode, http.StatusSeeOther)

		companyMemberships, err = companyACL.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(companyMemberships), 2, assert.Must())
		if companyMemberships[0].User.ID == teammateUserID {
			assert.Equal(t, companyMemberships[0].Role, ucauthz.MemberRole)
		} else {
			assert.Equal(t, companyMemberships[1].User.ID, teammateUserID)
			assert.Equal(t, companyMemberships[1].Role, ucauthz.MemberRole)
		}

		// New user can NOT create a new company either
		_, err = createCompany(tf, teammateUserID, teammateEmail, "testcompany3")
		assert.NotNil(t, err, assert.Must())
	})
}

func setupForAccessorsAndMutators(t *testing.T, tf *consoletesthelpers.TestFixture, ctx context.Context, apiClient *jsonclient.Client) (column1, column2 userstore.Column, accessPolicyID, transformerID uuid.UUID) {
	var err error

	column1 = userstore.Column{
		ID:        uuid.Must(uuid.NewV4()),
		Table:     "users",
		Name:      "Field1",
		DataType:  datatype.String,
		IndexType: userstore.ColumnIndexTypeIndexed,
	}
	column2 = userstore.Column{
		ID:        uuid.Must(uuid.NewV4()),
		Table:     "users",
		Name:      "Field2",
		DataType:  datatype.Timestamp,
		IndexType: userstore.ColumnIndexTypeIndexed,
	}

	req := api.ConsoleSaveColumnRequest{
		Column: column1,
		ComposedAccessPolicy: &policy.AccessPolicy{
			PolicyType: policy.PolicyTypeCompositeAnd,
			Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}}},
		},
	}
	var resp userstore.Column
	err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/columns", tf.ConsoleTenantID), &req, &resp)
	assert.NoErr(t, err)

	req = api.ConsoleSaveColumnRequest{
		Column: column2,
		ComposedAccessPolicy: &policy.AccessPolicy{
			PolicyType: policy.PolicyTypeCompositeAnd,
			Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}}},
		},
	}
	err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/userstore/columns", tf.ConsoleTenantID), &req, &resp)
	assert.NoErr(t, err)

	apt := policy.AccessPolicyTemplate{
		Name:     "TrueTemplate" + crypto.MustRandomHex(4),
		Function: "function policy(x, y) { return true; } /* " + uuid.Must(uuid.NewV4()).String() + " */",
	}
	aptReq := tokenizer.CreateAccessPolicyTemplateRequest{
		AccessPolicyTemplate: apt,
	}
	var aptResp policy.AccessPolicyTemplate
	err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/policies/templates", tf.ConsoleTenantID), &aptReq, &aptResp)
	assert.NoErr(t, err)
	aptID := aptResp.ID

	ap := policy.AccessPolicy{
		Name:       "TruePolicy3" + crypto.MustRandomHex(4),
		Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: aptID}}},
		PolicyType: policy.PolicyTypeCompositeAnd,
	}
	apReq := tokenizer.CreateAccessPolicyRequest{
		AccessPolicy: ap,
	}
	var apResp *policy.AccessPolicy
	err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/policies/access", tf.ConsoleTenantID), &apReq, &apResp)
	assert.NoErr(t, err)
	accessPolicyID = apResp.ID

	transformer := policy.Transformer{
		Name:           "PolicyXYZ" + crypto.MustRandomHex(4),
		Description:    "Test Transformer",
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTransform,
		Function:       "function transform(x, y) { return 'xyz'; } /* " + uuid.Must(uuid.NewV4()).String() + " */",
	}
	tReq := tokenizer.CreateTransformerRequest{
		Transformer: transformer,
	}
	var tResp *policy.Transformer
	err = apiClient.Post(ctx, fmt.Sprintf("/api/tenants/%s/policies/transformation", tf.ConsoleTenantID), &tReq, &tResp)
	assert.NoErr(t, err)
	transformerID = tResp.ID

	return column1, column2, accessPolicyID, transformerID
}

func inviteUserToCompany(tf *consoletesthelpers.TestFixture, inviterUserID uuid.UUID, inviterEmail string, inviteeEmail string, companyID uuid.UUID) (*url.URL, error) {
	ctx := context.Background()

	userCookie, err := tf.GetUserCookie(inviterUserID, inviterEmail)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	emails := len(tf.Email.Bodies)
	htmlEmails := len(tf.Email.HTMLBodies)

	inviterAPIClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*userCookie))
	if err := inviterAPIClient.Post(ctx, fmt.Sprintf("/api/companies/%s/actions/inviteuser", companyID), &api.InviteUserRequest{
		InviteeEmails: inviteeEmail,
		CompanyRole:   ucauthz.MemberRole,
	}, nil); err != nil {
		return nil, ucerr.Wrap(err)
	}
	const expectedSubstr = "being invited to join"
	if len(tf.Email.Bodies) != emails+1 || len(tf.Email.HTMLBodies) != htmlEmails+1 {
		return nil, ucerr.New("inviting user to UC did not send 1 text + html email")
	}
	if !strings.Contains(tf.Email.Bodies[emails], expectedSubstr) ||
		!strings.Contains(tf.Email.HTMLBodies[htmlEmails], expectedSubstr) {
		return nil, ucerr.Errorf("invite email did not contain expected text (got '%s' and '%s', expected '%s'",
			tf.Email.Bodies[emails], tf.Email.HTMLBodies[htmlEmails], expectedSubstr)
	}
	inviteURL, err := uctest.ExtractURL(tf.Email.HTMLBodies[htmlEmails])
	return inviteURL, ucerr.Wrap(err)
}

func createUserWithInvite(tf *consoletesthelpers.TestFixture, inviteURL *url.URL, clientID, email, username, password string) (uuid.UUID, string, error) {
	ctx := context.Background()
	inviteResp, err := getWithoutRedirect(inviteURL.String())
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}
	if inviteResp.StatusCode != http.StatusTemporaryRedirect {
		return uuid.Nil, "", ucerr.Errorf("expected code %d, got %d", http.StatusTemporaryRedirect, inviteResp.StatusCode)
	}
	inviteLoginRedirect, err := inviteResp.Location()
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}
	sessionID, err := uuid.FromString(inviteLoginRedirect.Query().Get("session_id"))
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	plexClient := plex.NewClient(tf.TenantServerURL)
	createResp, err := plexClient.CreateUser(ctx, plex.CreateUserRequest{
		SessionID: sessionID,
		ClientID:  clientID,
		Email:     email,
		Username:  username,
		Password:  password,
	})
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	loginResp, err := plexClient.UsernamePasswordLogin(ctx, plex.LoginRequest{
		Username:  username,
		Password:  password,
		SessionID: sessionID,
	})
	if err != nil {
		return uuid.Nil, "", ucerr.Wrap(err)
	}

	return uuid.Must(uuid.FromString(createResp.UserID)), loginResp.RedirectTo, nil
}

func createCompany(tf *consoletesthelpers.TestFixture, creatorUserID uuid.UUID, creatorEmail, companyName string) (uuid.UUID, error) {
	ctx := context.Background()
	// Make request on behalf of creator by setting cookie
	userCookie, err := tf.GetUserCookie(creatorUserID, creatorEmail)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*userCookie))
	createCompanyReq := api.CreateCompanyRequest{
		Company: companyconfig.NewCompany(companyName, companyconfig.CompanyTypeCustomer),
	}
	if err := apiClient.Post(ctx, "/api/companies", createCompanyReq, nil); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return createCompanyReq.Company.ID, nil
}

func createUser(tf *consoletesthelpers.TestFixture, emailUsername, password string) (uuid.UUID, error) {
	ctx := context.Background()
	plexClient := plex.NewClient(tf.TenantServerURL)
	user, err := plexClient.CreateUser(ctx, plex.CreateUserRequest{
		ClientID: tf.ClientID,
		Email:    emailUsername,
		Username: emailUsername,
		Password: password,
	})
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	userID, err := uuid.FromString(user.UserID)
	return userID, ucerr.Wrap(err)
}

func genEmailAndPassword() (string, string) {
	email := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
	password := fmt.Sprintf("%s_password", email)
	return email, password
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
