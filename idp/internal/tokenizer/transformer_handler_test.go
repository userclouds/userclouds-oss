package tokenizer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	tokenizerInternal "userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/testhelpers"
)

func TestTransformerHandler(t *testing.T) {
	ctx := context.Background()

	tf := newTestFixture(t)
	s := tf.storage

	// sort function
	sortAPs := func(expected []policy.Transformer) func(i, j int) bool {
		return func(i, j int) bool { return expected[i].ID.String() < expected[j].ID.String() }
	}

	t.Run("TestList", func(t *testing.T) {
		t1 := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Policy1",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "{}",
		}
		assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(t1)), assert.Must())
		t2 := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Policy2",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function foo() {}",
			Parameters:     "[1, 2, 3]",
		}
		assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(t2)), assert.Must())

		expected := []policy.Transformer{}
		for _, t := range defaults.GetDefaultTransformers() {
			expected = append(expected, t.ToClientModel())
		}
		expected = append(expected, t1, t2)
		sort.Slice(expected, sortAPs(expected))
		rr := tf.runTokenizerRequest(http.MethodGet, paths.BaseTransformerPath, nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		var resp idp.ListTransformersResponse
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, len(resp.Data), len(expected))

		sort.Slice(resp.Data, sortAPs(resp.Data))
		for i := range resp.Data {
			assert.True(
				t,
				storage.NewTransformerFromClient(
					resp.Data[i],
				).EqualsIgnoringNilID(
					storage.NewTransformerFromClient(expected[i]),
				),
			)
		}
	})

	t.Run("TestCreate", func(t *testing.T) {
		type testcase struct {
			ID       uuid.UUID
			Name     string
			Function string
			Code     int
		}

		cases := []testcase{
			{
				Name:     "Policy_1",
				Function: "function transform(x, y) { return 'token'; }",
				Code:     http.StatusCreated,
			},
			{
				Name:     "Policy_2",
				Function: "function invalid javascript",
				Code:     http.StatusBadRequest,
			},
			{
				// duplicate of case 0 by name, should fail
				Name:     "Policy_1",
				Function: "function transform(x, y) { return 'false'; }",
				Code:     http.StatusConflict,
			},
			{
				// should fail due to blank name
				Name:     "",
				Function: "function transform(x, y) { return 'false'; }",
				Code:     http.StatusBadRequest,
			},
			{
				Name:     "Policy_2",
				Function: "function transform(x, y) { return true; }",
				Code:     http.StatusCreated,
			},
		}

		for i, c := range cases {
			req := tokenizer.CreateTransformerRequest{
				Transformer: policy.Transformer{
					Name:           c.Name,
					InputDataType:  datatype.String,
					OutputDataType: datatype.String,
					TransformType:  policy.TransformTypeTransform,
					Function:       c.Function,
				},
			}
			rr := tf.runTokenizerRequest(http.MethodPost, paths.BaseTransformerPath, req)
			assert.Equal(t, rr.Code, c.Code, assert.Errorf("test case %d: expected %d, got %d", i, c.Code, rr.Code))

			if rr.Code != http.StatusCreated {
				continue
			}

			// only check the body if the create worked
			var resp policy.Transformer
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp), assert.Errorf("failed to decode response on case %d", i))
			assert.Equal(t, resp.Function, c.Function, assert.Errorf("response function body doesn't match on case %d", i))
		}
		testhelpers.CheckAuditLog(ctx, t, tf.tenantDB, internal.AuditLogEventTypeCreateTransformer, tf.userID.String(), time.Time{}, 2)
	})

	t.Run("TestGet", func(t *testing.T) {
		ap1 := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Policy1TestGet",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function ap2() {}",
			Parameters:     "[1]",
		}
		assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(ap1)), assert.Must())

		rr := tf.runTokenizerRequest(http.MethodGet, fmt.Sprintf("%s/%v", paths.BaseTransformerPath, ap1.ID), nil)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Must())

		var resp policy.Transformer
		assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, resp.ID, ap1.ID)
		assert.Equal(t, resp.Function, ap1.Function)
		assert.Equal(t, resp.Parameters, ap1.Parameters)
	})

	t.Run("TestDelete", func(t *testing.T) {
		ap1 := policy.Transformer{
			ID:             uuid.Must(uuid.NewV4()),
			Name:           "Policy1TestDelete",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function:       "function ap3() {}",
		}
		assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(ap1)), assert.Must())

		// delete the policy
		rr := tf.runTokenizerRequest(http.MethodDelete, fmt.Sprintf("%s/%v", paths.BaseTransformerPath, ap1.ID), nil)
		assert.Equal(t, rr.Code, http.StatusNoContent, assert.Must())

		got, err := s.GetLatestTransformer(ctx, ap1.ID)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.IsNil(t, got)

		testhelpers.CheckAuditLog(ctx, t, tf.tenantDB, internal.AuditLogEventTypeDeleteTransformer, tf.userID.String(), time.Time{}, 1)
	})
	t.Run("TestTestTransformerWithJSErrors", func(t *testing.T) {
		type testcase struct {
			function     string
			errorMessage string
		}
		cases := []testcase{
			{
				function:     "function transform(x, y}",
				errorMessage: "Transformer javascript validation error: SyntaxError: Unexpected token '}'",
			},
			{
				function:     "function transform(x, y) { return puddy; }",
				errorMessage: "error executing transformer: ReferenceError: puddy is not defined",
			},
			{
				function:     "function transform(data, y) { return data.splice(22); }",
				errorMessage: "error executing transformer: TypeError: data.splice is not a function",
			},
		}

		for i, c := range cases {
			req := tokenizer.TestTransformerRequest{
				Transformer: policy.Transformer{
					Name:           "JerrySeinfeld",
					InputDataType:  datatype.String,
					OutputDataType: datatype.String,
					TransformType:  policy.TransformTypeTransform,
					Function:       c.function,
				},
			}
			rr := tf.runTokenizerRequest(http.MethodPost, paths.TestTransformer, req)
			assert.Equal(t, rr.Code, http.StatusBadRequest, assert.Errorf("Unexpected HTTP response code, test case: %d", i))
			var resp jsonapi.JSONErrorMessage
			assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &resp), assert.Errorf("Unexpected response body, test case: %d", i))
			assert.Equal(t, resp.Error, c.errorMessage, assert.Errorf("Unexpected error message, test case: %d", i))
		}
	})
}

