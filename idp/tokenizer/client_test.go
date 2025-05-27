package tokenizer_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/storage"
	tokenizerInternal "userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

type testFixture struct {
	tokenizerClient *idp.TokenizerClient
	authzClient     *authz.Client
	storage         *storage.Storage
	userID          uuid.UUID
	tenant          *companyconfig.Tenant
}

func newTestFixture(t *testing.T) *testFixture {
	ctx := context.Background()
	// Provision a test org and tenant
	_, tenant, _, tenantDB, _, _ := testhelpers.CreateTestServer(ctx, t)
	hostname := tenant.GetHostName()

	// Get the storage for this tenant
	s := idptesthelpers.NewStorage(ctx, t, tenantDB, tenant.ID)

	// Set up a test user in the tenant with appropriate privileges
	userID := uuid.Must(uuid.NewV4())
	idptesthelpers.CreateUser(t, tenantDB, userID, uuid.Nil, tenant.ID, tenant.TenantURL)
	// Create the authz client and use it to create some edges for the user
	tokenSource := uctest.TokenSource(t, tenant.TenantURL)
	authzClient, err := authz.NewClient(tenant.TenantURL, authz.JSONClient(jsonclient.TokenSource(tokenSource)))
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, tenant.CompanyID, ucauthz.AdminEdgeTypeID)
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, idpAuthz.PoliciesObjectID, idpAuthz.UserPolicyFullAccessEdgeTypeID)
	assert.NoErr(t, err)

	// Create a JWT for the user and use it to create a tokenizer client
	claims := oidc.UCTokenClaims{}
	claims.Subject = userID.String()
	jwt := uctest.CreateJWT(t, claims, tenant.TenantURL)
	tokenizerClient := idp.NewTokenizerClient(
		tenant.TenantURL,
		idp.JSONClient(jsonclient.HeaderHost(hostname),
			jsonclient.HeaderAuthBearer(jwt),
		))

	return &testFixture{
		tokenizerClient: tokenizerClient,
		authzClient:     authzClient,
		storage:         s,
		userID:          userID,
		tenant:          tenant,
	}
}

