package tokenizer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	tokenizerInternal "userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

type testFixture struct {
	t               *testing.T
	tenant          *companyconfig.Tenant
	tokenizerClient *idp.TokenizerClient
	authzClient     *authz.Client
	storage         *storage.Storage
	handler         *uchttp.ServeMux
	userID          uuid.UUID
	authHeader      string
	userID2         uuid.UUID
	authHeader2     string

	tenantDB *ucdb.DB
}

func newTestFixture(t *testing.T) *testFixture {
	ctx := context.Background()

	// Provision a test org and tenant
	_, tenant, _, tenantDB, handler, _ := testhelpers.CreateTestServer(ctx, t)
	hostname := tenant.GetHostName()
	s := idptesthelpers.NewStorage(ctx, t, tenantDB, tenant.ID)

	// Set up a couple of test users in the tenant with appropriate privileges
	userID := uuid.Must(uuid.NewV4())
	idptesthelpers.CreateUser(t, tenantDB, userID, uuid.Nil, tenant.ID, tenant.TenantURL)
	userID2 := uuid.Must(uuid.NewV4())
	idptesthelpers.CreateUser(t, tenantDB, userID2, uuid.Nil, tenant.ID, tenant.TenantURL)

	// Create the authz client and use it to create an edge for the user
	tokenSource := uctest.TokenSource(t, tenant.TenantURL)
	authzClient, err := authz.NewClient(tenant.TenantURL, authz.JSONClient(jsonclient.TokenSource(tokenSource)))
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, tenant.CompanyID, ucauthz.AdminEdgeTypeID)
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, idpAuthz.PoliciesObjectID, idpAuthz.UserPolicyFullAccessEdgeTypeID)
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID2, tenant.CompanyID, ucauthz.AdminEdgeTypeID)
	assert.NoErr(t, err)
	_, err = authzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID2, idpAuthz.PoliciesObjectID, idpAuthz.UserPolicyFullAccessEdgeTypeID)
	assert.NoErr(t, err)

	// Create a JWT for the user and use it to create a tokenizer client
	claims := oidc.UCTokenClaims{}
	claims.Subject = userID.String()
	jwt := uctest.CreateJWT(t, claims, tenant.TenantURL)
	authHeader := fmt.Sprintf("Bearer %s", jwt)
	tokenizerClient := idp.NewTokenizerClient(
		tenant.TenantURL,
		idp.JSONClient(jsonclient.HeaderHost(hostname),
			jsonclient.HeaderAuthBearer(jwt)))

	// Create a second JWT for test cases needing a separate user
	claims.Subject = userID2.String()
	jwt = uctest.CreateJWT(t, claims, tenant.TenantURL)
	authHeader2 := fmt.Sprintf("Bearer %s", jwt)

	return &testFixture{
		t:               t,
		tenant:          tenant,
		tokenizerClient: tokenizerClient,
		authzClient:     authzClient,
		storage:         s,
		tenantDB:        tenantDB,
		handler:         handler,
		userID:          userID,
		authHeader:      authHeader,
		userID2:         userID2,
		authHeader2:     authHeader2,
	}
}

func (tf *testFixture) runTokenizerRequestWithHeader(method, target string, requestBody any, authHeader string) *httptest.ResponseRecorder {
	tf.t.Helper()
	var reader io.Reader
	if requestBody != nil {
		reader = uctest.IOReaderFromJSONStruct(tf.t, requestBody)
	} else {
		reader = nil
	}
	r := httptest.NewRequest(method, target, reader)
	r.Host = tf.tenant.GetHostName()
	r.Header.Set("Authorization", authHeader)
	rr := httptest.NewRecorder()
	tf.handler.ServeHTTP(rr, r)
	return rr
}
func (tf *testFixture) runTokenizerRequest(method, target string, requestBody any) *httptest.ResponseRecorder {
	tf.t.Helper()
	return tf.runTokenizerRequestWithHeader(method, target, requestBody, tf.authHeader)
}