func TestTransformerWithNetworkRequest(t *testing.T) {
	ctx := context.Background()

	tf := newTestFixture(t)
	s := tf.storage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "foo")
	}))
	defer srv.Close()

	transformer := &storage.Transformer{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     "TestNetworkRequestTransformer",
		InputDataTypeID:          datatype.String.ID,
		OutputDataTypeID:         datatype.String.ID,
		TransformType:            storage.InternalTransformTypeFromClient(policy.TransformTypeTransform),
		Function:                 fmt.Sprintf("function transform(x, y) { return networkRequest({method: 'GET', url: '%s'}).trim(); }", srv.URL),
	}

	token, _, err := tokenizerInternal.ExecuteTransformer(ctx, s, tf.authzClient, uuid.Nil, transformer, "test data", nil)
	assert.NoErr(t, err)
	assert.Equal(t, fmt.Sprintf("*%s*", token), "*foo*")
}

func TestExecuteTransformers(t *testing.T) {
	ctx := context.Background()

	testUserID := uuid.Must(uuid.NewV4())

	tf := newTestFixture(t)
	s := tf.storage

	pt1 := policy.Transformer{
		ID:             uuid.Must(uuid.NewV4()),
		Name:           "Transformer1",
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTransform,
		Function:       "function transform(x, y) { return x; } // Transformer1",
	}
	assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(pt1)), assert.Must())
	t1, err := s.GetLatestTransformer(ctx, pt1.ID)
	assert.NoErr(t, err)

	pt2 := policy.Transformer{
		ID:             uuid.Must(uuid.NewV4()),
		Name:           "Transformer2",
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTokenizeByValue,
		Function:       `const a = "this is a global constant"; function transform(x, y) { return x; } // Transformer2`,
	}
	assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(pt2)), assert.Must())
	t2, err := s.GetLatestTransformer(ctx, pt2.ID)
	assert.NoErr(t, err)

	pt3 := policy.Transformer{
		ID:             uuid.Must(uuid.NewV4()),
		Name:           "Transformer3",
		InputDataType:  datatype.String,
		OutputDataType: datatype.String,
		TransformType:  policy.TransformTypeTokenizeByReference,
		Function:       "function transform(x, y) { return x; } // Transformer3",
	}
	assert.IsNil(t, tokenizerInternal.SaveTransformerWithAuthz(ctx, s, tf.authzClient, storage.NewTransformerFromClient(pt3)), assert.Must())
	t3, err := s.GetLatestTransformer(ctx, pt3.ID)
	assert.NoErr(t, err)

	t4, err := s.GetLatestTransformer(ctx, policy.TransformerPassthrough.ID)
	assert.NoErr(t, err)

	params := []tokenizerInternal.ExecuteTransformerParameters{
		{
			Transformer: t1,
			Data:        "1",
		},
		{
			Transformer:         t2,
			TokenAccessPolicyID: policy.AccessPolicyAllowAll.ID,
			Data:                "2",
		},
		{
			Transformer:         t2,
			TokenAccessPolicyID: policy.AccessPolicyAllowAll.ID,
			Data:                "3",
		},
		{
			Transformer:         t3,
			TokenAccessPolicyID: policy.AccessPolicyAllowAll.ID,
			Data:                "4",
			DataProvenance: &policy.UserstoreDataProvenance{
				UserID:   testUserID,
				ColumnID: column.NameColumnID,
			},
		},
		{
			Transformer: t4,
			Data:        "5",
		},
		{
			Transformer:         t2,
			TokenAccessPolicyID: policy.AccessPolicyAllowAll.ID,
			Data:                "6",
		},
		{
			Transformer: t1,
			Data:        "7",
		},
		{
			Transformer:         t3,
			TokenAccessPolicyID: policy.AccessPolicyAllowAll.ID,
			Data:                "8",
			DataProvenance: &policy.UserstoreDataProvenance{
				UserID:   testUserID,
				ColumnID: column.NicknameColumnID,
			},
		},
		{
			Transformer: t4,
			Data:        "9",
		},
	}

	te := tokenizerInternal.NewTransformerExecutor(s, tf.authzClient)
	defer te.CleanupExecution()
	results, _, err := te.Execute(ctx, params...)
	assert.NoErr(t, err)
	assert.Equal(t, len(results), len(params))
	for i := range params {
		assert.Equal(t, params[i].Data, results[i])
	}
}
