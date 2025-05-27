package api_test

import (
	"context"
	"net/http"
	"testing"

	"userclouds.com/console/internal/api"
	consoletesthelpers "userclouds.com/console/testhelpers"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/plex/manager"
)

func TestCompany(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tf := consoletesthelpers.NewTestFixture(t)

	type c struct {
		cookie         *http.Cookie
		disableSignups bool
		expectError    bool
	}
	_, cookie, _ := tf.MakeUCAdmin(ctx)
	cases := make(map[string]c)
	cases["ucadmin"] = c{
		cookie,
		false,
		false,
	}

	uid, email := provisionUser(t, tf.ConsoleTenantDB, tf.UserCloudsCompanyID, tf.TenantServerURL)
	cookie, err := tf.GetUserCookie(uid, email)
	assert.NoErr(t, err)
	cases["normal"] = c{
		cookie,
		false,
		false,
	}

	cases["normal_nosignups"] = c{
		cookie,
		true,
		true,
	}

	for k, v := range cases {
		t.Run("Test Create"+k, func(t *testing.T) {
			// set signup flag
			cacheCfg := cachetesthelpers.NewCacheConfig()
			mgr, err := manager.NewFromCompanyConfig(ctx, tf.CompanyConfigStorage, tf.ConsoleTenantID, cacheCfg)
			assert.NoErr(t, err)
			pc, err := mgr.GetTenantPlex(ctx, tf.ConsoleTenantID)
			assert.NoErr(t, err)
			pc.PlexConfig.DisableSignUps = v.disableSignups
			assert.NoErr(t, mgr.SaveTenantPlex(ctx, pc))

			// create req
			apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*v.cookie))
			createCompanyReq := api.CreateCompanyRequest{
				Company: companyconfig.NewCompany("Original Name", companyconfig.CompanyTypeCustomer),
			}
			var createCompanyResp companyconfig.Company
			err = apiClient.Post(ctx, "/api/companies", createCompanyReq, &createCompanyResp)

			if v.expectError {
				assert.NotNil(t, err, assert.Must())
			} else {
				assert.NoErr(t, err)

				dbCompany, err := tf.CompanyConfigStorage.GetCompany(ctx, createCompanyReq.Company.ID)
				assert.NoErr(t, err)
				assert.Equal(t, *dbCompany, createCompanyResp)

				err = apiClient.Delete(ctx, "/api/companies/"+dbCompany.ID.String(), nil)
				assert.NoErr(t, err)
			}
		})
	}

	t.Run("TestUpdateCompany", func(t *testing.T) {
		uid, cookie, _ := tf.MakeUCAdmin(ctx)
		assert.NoErr(t, err)
		apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))

		oid, err := createCompany(tf, uid, "testuser@contoso.com", "Original Name")
		assert.NoErr(t, err)

		newName := "New Name"
		var updateCompanyReq api.UpdateCompanyRequest
		updateCompanyReq.Company.Name = &newName

		var updateCompanyResp api.UpdateCompanyResponse
		err = apiClient.Put(ctx, "/api/companies/"+oid.String(), updateCompanyReq, &updateCompanyResp)
		assert.NoErr(t, err)

		dbCompany, err := tf.CompanyConfigStorage.GetCompany(ctx, oid)
		assert.NoErr(t, err)
		assert.Equal(t, dbCompany.Name, newName)
		assert.Equal(t, dbCompany.Type, companyconfig.CompanyTypeCustomer)
		assert.Equal(t, *dbCompany, updateCompanyResp.Company.Company)

		updateCompanyReq.Company.Name = nil
		ct := companyconfig.CompanyTypeDemo
		updateCompanyReq.Company.Type = &ct
		err = apiClient.Put(ctx, "/api/companies/"+oid.String(), updateCompanyReq, &updateCompanyResp)
		assert.NoErr(t, err)

		dbCompany, err = tf.CompanyConfigStorage.GetCompany(ctx, oid)
		assert.NoErr(t, err)
		assert.Equal(t, dbCompany.Name, "New Name")
		assert.Equal(t, dbCompany.Type, companyconfig.CompanyTypeDemo)
		assert.Equal(t, *dbCompany, updateCompanyResp.Company.Company)

		err = apiClient.Delete(ctx, "/api/companies/"+oid.String(), nil)
		assert.NoErr(t, err)
	})

	t.Run("TestAllCompanies", func(t *testing.T) {
		ctx := context.Background()
		tf := consoletesthelpers.NewTestFixture(t, consoletesthelpers.ProvisionUniqueDatabase{})

		adminID, cookie, _ := tf.MakeUCAdmin(ctx)
		apiClient := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))

		var companies []companyconfig.Company
		err = apiClient.Get(ctx, "/api/allcompanies", &companies)
		assert.NoErr(t, err)

		assert.Equal(t, len(companies), 1, assert.Must())
		// 1 company ACL (for the UC company) should exist with 1 user/role (the UC company admin)
		assert.Equal(t, companies[0].ID, tf.UserCloudsCompanyID)

		// Create a new "customer" company
		newCompany := companyconfig.NewCompany("NewCompany", companyconfig.CompanyTypeCustomer)

		// Provision the new company with a single new user owner
		testhelpers.ProvisionTestCompany(ctx, t, tf.CompanyConfigStorage, &newCompany, tf.ConsoleTenantDB, tf.ConsoleTenantID, tf.ConsoleTenantCompanyID, provisioning.Owner(adminID))

		err = apiClient.Get(ctx, "/api/allcompanies", &companies)
		assert.NoErr(t, err)
		assert.Equal(t, len(companies), 2, assert.Must())
		var newCompanyPtr *companyconfig.Company
		for _, company := range companies {
			if company.ID == newCompany.ID {
				newCompanyPtr = &company
				break
			}
		}
		assert.NotNil(t, newCompanyPtr, assert.Must())
		assert.Equal(t, newCompanyPtr.Name, newCompany.Name)
	})
}