func TestAccessPolicyHandler(t *testing.T) {
	tf := newTestFixture(t)
	ctx := context.Background()
	s := tf.storage

	// sort function
	sortAPs := func(expected []policy.AccessPolicy) func(i, j int) bool {
		return func(i, j int) bool { return expected[i].ID.String() < expected[j].ID.String() }
	}

	t.Run("TestList", func(t *testing.T) {
		apt := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template_For_List",
			Function:                 "{} // template_for_list",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt), assert.Must())

		expected := []policy.AccessPolicy{}
		for _, p := range defaults.GetDefaultAccessPolicies() {
			expected = append(expected, *p.ToClientModel())
		}
		total_versioned := len(expected)

		for i := 1; i <= pagination.DefaultLimit+10; i++ {
			ap := &storage.AccessPolicy{
				SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
				Name:                     fmt.Sprintf("PolicyList%d", i),
				ComponentIDs:             []uuid.UUID{apt.ID},
				ComponentParameters:      []string{""},
				ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
				PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
			}
			assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap), assert.Must())

			total_versioned++
			if i%2 == 0 {
				ap.Version = 1
				assert.IsNil(t, s.SaveAccessPolicy(ctx, ap), assert.Must())
				total_versioned++
			}
			expected = append(expected, *ap.ToClientModel())
		}
		sort.Slice(expected, sortAPs(expected))
		rr := tf.runTokenizerRequest(http.MethodGet, paths.BaseAccessPolicyPath, nil)

		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		got := []policy.AccessPolicy{}

		var resp idp.ListAccessPoliciesResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), pagination.DefaultLimit)
		assert.True(t, resp.ResponseFields.HasNext)
		got = append(got, resp.Data...)

		rr = tf.runTokenizerRequest(http.MethodGet, fmt.Sprintf("%s?starting_after=%s", paths.BaseAccessPolicyPath, resp.ResponseFields.Next), nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), len(expected)-pagination.DefaultLimit)
		assert.False(t, resp.ResponseFields.HasNext)
		got = append(got, resp.Data...)

		sort.Slice(got, sortAPs(got))
		for i := range got {
			assert.True(t, got[i].ID == expected[i].ID && got[i].Name == expected[i].Name && got[i].Version == expected[i].Version)
		}
		rr = tf.runTokenizerRequest(http.MethodGet, fmt.Sprintf("%s?versioned=true&limit=200", paths.BaseAccessPolicyPath), nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), total_versioned)
	})

	t.Run("TestGetWithMultiplePolicies", func(t *testing.T) {
		apt1 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template_1",
			Function:                 "function bar() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt1), assert.Must())

		ap1 := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "PolicyGet1",
			ComponentIDs:             []uuid.UUID{apt1.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap1), assert.Must())

		apt2 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template1",
			Function:                 "function baz() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt2), assert.Must())

		ap2 := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "PolicyGet2",
			ComponentIDs:             []uuid.UUID{apt2.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap2), assert.Must())

		type testcase struct {
			ID         uuid.UUID
			Name       string
			TemplateID uuid.UUID
			Version    int
			URL        string
		}
		cases := []testcase{
			{
				// Get without version, when only one version exists
				Name:       "PolicyGet1",
				TemplateID: apt1.ID,
				Version:    0,
				URL:        paths.GetAccessPolicy(ap1.ID),
			},
			{
				Name:       "PolicyGet2",
				TemplateID: apt2.ID,
				Version:    0,
				URL:        paths.GetAccessPolicy(ap2.ID),
			},
		}

		for _, c := range cases {
			rr := tf.runTokenizerRequest(http.MethodGet, c.URL, nil)
			assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

			var resp policy.AccessPolicy
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Name, c.Name)
			assert.Equal(t, len(resp.Components), 1)
			assert.Equal(t, resp.Components[0].Template.ID, c.TemplateID)
			assert.Equal(t, resp.Version, c.Version)
		}

		ap2.ComponentIDs = []uuid.UUID{apt1.ID}
		ap2.Version++
		assert.IsNil(t, s.SaveAccessPolicy(ctx, ap2), assert.Must())

		cases = []testcase{
			{
				// Get without version when a second version exists (should return latest)
				Name:       "PolicyGet2",
				TemplateID: apt1.ID,
				Version:    1,
				URL:        paths.GetAccessPolicy(ap2.ID),
			},
			{
				// Get with version
				Name:       "PolicyGet2",
				TemplateID: apt2.ID,
				Version:    0,
				URL:        paths.GetAccessPolicyByVersion(ap2.ID, 0),
			},
			{
				Name:       "PolicyGet2",
				TemplateID: apt1.ID,
				Version:    1,
				URL:        paths.GetAccessPolicyByVersion(ap2.ID, 1),
			},
		}

		for _, c := range cases {
			rr := tf.runTokenizerRequest(http.MethodGet, c.URL, nil)
			assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

			var resp policy.AccessPolicy
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Name, c.Name)
			assert.Equal(t, len(resp.Components), 1)
			assert.Equal(t, resp.Components[0].Template.ID, c.TemplateID)
			assert.Equal(t, resp.Version, c.Version)
		}
	})

	t.Run("TestGet", func(t *testing.T) {
		apt1 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template4",
			Function:                 "function ap2() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt1), assert.Must())

		ap1 := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Policy4",
			ComponentIDs:             []uuid.UUID{apt1.ID},
			ComponentParameters:      []string{`{"a":1}`},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap1), assert.Must())
		rr := tf.runTokenizerRequestWithHeader(http.MethodGet, paths.GetAccessPolicy(ap1.ID), nil, tf.authHeader2)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		var resp policy.AccessPolicy
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, resp.ID, ap1.ID)
		assert.Equal(t, resp.Components[0].Template.ID, apt1.ID)
		assert.Equal(t, resp.Components[0].TemplateParameters, `{"a":1}`)

		// update the policy
		ap1.Version++
		ap1.ComponentParameters = []string{`{"a":2}`}
		assert.IsNil(t, s.SaveAccessPolicy(ctx, ap1), assert.Must())

		// make sure we're getting latest version
		rr = tf.runTokenizerRequestWithHeader(http.MethodGet, paths.GetAccessPolicy(ap1.ID), nil, tf.authHeader2)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, resp.ID, ap1.ID)
		assert.Equal(t, resp.Components[0].Template.ID, apt1.ID)
		assert.Equal(t, resp.Components[0].TemplateParameters, `{"a":2}`)
	})

	t.Run("TestUpdate", func(t *testing.T) {
		apt := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TemplateTestUpdate",
			Function:                 "function ap1() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt), assert.Must())

		ap1 := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "PolicyTestUpdate",
			ComponentIDs:             []uuid.UUID{apt.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap1), assert.Must())

		type testcase struct {
			ap      policy.AccessPolicy
			code    int
			version int
		}

		cases := []testcase{
			{
				ap: policy.AccessPolicy{
					Name: ap1.Name,
					Components: []policy.AccessPolicyComponent{
						{Template: &userstore.ResourceID{ID: apt.ID}, TemplateParameters: "{}"},
					},
					PolicyType: policy.PolicyTypeCompositeAnd,
				},
				code:    http.StatusOK,
				version: 0,
			},
			{
				ap: policy.AccessPolicy{
					// make sure including an ID doesn't hose us
					ID:   ap1.ID,
					Name: ap1.Name,
					Components: []policy.AccessPolicyComponent{
						{Template: &userstore.ResourceID{ID: apt.ID}, TemplateParameters: `{"a":1}`},
					},
					PolicyType: policy.PolicyTypeCompositeAnd,
				},
				code:    http.StatusOK,
				version: 1,
			},
			{
				ap: policy.AccessPolicy{
					// ID mismatch witH URL should barf
					ID:         uuid.Must(uuid.NewV4()),
					Name:       ap1.Name,
					PolicyType: policy.PolicyTypeCompositeAnd,
				},
				code:    http.StatusBadRequest,
				version: 2,
			}, {
				ap: policy.AccessPolicy{
					ID:         ap1.ID,
					Name:       ap1.Name,
					PolicyType: policy.PolicyTypeCompositeAnd,
					Components: []policy.AccessPolicyComponent{
						{Policy: &userstore.ResourceID{ID: ap1.ID}},
					},
				},
				code:    http.StatusBadRequest, // loop in policy: policy component points to itself
				version: 2,
			},
		}

		for i, c := range cases {
			req := tokenizer.UpdateAccessPolicyRequest{
				AccessPolicy: c.ap,
			}
			req.AccessPolicy.Version = c.version
			rr := tf.runTokenizerRequest(http.MethodPut, paths.UpdateAccessPolicy(ap1.ID), req)
			assert.Equal(t, rr.Code, c.code, assert.Must())

			// don't expect a response body on failure
			if c.code != http.StatusOK {
				continue
			}

			var resp policy.AccessPolicy
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.ID, ap1.ID)
			assert.Equal(t, resp.Components[0].Template.ID, req.AccessPolicy.Components[0].Template.ID)

			got, err := s.GetLatestAccessPolicy(ctx, ap1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, got.Version, i+1)
		}
		testhelpers.CheckAuditLog(ctx, t, tf.tenantDB, internal.AuditLogEventTypeUpdateAccessPolicy, tf.userID.String(), time.Time{}, 2)
	})

	t.Run("TestDelete", func(t *testing.T) {
		apt1 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template5",
			Function:                 "function ap5() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt1), assert.Must())

		ap := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Policy5",
			ComponentIDs:             []uuid.UUID{apt1.ID},
			ComponentParameters:      []string{``},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}
		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap), assert.Must())

		// rev the version
		ap1 := *ap
		ap1.ComponentParameters = []string{`{"a":1}`}
		ap1.Version++
		assert.IsNil(t, s.SaveAccessPolicy(ctx, &ap1), assert.Must())

		// delete the policy by ID & version
		rr := tf.runTokenizerRequest(http.MethodDelete, paths.DeleteAccessPolicy(ap1.ID, ap1.Version), nil)
		assert.Equal(t, rr.Code, http.StatusNoContent, assert.Must())

		got, err := s.GetAccessPolicyByVersion(ctx, ap1.ID, 0)
		assert.NoErr(t, err)
		assert.Equal(t, *got, *ap)

		got, err = s.GetAccessPolicyByVersion(ctx, ap1.ID, 1)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)

		testhelpers.CheckAuditLog(ctx, t, tf.tenantDB, internal.AuditLogEventTypeDeleteAccessPolicy, tf.userID.String(), time.Time{}, 1)

		// re-add v1 of the policy and then try deleting all versions
		assert.IsNil(t, s.SaveAccessPolicy(ctx, &ap1), assert.Must())
		rr = tf.runTokenizerRequest(http.MethodDelete, paths.DeleteAllAccessPolicyVersions(ap1.ID), nil)
		assert.Equal(t, rr.Code, http.StatusNoContent, assert.Must())

		got, err = s.GetAccessPolicyByVersion(ctx, ap1.ID, 0)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)

		got, err = s.GetAccessPolicyByVersion(ctx, ap1.ID, 1)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)
	})

	t.Run("TestRun", func(t *testing.T) {
		apt1 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template6",
			Function:                 "function policy(context, params) { return context.client.purpose === params.expected_purpose; }",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt1), assert.Must())

		req := tokenizer.TestAccessPolicyRequest{
			AccessPolicy: policy.AccessPolicy{
				Name: "TestRunPolicy",
				Components: []policy.AccessPolicyComponent{{
					Template:           &userstore.ResourceID{ID: apt1.ID},
					TemplateParameters: `{"expected_purpose":"foo"}`,
				}},
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
		}
		rr := tf.runTokenizerRequest(http.MethodPost, paths.TestAccessPolicy, req)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
		var res tokenizer.TestAccessPolicyResponse
		assert.IsNil(t, json.Unmarshal(rr.Body.Bytes(), &res), assert.Must())
		assert.False(t, res.Allowed) // fails because no purpose is set in client context

		// set purpose in client context and retry
		req.Context = policy.AccessPolicyContext{Client: policy.ClientContext{"purpose": "foo"}}
		rr = tf.runTokenizerRequest(http.MethodPost, paths.TestAccessPolicy, req)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
		assert.IsNil(t, json.Unmarshal(rr.Body.Bytes(), &res), assert.Must())
		assert.True(t, res.Allowed)

		// check that empty policy is not allowed
		req = tokenizer.TestAccessPolicyRequest{
			AccessPolicy: policy.AccessPolicy{
				Name:       `TestRunPolicy`,
				Components: nil,
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
		}
		rr = tf.runTokenizerRequest(http.MethodPost, paths.TestAccessPolicy, req)
		assert.Equal(t, rr.Code, http.StatusBadRequest, assert.Must())
	})
}