func TestClient(t *testing.T) {
	f := newTestFixture(t)
	ctx := context.Background()
	client := f.tokenizerClient
	s := f.storage

	t.Run("TestCreateToken", func(t *testing.T) {
		token, err := client.CreateToken(ctx, "create@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		tr, err := s.GetTokenRecordByToken(ctx, token)
		assert.NoErr(t, err)
		assert.Equal(t, tr.Data, "create@company.com")
	})

	t.Run("ResolveToken", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "IntegrityPolicyTemplate",
			Function: "function policy(context, params) { return context['client']['purpose'] === 'integrity' }",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := &policy.AccessPolicy{
			Name:       "IntegrityPolicy",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt.ID}}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}
		ap, err = client.CreateAccessPolicy(ctx, *ap)
		assert.NoErr(t, err)

		token, err := client.CreateToken(ctx, "resolve@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: ap.ID})
		assert.NoErr(t, err)

		data, err := client.ResolveToken(ctx, token, policy.ClientContext{"purpose": "integrity"}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "resolve@company.com")

		data, err = client.ResolveToken(ctx, token, policy.ClientContext{"purpose": "marketing"}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "")
	})

	t.Run("ResolveTokens", func(t *testing.T) {
		apt1 := &policy.AccessPolicyTemplate{
			Name:     "BatchPolicyTemplatePass",
			Function: "function policy(context, params) { return context['client']['value'] }",
		}
		apt1, err := client.CreateAccessPolicyTemplate(ctx, *apt1)
		assert.NoErr(t, err)

		apt2 := &policy.AccessPolicyTemplate{
			Name:     "BatchPolicyTemplateFail",
			Function: "function policy(context, params) { return !context['client']['value'] }",
		}
		apt2, err = client.CreateAccessPolicyTemplate(ctx, *apt2)
		assert.NoErr(t, err)

		ap1 := &policy.AccessPolicy{
			Name:       "BatchPolicyPass",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt1.ID}}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}
		ap1, err = client.CreateAccessPolicy(ctx, *ap1)
		assert.NoErr(t, err)

		ap2 := &policy.AccessPolicy{
			Name:       "BatchPolicyFail",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt2.ID}}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}
		ap2, err = client.CreateAccessPolicy(ctx, *ap2)
		assert.NoErr(t, err)

		var tokens []string
		// Generate 10 tokens that will resolve and 10 that will not
		for range 10 {
			token, err := client.CreateToken(ctx, "resolve@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: ap1.ID})
			assert.NoErr(t, err)
			tokens = append(tokens, token)
			token, err = client.CreateToken(ctx, "notresolve@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: ap2.ID})
			assert.NoErr(t, err)
			tokens = append(tokens, token)
		}

		data, err := client.ResolveTokens(ctx, tokens, policy.ClientContext{"value": "true"}, nil)
		assert.NoErr(t, err)

		// Verify that only tokens that should get resolved got resolved
		for i := range data {
			if i%2 == 0 {
				assert.Equal(t, data[i], "resolve@company.com")
			} else {
				assert.Equal(t, data[i], "")
			}
		}

	})

	t.Run("TestInspectToken", func(t *testing.T) {
		token, err := client.CreateToken(ctx, "inspect@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		res, err := client.InspectToken(ctx, token)
		assert.NoErr(t, err)
		assert.Equal(t, res.Token, token)
		assert.True(t, res.AccessPolicy.ID == policy.AccessPolicyAllowAll.ID)
		assert.True(t, defaults.TransformerEmail.Equals(storage.NewTransformerFromClient(res.Transformer)))
	})

	t.Run("TestLookupToken", func(t *testing.T) {
		token, err := client.CreateToken(ctx, "lookup@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		got, err := client.LookupTokens(ctx, "lookup@company.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)
		assert.Equal(t, len(got), 1, assert.Must())
		assert.Equal(t, got[0], token)
	})

	t.Run("TestRunAccessPolicy", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TestRunPolicyTemplate",
			Function: "function policy(x, y) { return true; } // TRAP",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := policy.AccessPolicy{
			Name:       "TestRunPolicy",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt.ID}}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}

		resp, err := client.TestAccessPolicy(ctx, ap, policy.AccessPolicyContext{})
		assert.NoErr(t, err)
		assert.True(t, resp.Allowed)
	})

	t.Run("TestRunCheckAttributePolicy", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name: "TestRunCheckAttributeTemplate",
			Function: `function policy(context, params) {
				const id1 = params.userIDUsage === "id1" ? context.user.id : params.id1;
				const id2 = params.userIDUsage === "id2" ? context.user.id : params.id2;
				const attribute = params.attribute;
				if (!id1 || !id2 || !attribute) {
					return false;
				}
				return checkAttribute(id1, id2, attribute);
			} // TRCAP`,
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		templateParameters := fmt.Sprintf(`{"userIDUsage": "id1", "id2": "%s" , "attribute": "_admin"}`, f.tenant.CompanyID)
		apContext := policy.AccessPolicyContext{User: userstore.Record{"id": f.userID.String()}}

		ap_raw := policy.AccessPolicy{
			Name:       "TestRunCheckAttributePolicy",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt.ID}, TemplateParameters: templateParameters}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}

		resp, err := client.TestAccessPolicy(ctx, ap_raw, apContext)
		assert.NoErr(t, err)
		assert.True(t, resp.Allowed)

		ap_native := policy.AccessPolicy{
			Name:       "TestRunCheckAttributePolicy",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: policy.AccessPolicyTemplateCheckAttribute.ID}, TemplateParameters: templateParameters}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}

		resp, err = client.TestAccessPolicy(ctx, ap_native, apContext)
		assert.NoErr(t, err)
		assert.True(t, resp.Allowed)

		// Test calling checkAttribute with invalid parameters -- we expect the JS context to throw an exception
		// so that we get an error when the policy is tested
		apt.Function = `function policy(context, params) {
			return checkAttribute("a", "b", "c");
		} // TRCAP`
		apt, err = client.UpdateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)
		_, err = client.TestAccessPolicy(ctx, ap_raw, apContext)
		assert.NotNil(t, err)

		// Do the same test above, but this time we expect the policy to return false since the call to
		// checkAttribute is wrapped by try/catch
		apt.Function = `function policy(context, params) {
			try {
				return checkAttribute("a", "b", "c");
			} catch (e) {
				return false;
			}
		} // TRCAP`
		_, err = client.UpdateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)
		resp, err = client.TestAccessPolicy(ctx, ap_raw, apContext)
		assert.NoErr(t, err)
		assert.False(t, resp.Allowed)
	})

	t.Run("TestListAccessPolicies", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TestListPolicyTemplate",
			Function: "function policy(x, y) { return true; } // TLAP",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TruePolicy",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.NoErr(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, f.authzClient, ap))

		aps, err := client.ListAccessPolicies(ctx, false)
		assert.NoErr(t, err)
		// we don't assert length because other tests could create access policies in parallel
		var count int
		for _, a := range aps.Data {
			if a.ID == ap.ID {
				count++
			}
		}
		assert.Equal(t, count, 1)
	})

	t.Run("TestGetAccessPolicyTemplate", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TestGetAccessPolicyTemplate",
			Function: "function policy(x, y) { return true; } // TGAPT",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		// get via ID
		got, err := client.GetAccessPolicyTemplate(ctx, userstore.ResourceID{ID: apt.ID})
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt.ID)
		assert.Equal(t, got.Name, apt.Name)
		assert.Equal(t, got.Function, apt.Function)
		assert.Equal(t, got.Version, apt.Version)

		// get via Name
		got, err = client.GetAccessPolicyTemplate(ctx, userstore.ResourceID{Name: apt.Name})
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt.ID)
		assert.Equal(t, got.Name, apt.Name)
		assert.Equal(t, got.Function, apt.Function)
		assert.Equal(t, got.Version, apt.Version)

		// create new version
		got.Function = "function policy(x, y) { return false; } // TGAPT"
		_, err = client.UpdateAccessPolicyTemplate(ctx, *got)
		assert.NoErr(t, err)

		// get old version by ID
		got, err = client.GetAccessPolicyTemplateByVersion(ctx, userstore.ResourceID{ID: apt.ID}, apt.Version)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt.ID)
		assert.Equal(t, got.Name, apt.Name)
		assert.Equal(t, got.Function, apt.Function)
		assert.Equal(t, got.Version, apt.Version)

		// get old version by Name
		got, err = client.GetAccessPolicyTemplateByVersion(ctx, userstore.ResourceID{Name: apt.Name}, apt.Version)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt.ID)
		assert.Equal(t, got.Name, apt.Name)
		assert.Equal(t, got.Function, apt.Function)
		assert.Equal(t, got.Version, apt.Version)
	})

	t.Run("TestGetAccessPolicy", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TrueTemplate1",
			Function: "function policy(x, y) { return true; } // TGAP",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TruePolicy1",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.NoErr(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, f.authzClient, ap))
		assert.NotEqual(t, ap.ID, uuid.Nil)
		assert.Equal(t, ap.Version, 0)

		// get via ID
		got, err := client.GetAccessPolicy(ctx, userstore.ResourceID{ID: ap.ID})
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, ap.Name)
		assert.Equal(t, len(got.Components), 1)
		assert.Equal(t, got.Components[0].Template.ID, apt.ID)
		assert.Equal(t, got.Version, ap.Version)

		// get via Name
		got, err = client.GetAccessPolicy(ctx, userstore.ResourceID{Name: ap.Name})
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, ap.Name)
		assert.Equal(t, len(got.Components), 1)
		assert.Equal(t, got.Components[0].Template.ID, apt.ID)
		assert.Equal(t, got.Version, ap.Version)

		// Create a JWT for another user and use it to create a tokenizer client
		claims := oidc.UCTokenClaims{}
		claims.Subject = uuid.Must(uuid.NewV4()).String()
		jwt := uctest.CreateJWT(t, claims, f.tenant.TenantURL)
		client2 := idp.NewTokenizerClient(
			f.tenant.TenantURL,
			idp.JSONClient(jsonclient.HeaderHost(f.tenant.GetHostName()),
				jsonclient.HeaderAuthBearer(jwt)))

		// Verify that the other user cannot get the access policy
		_, err = client2.GetAccessPolicy(ctx, userstore.ResourceID{ID: ap.ID})
		assert.NotNil(t, err)
	})

	t.Run("TestGetAccessPolicyByVersion", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TrueTemplate2",
			Function: "function policy(x, y) { return true; } // TGAPBV",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TruePolicy2",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.NoErr(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, f.authzClient, ap))
		assert.Equal(t, ap.Version, 0)
		assert.NotEqual(t, ap.ID, uuid.Nil)

		apt2 := &policy.AccessPolicyTemplate{
			Name:     "FalseTemplate",
			Function: "function policy(x, y) { return false; } // TGAPBV2",
		}
		apt2, err = client.CreateAccessPolicyTemplate(ctx, *apt2)
		assert.NoErr(t, err)

		ap.ComponentIDs = []uuid.UUID{apt2.ID}
		ap.Name = "FalsePolicy"
		ap.Version++
		assert.NoErr(t, s.SaveAccessPolicy(ctx, ap))
		assert.Equal(t, ap.Version, 1)
		assert.NotEqual(t, ap.ID, uuid.Nil)

		// get by ID
		got, err := client.GetAccessPolicyByVersion(ctx, userstore.ResourceID{ID: ap.ID}, 0)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, "TruePolicy2")
		assert.Equal(t, got.Components[0].Template.ID, apt.ID)
		assert.Equal(t, got.Version, 0)

		got, err = client.GetAccessPolicyByVersion(ctx, userstore.ResourceID{ID: ap.ID}, 1)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, "FalsePolicy")
		assert.Equal(t, got.Components[0].Template.ID, apt2.ID)
		assert.Equal(t, got.Version, 1)

		// get by Name
		got, err = client.GetAccessPolicyByVersion(ctx, userstore.ResourceID{Name: "TruePolicy2"}, 0)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, "TruePolicy2")
		assert.Equal(t, got.Components[0].Template.ID, apt.ID)
		assert.Equal(t, got.Version, 0)

		got, err = client.GetAccessPolicyByVersion(ctx, userstore.ResourceID{Name: "FalsePolicy"}, 1)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Name, "FalsePolicy")
		assert.Equal(t, got.Components[0].Template.ID, apt2.ID)
		assert.Equal(t, got.Version, 1)
	})

	t.Run("TestRunAccessPolicyTemplate", func(t *testing.T) {
		apt := policy.AccessPolicyTemplate{
			Name:     "TestRunPolicyTemplate",
			Function: "function policy(x, y) { return true; } // TRAP",
		}

		resp, err := client.TestAccessPolicyTemplate(ctx, apt, policy.AccessPolicyContext{User: userstore.Record{"id": f.userID.String()}}, "{}")
		assert.NoErr(t, err)
		assert.True(t, resp.Allowed)
	})

	t.Run("TestCreateAccessPolicy", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TrueTemplate3",
			Function: "function policy(x, y) { return true; } // TCAP",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := policy.AccessPolicy{
			Name:       "TruePolicy3",
			Components: []policy.AccessPolicyComponent{{Template: &userstore.ResourceID{ID: apt.ID}}},
			PolicyType: policy.PolicyTypeCompositeAnd,
		}

		got, err := client.CreateAccessPolicy(ctx, ap)
		assert.NoErr(t, err)
		assert.Equal(t, got.Name, ap.Name)
		assert.Equal(t, got.Components[0].Template.ID, apt.ID)
		assert.NotEqual(t, got.ID, uuid.Nil)
	})

	t.Run("TestDeleteAccessPolicy", func(t *testing.T) {
		apt := &policy.AccessPolicyTemplate{
			Name:     "TrueTemplate4",
			Function: "function policy(x, y) { return true; } // TDAP",
		}
		apt, err := client.CreateAccessPolicyTemplate(ctx, *apt)
		assert.NoErr(t, err)

		ap := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TruePolicy4",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}

		assert.NoErr(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, f.authzClient, ap))
		ap.Name = "TruePolicy4v2"
		ap.Version++
		assert.NoErr(t, s.SaveAccessPolicy(ctx, ap))

		assert.NoErr(t, client.DeleteAccessPolicy(ctx, ap.ID, ap.Version))

		got, err := s.GetLatestAccessPolicy(ctx, ap.ID)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, ap.ID)
		assert.Equal(t, got.Version, ap.Version-1)
	})

	t.Run("TestRunTransformer", func(t *testing.T) {
		transformer := policy.Transformer{
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(x, y) { return 'xyz'; } // TRGP",
		}

		resp, err := client.TestTransformer(ctx, "foo", transformer)
		assert.NoErr(t, err)
		assert.Equal(t, resp.Value, "xyz")
	})

	t.Run("TestRunTransformerArray", func(t *testing.T) {
		transformer := policy.Transformer{
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(data, params) { return data; }",
		}

		resp, err := client.TestTransformer(ctx, `["555-555-5555", "123 Evergreen Terrace"]`, transformer)
		assert.NoErr(t, err)
		assert.Equal(t, resp.Value, `["555-555-5555","123 Evergreen Terrace"]`)
	})

	t.Run("TestRunTransformerObject", func(t *testing.T) {
		transformer := policy.Transformer{
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(data, params) { return data; }",
		}

		resp, err := client.TestTransformer(ctx, `{"0": "555-555-5555", "1":"123 Evergreen Terrace"}`, transformer)
		assert.NoErr(t, err)
		assert.Equal(t, resp.Value, `{"0":"555-555-5555","1":"123 Evergreen Terrace"}`)
	})

	t.Run("TestCreateAndUpdateTransformer", func(t *testing.T) {
		transformer := policy.Transformer{
			Name:           "PolicyXYZ",
			Description:    "a test transformer",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(x, y) { return 'xyz'; } // TGAP",
		}

		tfmr, err := client.CreateTransformer(ctx, transformer)
		assert.NoErr(t, err)
		assert.Equal(t, tfmr.Name, transformer.Name)
		assert.Equal(t, tfmr.Description, transformer.Description)
		assert.Equal(t, tfmr.InputDataType, transformer.InputDataType)
		assert.Equal(t, tfmr.OutputDataType, transformer.OutputDataType)
		assert.Equal(t, tfmr.TransformType, transformer.TransformType)
		assert.Equal(t, tfmr.TagIDs, transformer.TagIDs)
		assert.Equal(t, tfmr.Function, transformer.Function)
		assert.Equal(t, tfmr.Parameters, transformer.Parameters)
		assert.Equal(t, tfmr.IsSystem, transformer.IsSystem)

		// get by ID
		got, err := client.GetTransformer(ctx, userstore.ResourceID{ID: tfmr.ID})
		assert.NoErr(t, err)
		assert.Equal(t, tfmr.Name, got.Name)
		assert.Equal(t, tfmr.Description, got.Description)
		assert.Equal(t, tfmr.InputDataType, got.InputDataType)
		assert.Equal(t, tfmr.OutputDataType, got.OutputDataType)
		assert.Equal(t, tfmr.TransformType, got.TransformType)
		assert.Equal(t, tfmr.TagIDs, got.TagIDs)
		assert.Equal(t, tfmr.Function, got.Function)
		assert.Equal(t, tfmr.Parameters, got.Parameters)
		assert.Equal(t, tfmr.IsSystem, got.IsSystem)

		// get by Name
		got, err = client.GetTransformer(ctx, userstore.ResourceID{Name: tfmr.Name})
		assert.NoErr(t, err)
		assert.Equal(t, tfmr.Name, got.Name)
		assert.Equal(t, tfmr.Description, got.Description)
		assert.Equal(t, tfmr.InputDataType, got.InputDataType)
		assert.Equal(t, tfmr.OutputDataType, got.OutputDataType)
		assert.Equal(t, tfmr.TransformType, got.TransformType)
		assert.Equal(t, tfmr.TagIDs, got.TagIDs)
		assert.Equal(t, tfmr.Function, got.Function)
		assert.Equal(t, tfmr.Parameters, got.Parameters)
		assert.Equal(t, tfmr.IsSystem, got.IsSystem)

		// update
		tfmr.Description = "an updated test transformer"
		tfmr.Function = "function transform(x, y) { return 'zyx'; } // TGAP"
		tfmr2, err := client.UpdateTransformer(ctx, *tfmr)
		assert.NoErr(t, err)
		assert.Equal(t, tfmr2.Name, tfmr.Name)
		assert.Equal(t, tfmr2.Description, tfmr.Description)
		assert.Equal(t, tfmr2.Version, tfmr.Version+1)

		// get by ID
		got, err = client.GetTransformer(ctx, userstore.ResourceID{ID: tfmr.ID})
		assert.NoErr(t, err)
		assert.Equal(t, tfmr2.Name, got.Name)
		assert.Equal(t, tfmr2.Description, got.Description)
		assert.Equal(t, tfmr2.Version, got.Version)

		// get old version by ID
		got, err = client.GetTransformerByVersion(ctx, userstore.ResourceID{ID: tfmr.ID}, tfmr.Version)
		assert.NoErr(t, err)
		assert.Equal(t, transformer.Name, got.Name)
		assert.Equal(t, transformer.Description, got.Description)
		assert.Equal(t, transformer.Version, got.Version)

		// get old version by name and ID
		got, err = client.GetTransformerByVersion(ctx, userstore.ResourceID{Name: tfmr.Name}, tfmr.Version)
		assert.NoErr(t, err)
		assert.Equal(t, transformer.Name, got.Name)
		assert.Equal(t, transformer.Description, got.Description)
		assert.Equal(t, transformer.Version, got.Version)

		// update the name
		tfmr2.Name = "PolicyXYZ2"
		tfmr3, err := client.UpdateTransformer(ctx, *tfmr2)
		assert.NoErr(t, err)
		assert.Equal(t, tfmr3.Name, tfmr2.Name)

		// make sure the old versions also have the new name
		got, err = client.GetTransformerByVersion(ctx, userstore.ResourceID{ID: tfmr.ID}, tfmr.Version)
		assert.NoErr(t, err)
		assert.Equal(t, tfmr3.Name, got.Name)

		got, err = client.GetTransformerByVersion(ctx, userstore.ResourceID{ID: tfmr2.ID}, tfmr2.Version)
		assert.NoErr(t, err)
		assert.Equal(t, tfmr3.Name, got.Name)
	})

	t.Run("TestListTransformers", func(t *testing.T) {
		transformer := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "PolicyXYZ1",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(x, y) { return 'xyz'; } // TLGP",
		}
		_, err := client.CreateTransformer(ctx, transformer)
		assert.NoErr(t, err)

		gps, err := client.ListTransformers(ctx)
		assert.NoErr(t, err)
		// we don't assert length because other tests could create transformers in parallel
		var count int
		for _, g := range gps.Data {
			if g.ID == transformer.ID {
				count++
			}
		}
		assert.Equal(t, count, 1)
	})

	t.Run("TestDeleteTransformer", func(t *testing.T) {
		transformer := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "PolicyXYZ3",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function transform(x, y) { return 'xyz'; } // TDGP",
		}
		_, err := client.CreateTransformer(ctx, transformer)
		assert.NoErr(t, err)

		assert.NoErr(t, client.DeleteTransformer(ctx, transformer.ID))

		_, err = s.GetLatestTransformer(ctx, transformer.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("TestEmailPolicy", func(t *testing.T) {
		token, err := client.CreateToken(ctx, "user@testdomain.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		emailParts := strings.Split(token, "@")
		assert.Equal(t, len(emailParts), 2)
		assert.Equal(t, len(emailParts[0]), 12)
		domainParts := strings.Split(emailParts[1], ".")
		assert.Equal(t, len(domainParts), 2)
		assert.Equal(t, len(domainParts[0]), 6)
		assert.Equal(t, domainParts[1], "com")

		data, err := client.ResolveToken(ctx, token, nil, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "user@testdomain.com")

		token, err = client.CreateToken(ctx, "user@gmail.com", userstore.ResourceID{ID: policy.TransformerEmail.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		emailParts = strings.Split(token, "@")
		assert.Equal(t, len(emailParts), 2)
		assert.Equal(t, len(emailParts[0]), 12)
		domainParts = strings.Split(emailParts[1], ".")
		assert.Equal(t, len(domainParts), 2)
		assert.Equal(t, domainParts[0], "gmail")
		assert.Equal(t, domainParts[1], "com")

		data, err = client.ResolveToken(ctx, token, nil, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "user@gmail.com")

		// Create custom email policy directly on token creation
		ps, err := client.ListTransformers(ctx)
		assert.NoErr(t, err)
		var transformer = &policy.TransformerEmail
		for _, p := range ps.Data {
			if p.ID == policy.TransformerEmail.ID {
				transformer = &p
				break
			}
		}

		transformer.Parameters = `[{
			"PreserveValue": false,
			"PreserveChars": 2,
			"FinalLength": 10
		  }, {
			"PreserveValue": false,
			"PreserveCommonValue": true,
			"PreserveChars": 0,
			"FinalLength": 6
		  }, {
			"PreserveValue": true
		  }]`
		transformer.ID = uuid.Nil
		transformer.Name = "_" + transformer.ID.String()
		transformer.InputDataType = datatype.String
		transformer.OutputDataType = datatype.String
		transformer.TransformType = policy.TransformTypeTokenizeByValue
		transformer, err = client.CreateTransformer(ctx, *transformer)
		assert.NoErr(t, err)

		token, err = client.CreateToken(ctx, "user@hotmail.com", userstore.ResourceID{ID: transformer.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		emailParts = strings.Split(token, "@")
		assert.Equal(t, len(emailParts), 2)
		assert.Equal(t, len(emailParts[0]), 10)
		assert.Equal(t, emailParts[0][0:2], "us")
		domainParts = strings.Split(emailParts[1], ".")
		assert.Equal(t, len(domainParts), 2)
		assert.Equal(t, domainParts[0], "hotmail")
		assert.Equal(t, domainParts[1], "com")

		data, err = client.ResolveToken(ctx, token, nil, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "user@hotmail.com")

		// Create the new policy first and refer to it
		transformer.Parameters = `[{
			"PreserveValue": false,
			"PreserveChars": 4,
			"FinalLength": 10
		  }, {
			"PreserveValue": false,
			"PreserveCommonValue": true,
			"PreserveChars": 0,
			"FinalLength": 8
		  }, {
			"PreserveValue": true
		  }]`
		transformer.ID = uuid.Nil
		transformer.Name = "_" + uuid.Must(uuid.NewV4()).String()
		transformer, err = client.CreateTransformer(ctx, *transformer)
		assert.NoErr(t, err)

		token, err = client.CreateToken(ctx, "user@yahoo.com", userstore.ResourceID{ID: transformer.ID}, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID})
		assert.NoErr(t, err)

		emailParts = strings.Split(token, "@")
		assert.Equal(t, len(emailParts), 2)
		assert.Equal(t, len(emailParts[0]), 10)
		assert.Equal(t, emailParts[0][0:4], "user")
		domainParts = strings.Split(emailParts[1], ".")
		assert.Equal(t, len(domainParts), 2)
		assert.Equal(t, domainParts[0], "yahoo")
		assert.Equal(t, domainParts[1], "com")

		data, err = client.ResolveToken(ctx, token, nil, nil)
		assert.NoErr(t, err)
		assert.Equal(t, data, "user@yahoo.com")

	})
}
