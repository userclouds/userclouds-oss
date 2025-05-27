package api_test

import (
	"context"
	"fmt"
	"testing"

	"userclouds.com/console/internal/api"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	plextest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/plex/manager"
	"userclouds.com/worker"
)

func TestTenant(t *testing.T) {
	ctx := context.Background()

	tc := workerclient.NewTestClient()

	// the client with worker is used by the last test,
	// everything else provisions its own client to
	// eliminate races
	tf, c := newAPIClientWithWorkerClient(ctx, t, tc)

	// currently this test just roundtrips the same config -- it doesn't actually make sure it
	// twiddles any bits, but it makes sure we don't break things like cmp.AllowUnexported (which
	// I did just break recently)
	t.Run("TestSaveTenant", func(t *testing.T) {
		c := newAPIClientForFixture(ctx, t, tf)

		mgr := manager.NewFromDB(tf.ConsoleTenantDB, cachetesthelpers.NewCacheConfig())
		tp, err := mgr.GetTenantPlex(ctx, tf.ConsoleTenantID)
		assert.NoErr(t, err)

		req := api.SaveTenantPlexConfigRequest{
			TenantConfig: tp.PlexConfig,
		}
		// TODO (sgarrity 8/24): this pattern is so broken
		assert.NoErr(t, req.TenantConfig.UpdateUISettings(ctx))

		var res api.SaveTenantPlexConfigResponse

		assert.IsNil(t, c.Post(ctx, fmt.Sprintf("/api/tenants/%v/plexconfig", tf.ConsoleTenantID), req, &res), assert.Must())
	})

	t.Run("TestRenameTenant", func(t *testing.T) {
		c := newAPIClientForFixture(ctx, t, tf)

		// set up a tenant to rename
		tenant := &companyconfig.Tenant{
			BaseModel: ucdb.NewBase(),
			Name:      "test tenant",
			CompanyID: tf.UserCloudsCompanyID,
			TenantURL: "http://testtenant.com",
		}
		assert.NoErr(t, tf.CompanyConfigStorage.SaveTenant(ctx, tenant))

		req := api.UpdateTenantRequest{
			Tenant: *tenant,
		}
		updatedName := "updated"
		req.Tenant.Name = updatedName

		var res api.SelectedTenantInfo
		assert.NoErr(t, c.Put(ctx, fmt.Sprintf("/api/companies/%v/tenants/%v", tf.UserCloudsCompanyID, tenant.ID), req, &res))

		assert.Equal(t, res.Tenant.Name, updatedName)

		got, err := tf.CompanyConfigStorage.GetTenant(ctx, tenant.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got.Name, updatedName)
	})

	t.Run("TestSaveTenantConfigWithExistingPageParametersAndMessageElements", func(t *testing.T) {
		c := newAPIClientForFixture(ctx, t, tf)

		// set up a customized message element
		tc := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).
			CustomizeMessageElement(message.EmailResetPassword, message.EmailSubjectTemplate, "test subject").
			Build()
		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tc, message.EmailResetPassword)
		assert.IsNil(t, postSaveTenantAppEmailElements(mmtme, tf, c), assert.Must())

		// set up customized page parameters
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password").
			SetAppPageParameter(pagetype.PlexCreateUserPage, param.EmailLabel, "new label").
			Build()
		sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		var appr api.AppPageParametersResponse
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

		// save tenant config
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).
			AddAllowedLogoutURI("http://nowhere.org").
			Build()

		// remove all message elements and page parameters to simulate front end behavior
		tc.PlexMap.Apps[0].MessageElements = nil
		tc.PlexMap.Apps[0].PageParameters = nil

		stpcr := api.SaveTenantPlexConfigRequest{TenantConfig: tc}
		var response api.SaveTenantPlexConfigResponse
		assert.IsNil(t, c.Post(ctx, fmt.Sprintf("/api/tenants/%v/plexconfig", tf.ConsoleTenantID), stpcr, &response), assert.Must())

		// verify that modified fields are there, but that message elements and page parameters are not since they are filtered out of response
		assert.Equal(t, len(response.TenantConfig.PlexMap.Apps[0].AllowedLogoutURIs), 1, assert.Must())
		assert.Equal(t, response.TenantConfig.PlexMap.Apps[0].AllowedLogoutURIs[0], "http://nowhere.org")
		assert.Equal(t, len(response.TenantConfig.PlexMap.Apps[0].MessageElements), 0)
		assert.Equal(t, len(response.TenantConfig.PlexMap.Apps[0].PageParameters), 0)

		// ensure updated message elements and page parameters are still present in saved tenant config
		tc = loadTenantConfig(t, tf)
		assert.Equal(t, tc.PlexMap.Apps[0].MessageElements[message.EmailResetPassword][message.EmailSubjectTemplate], "test subject")
		assert.Equal(t, tc.PlexMap.Apps[0].PageParameters[pagetype.EveryPage][param.AuthenticationMethods].Value, "password")
		assert.Equal(t, tc.PlexMap.Apps[0].PageParameters[pagetype.PlexCreateUserPage][param.EmailLabel].Value, "new label")
	})

	t.Run("TestCreateTenant", func(t *testing.T) {
		req := api.CreateTenantRequest{
			Tenant: companyconfig.Tenant{
				BaseModel: ucdb.NewBase(),
				Name:      "testtenant",
			},
		}

		var res companyconfig.Tenant
		assert.NoErr(t, c.Post(ctx, fmt.Sprintf("/api/companies/%v/tenants", tf.UserCloudsCompanyID), req, &res))
		assert.Equal(t, res.Name, req.Tenant.Name)
		msgs := tc.GetMessages()
		assert.Equal(t, len(msgs), 1, assert.Must())
		assert.Equal(t, msgs[0].Task, worker.TaskCreateTenant)
		assert.Equal(t, msgs[0].CreateTenant.Tenant.ID, req.Tenant.ID)
	})
}
