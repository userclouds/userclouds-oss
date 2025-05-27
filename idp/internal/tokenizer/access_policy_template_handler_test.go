package tokenizer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	tokenizerInternal "userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/test/testlogtransport"
)

func TestAccessPolicyTemplateHandler(t *testing.T) {
	tf := newTestFixture(t)
	ctx := context.Background()
	s := tf.storage

	// sort function
	sortAPTs := func(expected []policy.AccessPolicyTemplate) func(i, j int) bool {
		return func(i, j int) bool { return expected[i].ID.String() < expected[j].ID.String() }
	}

	t.Run("TestTemplateList", func(t *testing.T) {
		expected := []policy.AccessPolicyTemplate{}
		for _, apt := range defaults.GetDefaultAccessPolicyTemplates() {
			expected = append(expected, apt.ToClient())
		}
		total_versioned := len(expected)

		for i := 1; i <= pagination.DefaultLimit+10; i++ {
			apt := &storage.AccessPolicyTemplate{
				SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
				Name:                     fmt.Sprintf("TemplateList%d", i),
				Function:                 fmt.Sprintf("{} // TemplateList%d", i),
			}
			assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt), assert.Must())
			total_versioned++
			if i%2 == 0 {
				apt.Version = 1
				apt.Function = fmt.Sprintf("{} // TemplateList%d-v1", i)
				assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt), assert.Must())
				total_versioned++
			}
			expected = append(expected, apt.ToClient())
		}
		sort.Slice(expected, sortAPTs(expected))

		rr := tf.runTokenizerRequest(http.MethodGet, paths.BaseAccessPolicyTemplatePath, nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		got := []policy.AccessPolicyTemplate{}

		var resp idp.ListAccessPolicyTemplatesResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), pagination.DefaultLimit)
		assert.True(t, resp.ResponseFields.HasNext)
		got = append(got, resp.Data...)
		rr = tf.runTokenizerRequest(http.MethodGet, fmt.Sprintf("%s?starting_after=%s", paths.BaseAccessPolicyTemplatePath, resp.ResponseFields.Next), nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), len(expected)-pagination.DefaultLimit)
		assert.False(t, resp.ResponseFields.HasNext)
		got = append(got, resp.Data...)

		sort.Slice(got, sortAPTs(got))
		for i := range got {
			assert.True(t, got[i].ID == expected[i].ID && got[i].Name == expected[i].Name && got[i].Version == expected[i].Version)
		}

		rr = tf.runTokenizerRequest(http.MethodGet, fmt.Sprintf("%s?versioned=true&limit=200", paths.BaseAccessPolicyTemplatePath), nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), total_versioned)
	})

	t.Run("TestTemplateGet", func(t *testing.T) {
		apt3 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template3",
			Function:                 "function bar() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt3), assert.Must())

		apt4 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template4",
			Function:                 "function baz() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt4), assert.Must())

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
				Name:       "Template3",
				TemplateID: apt3.ID,
				Version:    0,
				URL:        paths.GetAccessPolicyTemplate(apt3.ID),
			},
			{
				Name:       "Template4",
				TemplateID: apt4.ID,
				Version:    0,
				URL:        paths.GetAccessPolicyTemplate(apt4.ID),
			},
		}

		for _, c := range cases {
			rr := tf.runTokenizerRequest(http.MethodGet, c.URL, nil)
			assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
			var resp policy.AccessPolicyTemplate
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Name, c.Name)
			assert.Equal(t, resp.ID, c.TemplateID)
			assert.Equal(t, resp.Version, c.Version)
		}

		apt3.Function = "function bar2() {}"
		apt3.Version++
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt3), assert.Must())

		cases = []testcase{
			{
				// Get without version when a second version exists (should return latest)
				Name:       "Template3",
				TemplateID: apt3.ID,
				Version:    1,
				URL:        paths.GetAccessPolicyTemplate(apt3.ID),
			},
			{
				// Get with version
				Name:       "Template3",
				TemplateID: apt3.ID,
				Version:    0,
				URL:        paths.GetAccessPolicyTemplateByVersion(apt3.ID, 0),
			},
			{
				Name:       "Template3",
				TemplateID: apt3.ID,
				Version:    1,
				URL:        paths.GetAccessPolicyTemplateByVersion(apt3.ID, 1),
			},
		}

		for _, c := range cases {
			rr := tf.runTokenizerRequest(http.MethodGet, c.URL, nil)
			assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

			var resp policy.AccessPolicyTemplate
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Name, c.Name)
			assert.Equal(t, resp.ID, c.TemplateID)
			assert.Equal(t, resp.Version, c.Version)
		}
	})

	t.Run("TestTemplateCreate", func(t *testing.T) {
		tt := testlogtransport.InitLoggerAndTransportsForTests(t)
		type testcase struct {
			Name     string
			Function string
			Code     int
		}

		cases := []testcase{
			{
				Name:     "Template5",
				Function: "function policy(x, y) { return 'token'; }",
				Code:     http.StatusCreated,
			},
			{
				Name:     "Template6",
				Function: "function invalid javascript",
				Code:     http.StatusBadRequest,
			},
			{
				// duplicate of case 0 by policy, should fail
				Name:     "Template5",
				Function: "function policy(x, y) { return 'token'; }",
				Code:     http.StatusConflict,
			},
			{
				// duplicate of case 0 by name, should fail
				Name:     "Template5",
				Function: "function policy(x, y) { return 'false'; }",
				Code:     http.StatusConflict,
			},
			{
				// fails due to blank name
				Name:     "",
				Function: "function policy(x, y) { return 'false'; }",
				Code:     http.StatusBadRequest,
			},
			{
				Name:     "Template7",
				Function: "function policy(x, y) { return true; }",
				Code:     http.StatusCreated,
			},
		}

		for i, c := range cases {
			reqTemplate := tokenizer.CreateAccessPolicyTemplateRequest{
				AccessPolicyTemplate: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     c.Name,
					Function:                 c.Function,
				},
			}
			rr := tf.runTokenizerRequest(http.MethodPost, paths.BaseAccessPolicyTemplatePath, reqTemplate)
			tt.AssertMessagesByLogLevel(uclog.LogLevelError, 0) // We should not log errors for this test since all responses are 20x or 40x
			if c.Code >= 400 && c.Code < 500 {
				// We should log a warning for this test since all responses are 40x
				// Two warnings, one from the logic and one form the uclog middleware logging the HTTP 40x response as a warning
				tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 2)
			}
			tt.ClearMessages()

			assert.Equal(t, rr.Code, c.Code, assert.Errorf("test case %d: expected %d, got %d", i, c.Code, rr.Code))

			if rr.Code != http.StatusCreated {
				continue
			}

			// only check the body if the create worked
			var resp policy.AccessPolicyTemplate
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp), assert.Errorf("failed to decode response on case %d", i))
			assert.Equal(t, resp.Function, c.Function, assert.Errorf("response function body doesn't match on case %d", i))

		}
	})

	t.Run("TestTemplateTest", func(t *testing.T) {
		apt := policy.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TemplateForTestingTesting",
			Function:                 "function policy(context, params) { return context.server.claims.sub === 'dave'; }",
		}

		type testcase struct {
			apt     policy.AccessPolicyTemplate
			ctx     policy.AccessPolicyContext
			params  string
			allowed bool
		}

		apc := policy.AccessPolicyContext{
			Server: policy.ServerContext{
				IPAddress: "127.0.0.1",
				Claims: map[string]any{
					"sub": "dave",
				},
				Action: "resolve",
			},
			Client: policy.ClientContext{},
			User:   userstore.Record{},
		}

		cases := []testcase{
			{
				apt:     apt,
				ctx:     apc,
				params:  "{}",
				allowed: true,
			},
			{
				apt: apt,
				ctx: policy.AccessPolicyContext{
					Server: policy.ServerContext{
						IPAddress: "127.0.0.1",
						Claims: map[string]any{
							"sub": "farah",
						},
						Action: "resolve",
					},
					Client: policy.ClientContext{},
					User: userstore.Record{
						"id": tf.userID.String(),
					},
				},
				params:  "{}",
				allowed: false,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     apt.Name,
					Function:                 "function policy(context, params) { return params.username === 'dave'; }",
				},
				ctx:     apc,
				params:  "{ username: \"dave\" }",
				allowed: true,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     apt.Name,
					Function:                 "function policy(context, params) { return params.username === 'dave'; }",
				},
				ctx:     apc,
				params:  "{ username: \"lourdes\" }",
				allowed: false,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     apt.Name,
					Function:                 "function policy(context, params) { return context.server.claims.sub === params.username; }",
				},
				ctx:     apc,
				params:  "{ username: \"dave\" }",
				allowed: true,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     apt.Name,
					Function:                 "function policy(context, params) { return context.server.claims.sub === params.username; }",
				},
				ctx:     apc,
				params:  "{ username: \"lily\" }",
				allowed: false,
			},
		}
		for _, c := range cases {
			req := tokenizer.TestAccessPolicyTemplateRequest{
				AccessPolicyTemplate: c.apt,
				Context:              c.ctx,
				Params:               c.params,
			}
			rr := tf.runTokenizerRequest(http.MethodPut, paths.TestAccessPolicyTemplate, req)
			assert.Equal(t, rr.Code, http.StatusOK)
			var resp tokenizer.TestAccessPolicyResponse
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Allowed, c.allowed)
		}
	})

	t.Run("TestTemplateTestErrors", func(t *testing.T) {

		type testcase struct {
			apt          policy.AccessPolicyTemplate
			ctx          policy.AccessPolicyContext
			params       string
			code         int
			errorMessage string
		}

		apc := policy.AccessPolicyContext{
			Server: policy.ServerContext{
				IPAddress: "127.0.0.1",
				Claims: map[string]any{
					"sub": "dave",
				},
				Action: "resolve",
			},
			Client: policy.ClientContext{},
			User:   userstore.Record{},
		}

		cases := []testcase{
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
					Name:                     "Jerry",
					Function:                 "function policy(context, params) { return undef + 'foo'; }", // runtime error
				},
				ctx:          apc,
				params:       "{ username: \"lily\" }",
				code:         http.StatusBadRequest,
				errorMessage: "error executing access policy template: ReferenceError: undef is not defined",
			},
		}
		for _, c := range cases {
			req := tokenizer.TestAccessPolicyTemplateRequest{
				AccessPolicyTemplate: c.apt,
				Context:              c.ctx,
				Params:               c.params,
			}
			rr := tf.runTokenizerRequest(http.MethodPut, paths.TestAccessPolicyTemplate, req)
			assert.Equal(t, rr.Code, c.code)
			var resp jsonapi.JSONErrorMessage
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.Error, c.errorMessage)
		}
	})

	t.Run("TestTemplateUpdate", func(t *testing.T) {
		apt := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "TemplateTestUpdate",
			Function:                 "function policy() {} // original",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt), assert.Must())

		type testcase struct {
			apt     policy.AccessPolicyTemplate
			code    int
			version int
		}

		cases := []testcase{
			{
				apt: policy.AccessPolicyTemplate{
					Name:     apt.Name,
					Function: "function policy() {} // v2",
				},
				code:    http.StatusOK,
				version: 0,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(apt.ID), // including ID
					Name:                     apt.Name,
					Function:                 "function policy() {} // v2",
				},
				code:    http.StatusOK,
				version: 1,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(apt.ID), // including ID
					Name:                     apt.Name,
					Function:                 "function policy() {} // v2",
				},
				code:    http.StatusBadRequest,
				version: 1,
			},
			{
				apt: policy.AccessPolicyTemplate{
					SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(), // mismatched ID
					Name:                     apt.Name,
					Function:                 "function policy() {} // v3",
				},
				code:    http.StatusBadRequest,
				version: 2,
			},
		}

		for i, c := range cases {
			req := tokenizer.TestAccessPolicyTemplateRequest{AccessPolicyTemplate: c.apt}
			req.AccessPolicyTemplate.Version = c.version
			rr := tf.runTokenizerRequest(http.MethodPut, fmt.Sprintf("%s/%s", paths.BaseAccessPolicyTemplatePath, apt.ID), req)
			assert.Equal(t, rr.Code, c.code)

			// don't expect a response body on failure
			if c.code != http.StatusOK {
				continue
			}

			var resp policy.AccessPolicyTemplate
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, resp.ID, apt.ID)

			got, err := s.GetLatestAccessPolicyTemplate(ctx, apt.ID)
			assert.NoErr(t, err)
			assert.Equal(t, got.Version, i+1)
		}
	})

	t.Run("TestTemplateDelete", func(t *testing.T) {
		apt8 := &storage.AccessPolicyTemplate{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Template8",
			Function:                 "function apt8() {}",
		}
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt8), assert.Must())

		apt8.Version++
		apt8.Function = "function apt8() {} // v2"
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt8), assert.Must())

		apt8.Version++
		apt8.Function = "function apt8() {} // v3"
		assert.IsNil(t, s.SaveAccessPolicyTemplate(ctx, apt8), assert.Must())

		ap1 := &storage.AccessPolicy{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
			Name:                     "Policy1",
			ComponentIDs:             []uuid.UUID{apt8.ID},
			ComponentParameters:      []string{""},
			ComponentTypes:           []int32{int32(storage.AccessPolicyComponentTypeTemplate)},
			PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
		}

		assert.IsNil(t, tokenizerInternal.SaveAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap1), assert.Must())

		// delete the policy by ID & version
		rr := tf.runTokenizerRequest(http.MethodDelete, paths.DeleteAccessPolicyTemplate(apt8.ID, 1), nil)
		assert.Equal(t, rr.Code, http.StatusNoContent, assert.Must())

		got, err := s.GetAccessPolicyTemplateByVersion(ctx, apt8.ID, 0)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt8.ID)
		assert.Equal(t, got.Version, 0)

		got, err = s.GetAccessPolicyTemplateByVersion(ctx, apt8.ID, 1)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)

		got, err = s.GetAccessPolicyTemplateByVersion(ctx, apt8.ID, 2)
		assert.NoErr(t, err)
		assert.Equal(t, got.ID, apt8.ID)
		assert.Equal(t, got.Version, 2)

		// try to delete both remaining versions of this policy
		rr = tf.runTokenizerRequest(http.MethodDelete, paths.DeleteAllAccessPolicyTemplateVersions(apt8.ID), nil)
		assert.Equal(t, rr.Code, http.StatusConflict, assert.Must()) // it should fail

		// delete the access policy that references it and try again
		assert.IsNil(t, tokenizerInternal.DeleteAccessPolicyWithAuthz(ctx, s, tf.authzClient, ap1))

		rr = tf.runTokenizerRequest(http.MethodDelete, paths.DeleteAllAccessPolicyTemplateVersions(apt8.ID), nil)
		assert.Equal(t, rr.Code, http.StatusNoContent, assert.Must())

		got, err = s.GetAccessPolicyTemplateByVersion(ctx, apt8.ID, 0)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)

		got, err = s.GetAccessPolicyTemplateByVersion(ctx, apt8.ID, 2)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)
	})
}
