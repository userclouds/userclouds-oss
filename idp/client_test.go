package idp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/authz"
	"userclouds.com/idp"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/events"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/uclog"
	tenantplexstorage "userclouds.com/internal/tenantplex/storage"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/test/testlogtransport"
)

var badID = uuid.Must(uuid.NewV4())

func getUpdatableMutator(m1 *userstore.Mutator) *userstore.Mutator {
	m2 := *m1
	for i := range m2.Columns {
		m2.Columns[i].Validator = userstore.ResourceID{}
	}
	return &m2
}

func uniqueName(name string) string {
	return fmt.Sprintf("%s_%v", name, uuid.Must(uuid.NewV4()))
}

type externalTokenSource struct {
	t      *testing.T
	issuer string
}

// GetToken implements oidc.TokenSource
func (t externalTokenSource) GetToken() (string, error) {
	jwt := uctest.CreateJWT(t.t, oidc.UCTokenClaims{StandardClaims: oidc.StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "testuser"}}}, t.issuer)
	return jwt, nil
}

func TestSharedTestFixtureTests(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	t.Run("test_create_column", func(t *testing.T) {
		// create a column with an invalid name - should fail
		_, err := tf.IDPClient.CreateColumn(
			tf.Ctx,
			userstore.Column{
				Table:        "users",
				Name:         "foo.bar",
				DataType:     datatype.String,
				IsArray:      false,
				DefaultValue: "",
				IndexType:    userstore.ColumnIndexTypeIndexed,
			},
		)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// create a column
		colName := uniqueName("column")
		createdColumn := tf.CreateValidColumn(colName, datatype.String, false, "default", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		// create a column with the same name, same contents, different id - should fail
		col, err := tf.IDPClient.CreateColumn(tf.Ctx, userstore.Column{
			Name:         "columnWithoutTable",
			DataType:     datatype.String,
			IsArray:      false,
			DefaultValue: "default",
			IndexType:    userstore.ColumnIndexTypeIndexed,
		})
		assert.NoErr(tf.T, err)
		assert.NotNil(tf.T, col.ID)
		assert.Equal(tf.T, col.Table, "users") // automatically added
		assert.Equal(tf.T, col.Name, "columnWithoutTable")
		assert.Equal(tf.T, col.DataType, datatype.String)
		assert.Equal(tf.T, col.IsArray, false)
		assert.Equal(tf.T, col.DefaultValue, "default")
		assert.Equal(tf.T, col.IndexType, userstore.ColumnIndexTypeIndexed)

		// create a column with the same name, same contents, different id - should fail
		_, err = tf.IDPClient.CreateColumn(tf.Ctx, userstore.Column{
			Table:        "users",
			Name:         colName,
			DataType:     datatype.String,
			IsArray:      false,
			DefaultValue: "default",
			IndexType:    userstore.ColumnIndexTypeIndexed,
		})
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr := jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.True(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdColumn.ID)

		// create a column with the same name, different contents, different id - should fail
		_, err = tf.CreateColumn(colName, datatype.String, false, "new_default", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.False(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdColumn.ID)

		// create a column with the same name, same contents, same id - should fail
		badColumn := *createdColumn
		_, err = tf.IDPClient.CreateColumn(tf.Ctx, badColumn)
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.True(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdColumn.ID)

		// create a column with the same name, different contents, same id - should fail
		badColumn = *createdColumn
		badColumn.DefaultValue = "new_default"
		_, err = tf.IDPClient.CreateColumn(tf.Ctx, badColumn)
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.False(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdColumn.ID)

		// create a column with empty name - should fail
		_, err = tf.CreateColumn("", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NotNil(t, err)

		// create a column with invalid column type - should fail
		badDataType := userstore.ResourceID{Name: "invalid"}
		_, err = tf.CreateColumn(uniqueName("column"), badDataType, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NotNil(t, err)

		// create int column with good default value
		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, false, "42", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, err)

		// create int column with bad default value - should fail
		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, false, "bad_default", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// create a column that holds an array of integers
		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, err)

		// verify that array columns cannot pass in unique index type or default value
		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeUnique)
		assert.HTTPError(t, err, http.StatusBadRequest)

		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, true, "default", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// create a column that holds an array of addresses
		_, err = tf.CreateColumn(uniqueName("column"), datatype.Integer, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, err)
	})

	t.Run("test_get_user", func(t *testing.T) {
		_, err := tf.IDPClient.GetUser(tf.Ctx, uuid.Must(uuid.NewV4()))
		assert.NotNil(t, err)

		_, err = tf.IDPClient.GetUser(tf.Ctx, uuid.Nil)
		assert.NotNil(t, err)
	})

	t.Run("test_get_column", func(t *testing.T) {
		// try GetOne for non-existent column - should fail
		_, err := tf.IDPClient.GetColumn(tf.Ctx, badID)
		assert.NotNil(t, err)

		// create valid column and ensure GetOne and GetAll work
		createdCol := tf.CreateValidColumn(uniqueName("column"), datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		foundCol, err := tf.IDPClient.GetColumn(tf.Ctx, createdCol.ID)
		assert.NoErr(t, err)
		assert.Equal(t, createdCol, foundCol)

		foundCols, err := tf.IDPClient.ListColumns(tf.Ctx)
		assert.NoErr(t, err)

		columnFound := false
		for _, foundCol := range foundCols.Data {
			if foundCol.ID == createdCol.ID {
				assert.Equal(t, createdCol, &foundCol)
				columnFound = true
				break
			}
		}
		assert.True(t, columnFound)
	})

	t.Run("test_delete_column", func(t *testing.T) {
		createdCol := tf.CreateValidColumn(uniqueName("column"), datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		// delete column successfully, make sure not found via GetOne or GetAll
		err := tf.IDPClient.DeleteColumn(tf.Ctx, createdCol.ID)
		assert.NoErr(t, err)

		_, err = tf.IDPClient.GetColumn(tf.Ctx, createdCol.ID)
		assert.HTTPError(t, err, http.StatusNotFound)

		foundCols, err := tf.IDPClient.ListColumns(tf.Ctx)
		assert.NoErr(t, err)

		for _, foundCol := range foundCols.Data {
			assert.NotEqual(t, foundCol.ID, createdCol.ID)
		}

		// delete a non-existent column - should fail
		err = tf.IDPClient.DeleteColumn(tf.Ctx, badID)
		assert.HTTPError(t, err, http.StatusNotFound)

		// create and delete an array column
		createdCol = tf.CreateValidColumn(uniqueName("column"), datatype.Boolean, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, tf.IDPClient.DeleteColumn(tf.Ctx, createdCol.ID))
	})

	t.Run("test_execute_accessor", func(t *testing.T) {
		transformerBody := `function id(len) {
		var s = "0123456789";
		return Array(len).join().split(',').map(function() {
			return s.charAt(Math.floor(Math.random() * s.length));
		}).join('');
	}

	function validate(str) {
		return (str.length === 10);
	}

	function transform(data, params) {
	  // Strip non numeric characters if present
	  orig_data = data;
	  data = data.replace(/\D/g, '');

	  if (data.length === 11 ) {
		data = data.substr(1, 11);
	  }

	  if (!validate(data)) {
			throw new Error('Invalid US Phone Number Provided');
	  }

	  return '1' + id(10);
	}`
		colName := uniqueName("column")
		col := tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		alias := uniqueName("alias")
		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"external_alias": alias}, idp.DataRegion("aws-us-east-1"), idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		assert.NotNil(t, uid)

		got, err := tf.IDPClient.GetUser(tf.Ctx, uid)
		assert.NoErr(t, err)
		assert.Equal(t, got.Profile["external_alias"], alias)

		transformerName := uniqueName("transformer")
		transformer, err := tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name:               transformerName,
				Function:           fmt.Sprintf("%s // %s", transformerBody, transformerName),
				Parameters:         "",
				InputDataType:      datatype.String,
				OutputDataType:     datatype.String,
				ReuseExistingToken: true,
				TransformType:      policy.TransformTypeTokenizeByValue,
			})
		assert.NoErr(t, err)

		ac, err := tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: colName},
					Transformer:       userstore.ResourceID{ID: transformer.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		alias = uniqueName("alias")
		pn := "+1 555 555 1212"
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{colName: pn, "external_alias": alias}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		// Test tokenize by value with reuse
		token, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields := map[string]string{}
		err = json.Unmarshal([]byte(token.Data[0]), &fields)
		assert.NoErr(t, err)

		resp, err := tf.IDPClient.ResolveToken(tf.Ctx, fields[colName], policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resp, pn)

		token2, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.Equal(t, token2, token)

		transformerName = uniqueName("transformer")
		transformer, err = tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name:               transformerName,
				Function:           fmt.Sprintf("%s // %s", transformerBody, transformerName),
				Parameters:         "",
				InputDataType:      datatype.String,
				OutputDataType:     datatype.String,
				ReuseExistingToken: true,
				TransformType:      policy.TransformTypeTokenizeByReference,
			})
		assert.NoErr(t, err)

		ac, err = tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: colName},
					Transformer:       userstore.ResourceID{ID: transformer.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		// Test tokenize by reference with reuse
		token, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields = map[string]string{}
		err = json.Unmarshal([]byte(token.Data[0]), &fields)
		assert.NoErr(t, err)

		resp, err = tf.IDPClient.ResolveToken(tf.Ctx, fields[colName], policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resp, pn)

		token2, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.Equal(t, token2, token)

		transformerName = uniqueName("transformer")
		transformer, err = tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name:               transformerName,
				Function:           fmt.Sprintf("%s // %s", transformerBody, transformerName),
				Parameters:         "",
				InputDataType:      datatype.String,
				OutputDataType:     datatype.String,
				ReuseExistingToken: false,
				TransformType:      policy.TransformTypeTokenizeByReference,
			})
		assert.NoErr(t, err)

		ac, err = tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: colName},
					Transformer:       userstore.ResourceID{ID: transformer.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		// Test tokenize by reference without reuse
		token, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields = map[string]string{}
		err = json.Unmarshal([]byte(token.Data[0]), &fields)
		assert.NoErr(t, err)

		resp, err = tf.IDPClient.ResolveToken(tf.Ctx, fields[colName], policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resp, pn)

		token2, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.NotEqual(t, token2, token)

		transformerName = uniqueName("transformer")
		transformer, err = tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name:               transformerName,
				Function:           fmt.Sprintf("%s // %s", transformerBody, transformerName),
				Parameters:         "",
				InputDataType:      datatype.String,
				OutputDataType:     datatype.String,
				ReuseExistingToken: false,
				TransformType:      policy.TransformTypeTokenizeByValue,
			})
		assert.NoErr(t, err)

		ac, err = tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: colName},
					Transformer:       userstore.ResourceID{ID: transformer.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		// Test tokenize by value without reuse
		token, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields = map[string]string{}
		err = json.Unmarshal([]byte(token.Data[0]), &fields)
		assert.NoErr(t, err)

		resp, err = tf.IDPClient.ResolveToken(tf.Ctx, fields[colName], policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resp, pn)

		token2, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.NotEqual(t, token2, token)

		// Test column default transformer
		ac, err = tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			ID:                 uuid.Must(uuid.NewV4()),
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column: userstore.ResourceID{Name: colName},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		executeAccessorResp, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields = map[string]string{}
		err = json.Unmarshal([]byte(executeAccessorResp.Data[0]), &fields)
		assert.NoErr(t, err)

		// since default transformer is passthrough, the value should be the same as the input
		assert.Equal(t, fields[colName], pn)

		// Update column with a transformer that always returns the string "blah"
		transformerBlahName := uniqueName("transformerblah")
		transformerBlah, err := tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name:           transformerBlahName,
				Function:       `function transform(data, params) { return 'blah'; }`,
				Parameters:     "",
				InputDataType:  datatype.String,
				OutputDataType: datatype.String,
				TransformType:  policy.TransformTypeTransform,
			})
		assert.NoErr(t, err)

		_, err = tf.UpdateColumn(col.ID, col.Name, col.DataType, col.IsArray, col.DefaultValue,
			userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: transformerBlah.ID}, col.IndexType)
		assert.NoErr(t, err)

		executeAccessorResp, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)

		fields = map[string]string{}
		err = json.Unmarshal([]byte(executeAccessorResp.Data[0]), &fields)
		assert.NoErr(t, err)

		// verify that the column's default transformer was applied
		assert.Equal(t, fields[colName], "blah")

		// Update the text column to have a deny-all access policy
		_, err = tf.UpdateColumn(col.ID, col.Name, col.DataType, col.IsArray, col.DefaultValue,
			userstore.ResourceID{ID: policy.AccessPolicyDenyAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, col.IndexType)
		assert.NoErr(t, err)

		executeAccessorResp, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.True(t, len(executeAccessorResp.Data) == 0)

		// Update the accessor to override column access policies
		ac.AreColumnAccessPoliciesOverridden = true
		ac, err = tf.IDPClient.UpdateAccessor(tf.Ctx, ac.ID, *ac)
		assert.NoErr(t, err)

		executeAccessorResp, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.True(t, len(executeAccessorResp.Data) == 1)
	})

	t.Run("test_update_column", func(t *testing.T) {
		colName := uniqueName("column")
		createdCol := tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		// create an accessor referencing this one column
		acc, err := tf.IDPClient.CreateAccessor(tf.Ctx,
			userstore.Accessor{
				Name:               uniqueName("accessor"),
				DataLifeCycleState: "live",
				Columns:            []userstore.ColumnOutputConfig{{Column: userstore.ResourceID{ID: createdCol.ID}, Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}},
				AccessPolicy:       userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: fmt.Sprintf("{%s} = ?", colName),
				},
				Purposes: []userstore.ResourceID{{Name: "operational"}},
			})
		assert.NoErr(t, err)

		// create a mutator referencing this one column
		mut, err := tf.IDPClient.CreateMutator(tf.Ctx,
			userstore.Mutator{
				Name:         uniqueName("mutator"),
				Columns:      []userstore.ColumnInputConfig{{Column: userstore.ResourceID{ID: createdCol.ID}, Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}},
				AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: fmt.Sprintf("{%s} = ?", colName),
				},
			})
		assert.NoErr(t, err)

		// update a column successfully
		updatedName := uniqueName("column")
		updatedCol, err := tf.UpdateColumn(createdCol.ID, updatedName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, err)
		assert.Equal(t, updatedCol.ID, createdCol.ID)
		assert.Equal(t, updatedCol.Name, updatedName)

		foundCol, err := tf.IDPClient.GetColumn(tf.Ctx, updatedCol.ID)
		assert.NoErr(t, err)
		assert.Equal(t, foundCol, updatedCol)

		// check that the accessor's selector config was updated
		foundAcc, err := tf.IDPClient.GetAccessor(tf.Ctx, acc.ID)
		assert.NoErr(t, err)
		assert.Equal(t, foundAcc.SelectorConfig.WhereClause, fmt.Sprintf("{%s} = ?", updatedName))

		// check that the mutator's selector config was updated
		foundMut, err := tf.IDPClient.GetMutator(tf.Ctx, mut.ID)
		assert.NoErr(t, err)
		assert.Equal(t, foundMut.SelectorConfig.WhereClause, fmt.Sprintf("{%s} = ?", updatedName))

		// create a second column
		anotherColName := uniqueName("column")
		secondCol := tf.CreateValidColumn(anotherColName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		// update a column with id that does not match column id - should fail
		_, err = tf.IDPClient.UpdateColumn(tf.Ctx, secondCol.ID, *createdCol)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// attempt to update a non-existent column - should fail
		_, err = tf.UpdateColumn(badID, "column", datatype.Timestamp, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusNotFound)

		// update a column with empty name - should fail
		_, err = tf.UpdateColumn(updatedCol.ID, "", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update a column with invalid name - should fail
		_, err = tf.UpdateColumn(updatedCol.ID, "foo.bar", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update a column with invalid column type - should fail
		badDataType := userstore.ResourceID{Name: "invalid"}
		_, err = tf.UpdateColumn(updatedCol.ID, anotherColName, badDataType, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NotNil(t, err)

		// update a column trying to change type or array
		_, err = tf.UpdateColumn(secondCol.ID, anotherColName, datatype.Integer, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		_, err = tf.UpdateColumn(secondCol.ID, anotherColName, datatype.String, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update column with bad default value - should fail
		goodIntCol := tf.CreateValidColumn(uniqueName("column"), datatype.Integer, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		_, err = tf.UpdateColumn(goodIntCol.ID, goodIntCol.Name, goodIntCol.DataType, goodIntCol.IsArray, "bad_default", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, goodIntCol.IndexType)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update column with good default value - should succeed
		goodIntCol.DefaultValue = "42"
		updatedIntCol, err := tf.UpdateColumn(goodIntCol.ID, goodIntCol.Name, goodIntCol.DataType, goodIntCol.IsArray, goodIntCol.DefaultValue, userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, goodIntCol.IndexType)
		assert.NoErr(t, err)
		assert.Equal(t, goodIntCol, updatedIntCol)

		foundIntCol, err := tf.IDPClient.GetColumn(tf.Ctx, goodIntCol.ID)
		assert.NoErr(t, err)
		assert.Equal(t, goodIntCol, foundIntCol)

		// update array column with default value - should fail
		goodIntArrayCol := tf.CreateValidColumn(uniqueName("column"), datatype.Integer, true, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		_, err = tf.UpdateColumn(goodIntArrayCol.ID, goodIntArrayCol.Name, goodIntArrayCol.DataType, goodIntArrayCol.IsArray, "42", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, goodIntArrayCol.IndexType)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_create_accessor", func(t *testing.T) {
		colName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		accessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey1")

		// successfully create an accessor with one column
		accessorName := uniqueName("accessor")
		createdAccessor, err := tf.CreateLiveAccessor(
			accessorName,
			accessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.NoErr(t, err)

		// try to create an accessor with the same name but same contents - should fail
		_, err = tf.CreateLiveAccessor(
			accessorName,
			accessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr := jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.True(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdAccessor.ID)

		// try to create an accessor with the same name and different contents - should fail
		_, err = tf.CreateLiveAccessor(
			accessorName,
			policy.AccessPolicyAllowAll.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.False(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdAccessor.ID)

		// try to create an accessor with the same ID and same contents - should fail
		badAccessor := *createdAccessor
		_, err = tf.IDPClient.CreateAccessor(tf.Ctx, badAccessor)
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.True(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdAccessor.ID)

		// try to create an accessor with the same ID and different contents - should fail
		badAccessor = *createdAccessor
		badAccessor.Name = uniqueName("accessor")
		badAccessor.AccessPolicy.ID = policy.AccessPolicyAllowAll.ID
		badAccessor.AccessPolicy.Name = ""
		_, err = tf.IDPClient.CreateAccessor(tf.Ctx, badAccessor)
		assert.HTTPError(t, err, http.StatusConflict)

		sdkErr = jsonclient.GetDetailedErrorInfo(err)
		assert.NotNil(t, sdkErr)
		assert.False(t, sdkErr.Identical)
		assert.Equal(t, sdkErr.ID, createdAccessor.ID)

		// try to create an accessor with no columns - should fail
		_, err = tf.CreateLiveAccessor(
			uniqueName("accessor"),
			accessPolicy.ID,
			[]string{},
			[]uuid.UUID{},
			[]string{"operational"})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create an accessor with bad column - should fail
		_, err = tf.CreateLiveAccessor(
			uniqueName("accessor"),
			accessPolicy.ID,
			[]string{"bad column"},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create an accessor with invalid access policy id - should fail
		_, err = tf.CreateLiveAccessor(
			uniqueName("accessor"),
			uuid.Nil,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create an accessor with nil transformer id - should succeed, as nil indicates to use default transformer for the column
		_, err = tf.CreateLiveAccessor(
			uniqueName("accessor"),
			accessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{uuid.Nil},
			[]string{"operational"})
		assert.NoErr(t, err)

		// try to create an accessor with more than one column - should succeed
		anotherColName := uniqueName("column")
		tf.CreateValidColumn(anotherColName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		_, err = tf.CreateLiveAccessor(
			uniqueName("accessor"),
			accessPolicy.ID,
			[]string{colName, anotherColName},
			[]uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.NoErr(t, err)
	})

	t.Run("test_get_accessor", func(t *testing.T) {
		// try GetOne for non-existent accessor
		_, err := tf.IDPClient.GetAccessor(tf.Ctx, badID)
		assert.HTTPError(t, err, http.StatusNotFound)

		// create valid accessor and ensure GetOne and GetAll work
		createdAccessor := tf.CreateValidAccessor(uniqueName("accessor"))

		foundAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, createdAccessor.ID)
		assert.NoErr(t, err)
		// TODO: remove this once we get rid of TokenAccessPolicy
		foundAccessor.TokenAccessPolicy = createdAccessor.TokenAccessPolicy
		assert.Equal(t, foundAccessor, createdAccessor)

		foundAccessors, err := tf.IDPClient.ListAccessors(tf.Ctx, false)
		assert.NoErr(t, err)

		accessorFound := false
		for _, foundAccessor := range foundAccessors.Data {
			if foundAccessor.ID == createdAccessor.ID {
				// TODO: remove this once we get rid of TokenAccessPolicy
				foundAccessor.TokenAccessPolicy = createdAccessor.TokenAccessPolicy
				assert.Equal(t, createdAccessor, &foundAccessor)
				accessorFound = true
				break
			}
		}
		assert.True(t, accessorFound)

		// look up accessors using a column id filter
		assert.True(t, len(foundAccessor.Columns) > 0)
		assert.True(t, foundAccessor.Columns[0].Column.ID != uuid.Nil)
		columnID := foundAccessor.Columns[0].Column.ID
		filter := pagination.Filter(fmt.Sprintf(`('column_ids',HAS,'%v')`, columnID))
		foundAccessors, err = tf.IDPClient.ListAccessors(tf.Ctx, false, idp.Pagination(filter))
		assert.NoErr(t, err)
		assert.Equal(t, len(foundAccessors.Data), 2)

		// make sure no accessors returned with invalid column id filter
		filter = pagination.Filter(fmt.Sprintf(`('column_ids',HAS,'%v')`, uuid.Nil))
		foundAccessors, err = tf.IDPClient.ListAccessors(tf.Ctx, false, idp.Pagination(filter))
		assert.NoErr(t, err)
		assert.Equal(t, len(foundAccessors.Data), 0)
	})

	t.Run("test_update_accessor", func(t *testing.T) {
		// set up an accessor
		colName := uniqueName("column")
		anotherColName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		tf.CreateValidColumn(anotherColName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		firstAccessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey1")

		firstAccessor, err := tf.CreateLiveAccessor(
			uniqueName("accessor"),
			firstAccessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"})
		assert.NoErr(t, err)

		// update an accessor successfully
		firstAccessor.Name = uniqueName("accessor")
		firstAccessor.Columns = append(firstAccessor.Columns, userstore.ColumnOutputConfig{
			Column:      userstore.ResourceID{Name: anotherColName},
			Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}})
		updatedAccessor, err := tf.IDPClient.UpdateAccessor(tf.Ctx, firstAccessor.ID, *firstAccessor)
		assert.NoErr(t, err)

		foundAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		// TODO: remove this once we get rid of TokenAccessPolicy
		foundAccessor.TokenAccessPolicy = updatedAccessor.TokenAccessPolicy
		assert.Equal(t, foundAccessor, updatedAccessor)

		foundAccessors, err := tf.IDPClient.ListAccessors(tf.Ctx, false)
		assert.NoErr(t, err)

		accessorFound := false
		for _, foundAccessor := range foundAccessors.Data {
			if foundAccessor.ID == updatedAccessor.ID {
				// TODO: remove this once we get rid of TokenAccessPolicy
				foundAccessor.TokenAccessPolicy = updatedAccessor.TokenAccessPolicy
				assert.Equal(t, updatedAccessor, &foundAccessor)
				accessorFound = true
				break
			}
		}
		assert.True(t, accessorFound)

		// create a second accessor, attempt to update the first accessor's name to the second - should fail
		secondAccessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey2")

		nonUserCol, err := tf.CreateTableColumn("othertable", uniqueName("column"), datatype.String, false, "", userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		assert.NoErr(t, err)

		secondAccessor, err := tf.CreateAccessorWithWhereClause(
			uniqueName("accessor"),
			userstore.DataLifeCycleStateLive,
			[]string{""},
			[]uuid.UUID{nonUserCol.ID},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"},
			"{id} = ?",
			secondAccessPolicy.ID,
		)
		assert.NoErr(t, err)

		badAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.Name = secondAccessor.Name
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusConflict)

		// attempt to update the second accessor's non-user-table column to have a transformer that tokenizes by reference -- should fail
		// First try one that tokenizes by reference, and verify that it doesn't work
		transformer := &policy.Transformer{
			Name:           "TokenizeByReferenceTransformer",
			Description:    "a transformer that tokenizes a string by reference",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTokenizeByReference,
			Function:       "function transform(x, y) { return x; }",
		}
		transformer, err = tf.IDPClient.CreateTransformer(tf.Ctx, *transformer)
		assert.NoErr(t, err)

		secondAccessor.Columns[0].Transformer = userstore.ResourceID{ID: transformer.ID}
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, secondAccessor.ID, *secondAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update an accessor with id that does not match accessor id - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.ID = badID
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, firstAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// attempt to update a non-existent accessor - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.ID = badID
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusNotFound)

		// update an accessor with an empty name - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.Name = ""
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update an accessor to have no columns - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.Columns = []userstore.ColumnOutputConfig{}
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update an accessor to use a bad column - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.Columns = []userstore.ColumnOutputConfig{
			{Column: userstore.ResourceID{Name: "bad_column"}, Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}}
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update an accessor with invalid access policy id - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.AccessPolicy.ID = badID
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update an accessor with invalid transformer id - should fail
		badAccessor, err = tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		badAccessor.Columns[0].Transformer.ID = badID
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, badAccessor.ID, *badAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update the accessor to use the same access policy as the second accessor
		goodAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, firstAccessor.ID)
		assert.NoErr(t, err)
		goodAccessor.AccessPolicy = userstore.ResourceID{ID: secondAccessPolicy.ID}
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, goodAccessor.ID, *goodAccessor)
		assert.NoErr(t, err)

		// verify that we can delete the original access policy
		err = tf.IDPClient.DeleteAccessPolicy(tf.Ctx, firstAccessPolicy.ID, firstAccessPolicy.Version)
		assert.NoErr(t, err)

		// verify that the default accessor cannot be updated
		getUserAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, constants.GetUserAccessorID)
		assert.NoErr(t, err)
		getUserAccessor.Name = uniqueName("accessor")
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, getUserAccessor.ID, *getUserAccessor)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_delete_accessor", func(t *testing.T) {
		createdAccessor := tf.CreateValidAccessor(uniqueName("accessor"))

		// delete accessor successfully, make sure not found via GetOne or GetAll
		err := tf.IDPClient.DeleteAccessor(tf.Ctx, createdAccessor.ID)
		assert.NoErr(t, err)

		_, err = tf.IDPClient.GetAccessor(tf.Ctx, createdAccessor.ID)
		assert.HTTPError(t, err, http.StatusNotFound)

		foundAccessors, err := tf.IDPClient.ListAccessors(tf.Ctx, false)
		assert.NoErr(t, err)

		for _, foundAccessor := range foundAccessors.Data {
			assert.NotEqual(t, createdAccessor.ID, foundAccessor.ID)
		}

		// delete a non-existent accessor - should fail
		err = tf.IDPClient.DeleteAccessor(tf.Ctx, badID)
		assert.HTTPError(t, err, http.StatusNotFound)

		// verify that the default accessor cannot be deleted
		getUserAccessor, err := tf.IDPClient.GetAccessor(tf.Ctx, constants.GetUserAccessorID)
		assert.NoErr(t, err)
		err = tf.IDPClient.DeleteAccessor(tf.Ctx, getUserAccessor.ID)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_create_mutator", func(t *testing.T) {
		colName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		accessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey1")

		// successfully create a mutator with one column
		mutatorName := uniqueName("mutator")
		createdMutator, err := tf.CreateMutator(
			mutatorName,
			accessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)

		// try to create a mutator with the same name - should fail
		_, err = tf.CreateMutator(
			mutatorName,
			accessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.HTTPError(t, err, http.StatusConflict)

		// try to create a mutator with the same ID - should fail
		badMutator := *createdMutator
		badMutator.Name = uniqueName("mutator")
		_, err = tf.IDPClient.CreateMutator(tf.Ctx, badMutator)
		assert.HTTPError(t, err, http.StatusConflict)

		// try to create a mutator with no columns - should fail
		_, err = tf.CreateMutator(
			uniqueName("mutator"),
			accessPolicy.ID,
			[]string{},
			[]uuid.UUID{})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create a mutator with bad column - should fail
		_, err = tf.CreateMutator(
			uniqueName("mutator"),
			accessPolicy.ID,
			[]string{"bad column"},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create a mutator with invalid access policy id - should fail
		_, err = tf.CreateMutator(
			uniqueName("mutator"),
			uuid.Nil,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.HTTPError(t, err, http.StatusBadRequest)

		// try to create a mutator with more than one column - should succeed
		anotherColName := uniqueName("column")
		tf.CreateValidColumn(anotherColName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		_, err = tf.CreateMutator(
			uniqueName("mutator"),
			accessPolicy.ID,
			[]string{colName, anotherColName},
			[]uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)
	})

	t.Run("test_get_mutator", func(t *testing.T) {
		// try GetOne for non-existent mutator
		_, err := tf.IDPClient.GetMutator(tf.Ctx, badID)
		assert.NotNil(t, err)

		// create valid mutator and ensure GetOne and GetAll work
		createdMutator := tf.CreateValidMutator(uniqueName("mutator"))

		foundMutator, err := tf.IDPClient.GetMutator(tf.Ctx, createdMutator.ID)
		assert.NoErr(t, err)
		assert.Equal(t, foundMutator, createdMutator)

		foundMutators, err := tf.IDPClient.ListMutators(tf.Ctx, false)
		assert.NoErr(t, err)

		mutatorFound := false
		for _, foundMutator := range foundMutators.Data {
			if foundMutator.ID == createdMutator.ID {
				assert.Equal(t, createdMutator, &foundMutator)
				mutatorFound = true
				break
			}
		}
		assert.True(t, mutatorFound)
	})

	t.Run("test_update_mutator", func(t *testing.T) {
		// set up a mutator
		colName := uniqueName("column")
		anotherColName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		tf.CreateValidColumn(anotherColName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		firstAccessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey1")

		firstMutator, err := tf.CreateMutator(
			uniqueName("mutator"),
			firstAccessPolicy.ID,
			[]string{colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)

		// update a mutator successfully
		firstMutator = getUpdatableMutator(firstMutator)
		firstMutator.Name = uniqueName("mutator")
		firstMutator.Columns = []userstore.ColumnInputConfig{
			{Column: userstore.ResourceID{Name: colName}, Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}},
			{Column: userstore.ResourceID{Name: anotherColName}, Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}}
		updatedMutator, err := tf.IDPClient.UpdateMutator(tf.Ctx, firstMutator.ID, *firstMutator)
		assert.NoErr(t, err)

		foundMutator, err := tf.IDPClient.GetMutator(tf.Ctx, firstMutator.ID)
		assert.NoErr(t, err)
		assert.Equal(t, foundMutator, updatedMutator)

		foundMutators, err := tf.IDPClient.ListMutators(tf.Ctx, false)
		assert.NoErr(t, err)

		mutatorFound := false
		for _, foundMutator := range foundMutators.Data {
			if foundMutator.ID == updatedMutator.ID {
				assert.Equal(t, updatedMutator, &foundMutator)
				mutatorFound = true
				break
			}
		}
		assert.True(t, mutatorFound)

		// create a second mutator, attempt to update the first mutator's name to the second - should fail
		secondAccessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey2")

		secondMutator, err := tf.CreateMutator(
			uniqueName("mutator"),
			secondAccessPolicy.ID,
			[]string{anotherColName},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)

		badMutator := getUpdatableMutator(firstMutator)
		badMutator.Name = secondMutator.Name
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusConflict)

		// update a mutator with id that does not match mutator id - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.ID = badID
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, firstMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// attempt to update a non-existent mutator - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.ID = badID
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusNotFound)

		// update a mutator with an empty name - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.Name = ""
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update a mutator to have no columns - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.Columns = []userstore.ColumnInputConfig{}
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update a mutator to use a bad column - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.Columns = []userstore.ColumnInputConfig{
			{Column: userstore.ResourceID{Name: "bad column"}, Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}}
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update a mutator with invalid access policy id - should fail
		badMutator = getUpdatableMutator(firstMutator)
		badMutator.AccessPolicy.ID = badID
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, badMutator.ID, *badMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// update the mutator to use a different access policy
		foundMutator, err = tf.IDPClient.GetMutator(tf.Ctx, firstMutator.ID)
		assert.NoErr(t, err)
		foundMutator.AccessPolicy = userstore.ResourceID{ID: secondAccessPolicy.ID}
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, foundMutator.ID, *foundMutator)
		assert.NoErr(t, err)

		// verify that we can delete the original access policy
		err = tf.IDPClient.DeleteAccessPolicy(tf.Ctx, firstAccessPolicy.ID, firstAccessPolicy.Version)
		assert.NoErr(t, err)

		// verify that the default mutator cannot be updated
		updateUserMutator, err := tf.IDPClient.GetMutator(tf.Ctx, constants.UpdateUserMutatorID)
		assert.NoErr(t, err)
		updateUserMutator.Name = uniqueName("mutator")
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, updateUserMutator.ID, *updateUserMutator)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_delete_mutator", func(t *testing.T) {
		createdMutator := tf.CreateValidMutator(uniqueName("mutator"))

		// delete mutator successfully, make sure not found via GetOne or GetAll
		err := tf.IDPClient.DeleteMutator(tf.Ctx, createdMutator.ID)
		assert.NoErr(t, err)

		_, err = tf.IDPClient.GetMutator(tf.Ctx, createdMutator.ID)
		assert.HTTPError(t, err, http.StatusNotFound)

		foundMutators, err := tf.IDPClient.ListMutators(tf.Ctx, false)
		assert.NoErr(t, err)

		for _, foundMutator := range foundMutators.Data {
			assert.NotEqual(t, createdMutator.ID, foundMutator.ID)
		}

		// delete a non-existent mutator - should fail
		err = tf.IDPClient.DeleteMutator(tf.Ctx, badID)
		assert.HTTPError(t, err, http.StatusNotFound)

		// verify that the default mutator cannot be deleted
		updateUserMutator, err := tf.IDPClient.GetMutator(tf.Ctx, constants.UpdateUserMutatorID)
		assert.NoErr(t, err)
		err = tf.IDPClient.DeleteMutator(tf.Ctx, updateUserMutator.ID)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_unique_column_values", func(t *testing.T) {
		// external_alias is a unique string column; ensure that we can
		// create users successfully when not specifying a column value
		// for external_alias, or when specifying a nil value (which is
		// equivalent to deleting the column value)
		_, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "foo@schmo.org"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "bar@schmo.org"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "baz@schmo.org", "external_alias": nil}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "biz@schmo.org", "external_alias": nil}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
	})

	t.Run("test_external_alias", func(t *testing.T) {
		tt := testlogtransport.InitLoggerAndTransportsForTests(tf.T)

		colName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		alias := uniqueName("alias")
		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"external_alias": alias}, idp.OrganizationID(tf.Company.ID))

		assert.NoErr(t, err)
		assert.NotNil(t, uid)

		got, err := tf.IDPClient.GetUser(tf.Ctx, uid)
		assert.NoErr(t, err)
		assert.Equal(t, got.Profile["external_alias"], alias)

		ac, err := tf.IDPClient.CreateAccessor(tf.Ctx, userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:      userstore.ResourceID{Name: colName},
					Transformer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = (((?)))",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		})
		assert.NoErr(t, err)

		alias = uniqueName("alias")
		pn := "+1 555 555 1212"
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{colName: pn, "external_alias": alias}, idp.OrganizationID(tf.Company.ID))

		assert.NoErr(t, err)
		resp, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.Equal(t, resp.Data, []string{fmt.Sprintf(`{"%s":"%s"}`, colName, pn)})

		m, err := tf.IDPClient.CreateMutator(tf.Ctx, userstore.Mutator{
			Name:         uniqueName("mutator"),
			Columns:      []userstore.ColumnInputConfig{{Column: userstore.ResourceID{Name: colName}, Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}}},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{external_alias} = ?",
			},
		})
		assert.NoErr(t, err)

		new_pn := "+1 111 111 1111"
		_, err = tf.IDPClient.ExecuteMutator(tf.Ctx, m.ID, nil, []any{alias}, map[string]idp.ValueAndPurposes{
			colName: {Value: new_pn, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}})
		assert.NoErr(t, err)
		resp, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{alias})
		assert.NoErr(t, err)
		assert.Equal(t, resp.Data, []string{fmt.Sprintf(`{"%s":"%s"}`, colName, new_pn)})

		// Clean up the accessor
		err = tf.IDPClient.DeleteAccessor(tf.Ctx, ac.ID)
		assert.NoErr(t, err)

		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryCall, events.AccessorPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryDuration, events.AccessorPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryResultSuccess, events.AccessorPrefix, "", 0))), 2)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryConfig, 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryAccessDenied, 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryNotFound, 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryTransformError, 0))), 0)
		assert.Equal(t, len(tt.GetEventsLoggedByName(events.GetEventName(ac.ID, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryValidationError, 0))), 0)
	})

	t.Run("test_organizations", func(t *testing.T) {
		// Create a new organization
		org, err := tf.AuthzClient.CreateOrganization(tf.Ctx, uuid.Nil, uniqueName("org"), "aws-us-west-2")
		assert.NoErr(t, err)

		// Create a new user
		userID, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"external_alias": uniqueName("alias")}, idp.OrganizationID(org.ID))
		assert.NoErr(t, err)
		user, err := tf.IDPClient.GetUser(tf.Ctx, userID)
		assert.NoErr(t, err)
		assert.Equal(t, user.OrganizationID, org.ID)

		// Check that this user shows up in the list of users for this organization
		users, err := tf.IDPMgmtClient.ListUserBaseProfiles(tf.Ctx, idp.OrganizationID(org.ID))
		assert.NoErr(t, err)
		assert.Equal(t, len(users.Data), 1)
		assert.Equal(t, users.Data[0].ID, userID.String())
		assert.Equal(t, users.Data[0].OrganizationID, org.ID.String())

		// Create a new organization
		org2, err := tf.AuthzClient.CreateOrganization(tf.Ctx, uuid.Nil, uniqueName("org"), "aws-us-west-2")
		assert.NoErr(t, err)

		// Verify that the user created above doesn't show up in the other organization
		users, err = tf.IDPMgmtClient.ListUserBaseProfiles(tf.Ctx, idp.OrganizationID(org2.ID))
		assert.NoErr(t, err)
		assert.Equal(t, len(users.Data), 0)
	})

	t.Run("test_purpose", func(t *testing.T) {
		purposeName := uniqueName("purpose")
		p, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name:        purposeName,
			Description: "purpose_description",
		})
		assert.NoErr(t, err)
		assert.Equal(t, p.Name, purposeName)
		assert.Equal(t, p.Description, "purpose_description")

		p.Description = "new_purpose_description"
		p, err = tf.IDPClient.UpdatePurpose(tf.Ctx, *p)
		assert.NoErr(t, err)
		assert.Equal(t, p.Name, purposeName)
		assert.Equal(t, p.Description, "new_purpose_description")

		newPurposeName := uniqueName("purpose")
		p.Name = newPurposeName
		_, err = tf.IDPClient.UpdatePurpose(tf.Ctx, *p)
		assert.HTTPError(t, err, http.StatusBadRequest)

		p, err = tf.IDPClient.GetPurpose(tf.Ctx, p.ID)
		assert.NoErr(t, err)
		assert.Equal(t, p.Name, purposeName)
		assert.Equal(t, p.Description, "new_purpose_description")

		foundPurposes, err := tf.IDPClient.ListPurposes(tf.Ctx)
		assert.NoErr(t, err)

		purposeFound := false
		for _, foundPurpose := range foundPurposes.Data {
			if foundPurpose.ID == p.ID {
				assert.Equal(t, p, &foundPurpose)
				purposeFound = true
				break
			}
		}
		assert.True(t, purposeFound)

		err = tf.IDPClient.DeletePurpose(tf.Ctx, p.ID)
		assert.NoErr(t, err)

		err = tf.IDPClient.DeletePurpose(tf.Ctx, constants.OperationalPurposeID)
		assert.HTTPError(t, err, http.StatusBadRequest)
	})

	t.Run("test_purpose_deletion_cleans_up_retention_durations", func(t *testing.T) {
		// Create a purpose
		purposeName := uniqueName("purpose")
		p, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name:        purposeName,
			Description: "purpose_for_retention_test",
		})
		assert.NoErr(t, err)

		// Create a column value retention duration for the purpose
		retentionDuration := idp.RetentionDuration{
			Unit:     idp.DurationUnitDay,
			Duration: 365,
		}

		columnRetentionDuration := idp.ColumnRetentionDuration{
			PurposeID: p.ID,
			Duration:  retentionDuration,
		}

		// Create the retention duration for the purpose
		resp, err := tf.IDPClient.CreateColumnRetentionDurationForPurpose(
			tf.Ctx,
			userstore.DataLifeCycleStateLive,
			p.ID,
			columnRetentionDuration,
		)
		assert.NoErr(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.RetentionDuration.ID)

		// Verify the retention duration was created
		getDurationResp, err := tf.IDPClient.GetColumnRetentionDurationForPurpose(
			tf.Ctx,
			userstore.DataLifeCycleStateLive,
			p.ID,
		)
		assert.NoErr(t, err)
		assert.NotNil(t, getDurationResp)
		assert.NotNil(t, getDurationResp.RetentionDuration)

		err = tf.IDPClient.DeletePurpose(tf.Ctx, p.ID)
		assert.NoErr(t, err)

		// Verify the retention duration was also deleted
		_, err = tf.IDPClient.GetColumnRetentionDurationForPurpose(
			tf.Ctx,
			userstore.DataLifeCycleStateLive,
			p.ID,
		)
		assert.HTTPError(t, err, http.StatusNotFound)
	})

	t.Run("test_purpose_in_accessor_and_mutator", func(t *testing.T) {
		purposeName := uniqueName("purpose")
		_, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name:        purposeName,
			Description: "used for marketing",
		})
		assert.NoErr(t, err)

		accessor, err := tf.CreateLiveAccessor(uniqueName("accessor"),
			policy.AccessPolicyAllowAll.ID,
			[]string{"email"},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational", purposeName})
		assert.NoErr(t, err)

		mutator, err := tf.CreateMutator(uniqueName("mutator"),
			policy.AccessPolicyAllowAll.ID,
			[]string{"email"},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)

		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "joe@bigcorp.com"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		ret, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.Equal(t, len(ret.Data), 0)

		consentedPurposes, err := tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, []userstore.ResourceID{{Name: "email"}})
		assert.NoErr(t, err)

		assert.Equal(t, len(consentedPurposes.Data), 1)
		assert.Equal(t, consentedPurposes.Data[0].Column.Name, "email")
		assert.Equal(t, len(consentedPurposes.Data[0].ConsentedPurposes), 1)
		assert.Equal(t, consentedPurposes.Data[0].ConsentedPurposes[0].Name, "operational")

		_, err = tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, []any{uid}, map[string]idp.ValueAndPurposes{
			"email": {Value: "joe@bigcorp.com", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}, {Name: purposeName}}}})
		assert.NoErr(t, err)

		ret, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.Equal(t, len(ret.Data), 1, assert.Must())
		assert.Equal(t, ret.Data[0], "{\"email\":\"joe@bigcorp.com\"}")

		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, []userstore.ResourceID{{Name: "email"}})
		assert.NoErr(t, err)

		assert.Equal(t, len(consentedPurposes.Data), 1)
		assert.Equal(t, consentedPurposes.Data[0].Column.Name, "email")
		assert.Equal(t, len(consentedPurposes.Data[0].ConsentedPurposes), 2)
		cps := consentedPurposes.Data[0].ConsentedPurposes
		assert.True(t, cps[0].Name == purposeName || cps[1].Name == purposeName)

		_, err = tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, []any{uid}, map[string]idp.ValueAndPurposes{
			"email": {Value: "joe@bigcorp.com", PurposeDeletions: []userstore.ResourceID{{Name: purposeName}}}})
		assert.NoErr(t, err)

		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, []userstore.ResourceID{{Name: "email"}})
		assert.NoErr(t, err)

		assert.Equal(t, len(consentedPurposes.Data), 1)
		assert.Equal(t, consentedPurposes.Data[0].Column.Name, "email")
		assert.Equal(t, len(consentedPurposes.Data[0].ConsentedPurposes), 1)
		assert.Equal(t, consentedPurposes.Data[0].ConsentedPurposes[0].Name, "operational")
	})

	t.Run("test_get_consented_purposes_for_user", func(t *testing.T) {
		columns := []userstore.ResourceID{{Name: "email"}, {Name: "id"}}
		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "jerry@seinfeld.com"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(tf.T, err)
		consentedPurposes, err := tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, columns)
		assert.NoErr(tf.T, err)
		assert.Equal(t, len(consentedPurposes.Data), 2)
		indexEmail := 0
		if consentedPurposes.Data[0].Column.Name != "email" {
			indexEmail = 1
		}
		assert.Equal(t, consentedPurposes.Data[indexEmail].Column.Name, "email")
		assert.Equal(t, len(consentedPurposes.Data[indexEmail].ConsentedPurposes), 1)
		assert.Equal(t, consentedPurposes.Data[indexEmail].ConsentedPurposes[0].Name, "operational")

		// All columns
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, nil)
		assert.NoErr(t, err)
		assert.True(t, len(consentedPurposes.Data) >= len(defaults.GetDefaultColumns()))

		// Invalid user id
		invalidUserID := uuid.Must(uuid.NewV4())
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, invalidUserID, columns)
		assert.HTTPError(t, err, http.StatusBadRequest)
		assert.Contains(t, err.Error(), fmt.Sprintf("user %v not found", invalidUserID))

		// invalid column
		invalidCols := append(columns, userstore.ResourceID{Name: "costanza"})
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, invalidCols)
		assert.HTTPError(t, err, http.StatusBadRequest)
		assert.Contains(t, err.Error(), "Column with name 'costanza' not found")

		invalidColumnID := uuid.Must(uuid.NewV4())
		invalidCols = append(columns, userstore.ResourceID{ID: invalidColumnID})
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, invalidCols)
		assert.HTTPError(t, err, http.StatusBadRequest)
		assert.Contains(t, err.Error(), fmt.Sprintf("Column with ID %v not found", invalidColumnID))

		// Empty resource ID
		invalidCols = append(columns, userstore.ResourceID{})
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, invalidCols)
		assert.HTTPError(t, err, http.StatusBadRequest)
		assert.Contains(t, err.Error(), "either ID or Name must be set")

		// deleted user
		assert.NoErr(t, tf.IDPClient.DeleteUser(tf.Ctx, uid))
		consentedPurposes, err = tf.IDPClient.GetConsentedPurposesForUser(tf.Ctx, uid, columns)
		assert.HTTPError(t, err, http.StatusBadRequest)
		assert.Contains(t, err.Error(), fmt.Sprintf("user %v not found", uid))
	})

	t.Run("test_purpose_in_access_policy_from_accessor", func(t *testing.T) {
		purposeName := uniqueName("purpose")
		_, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
			Name:        purposeName,
			Description: "used for marketing",
		})
		assert.NoErr(t, err)

		aptName := uniqueName("access_policy_template")
		apt, err := tf.TokenizerClient.CreateAccessPolicyTemplate(
			tf.Ctx,
			policy.AccessPolicyTemplate{
				Name: aptName,
				Function: fmt.Sprintf(`function policy(context, params) {
				return context.server.purpose_names.includes(params.purpose_name);
			} // %s`, aptName),
			},
			idp.IfNotExists(),
		)
		assert.NoErr(t, err)

		ap, err := tf.IDPClient.CreateAccessPolicy(
			tf.Ctx,
			policy.AccessPolicy{
				Name:        uniqueName("access_policy"),
				Description: "test policy",
				Components: []policy.AccessPolicyComponent{
					{
						Template:           &userstore.ResourceID{ID: apt.ID},
						TemplateParameters: fmt.Sprintf(`{ "purpose_name": "%s" }`, purposeName),
					},
				},
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
		)
		assert.NoErr(t, err)

		accessor, err := tf.CreateLiveAccessor(uniqueName("accessor"),
			ap.ID,
			[]string{"email"},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational", purposeName})
		assert.NoErr(t, err)

		mutator, err := tf.CreateMutator(uniqueName("mutator"),
			policy.AccessPolicyAllowAll.ID,
			[]string{"email"},
			[]uuid.UUID{policy.TransformerPassthrough.ID})
		assert.NoErr(t, err)

		uid, err := tf.IDPClient.CreateUserWithMutator(tf.Ctx,
			mutator.ID,
			policy.ClientContext{},
			map[string]idp.ValueAndPurposes{
				"email": {Value: "joe@bigcorp.com",
					PurposeAdditions: []userstore.ResourceID{{Name: "operational"}, {Name: purposeName}}},
			}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		ret, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.True(t, len(ret.Data) == 1)

		// now change the purpose_name in the template parameters to something that isn't in the accessor's purposes
		ap.Components[0].TemplateParameters = `{ "purpose_name": "marketing" }`
		_, err = tf.IDPClient.UpdateAccessPolicy(tf.Ctx, *ap)
		assert.NoErr(t, err)

		ret, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.True(t, len(ret.Data) == 0)
	})

	t.Run("test_tokenization_in_accessor", func(t *testing.T) {
		colName1 := uniqueName("column")
		tf.CreateValidColumn(colName1, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		colName2 := uniqueName("column")
		tf.CreateValidColumn(colName2, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{colName1: "123-456-7890", colName2: "123 Main St"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		assert.NotNil(t, uid)

		got, err := tf.IDPClient.GetUser(tf.Ctx, uid)
		assert.NoErr(t, err)
		assert.Equal(t, got.Profile[colName1], "123-456-7890")
		assert.Equal(t, got.Profile[colName2], "123 Main St")

		tfName := uniqueName("transformer")
		transformer, err := tf.TokenizerClient.CreateTransformer(tf.Ctx,
			policy.Transformer{
				Name: tfName,
				Function: fmt.Sprintf(`function transform(data, params) {
				return data.split("").reverse().join("");
			} // %s`, tfName),
				Parameters:     "",
				InputDataType:  datatype.String,
				OutputDataType: datatype.String,
				TransformType:  policy.TransformTypeTokenizeByReference,
			})
		assert.NoErr(t, err)

		accessorToCreate := userstore.Accessor{
			Name:               uniqueName("accessor"),
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			Columns: []userstore.ColumnOutputConfig{
				{
					Column:            userstore.ResourceID{Name: colName1},
					Transformer:       userstore.ResourceID{ID: transformer.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
				},
				{
					Column:      userstore.ResourceID{Name: colName2},
					Transformer: userstore.ResourceID{ID: transformer.ID},
				},
			},
			AccessPolicy: userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: "{id} = ?",
			},
			Purposes: []userstore.ResourceID{{Name: "operational"}},
		}
		_, err = tf.IDPClient.CreateAccessor(tf.Ctx, accessorToCreate)
		assert.NotNil(t, err)

		accessorToCreate.Columns[1].TokenAccessPolicy = userstore.ResourceID{ID: policy.AccessPolicyDenyAll.ID}
		ac, err := tf.IDPClient.CreateAccessor(tf.Ctx, accessorToCreate)
		assert.NoErr(t, err)

		ret, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, ac.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)

		assert.Equal(t, len(ret.Data), 1, assert.Must())
		var profileMap map[string]string
		assert.NoErr(tf.T, json.Unmarshal([]byte(ret.Data[0]), &profileMap))
		assert.Equal(t, profileMap[colName1], "0987-654-321")
		assert.Equal(t, profileMap[colName2], "tS niaM 321")

		resolved, err := tf.TokenizerClient.ResolveToken(tf.Ctx, "0987-654-321", policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resolved, "123-456-7890")

		resolved, err = tf.TokenizerClient.ResolveToken(tf.Ctx, "tS niaM 321", policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(t, resolved, "")
	})

	t.Run("test_name_uniqueness", func(t *testing.T) {
		// create userstore objects

		prefix1 := uniqueName("prefix")
		c1 := tf.CreateValidColumn(prefix1+"c", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		p1 := tf.CreateValidPurpose(prefix1 + "p")
		apt1 := tf.CreateValidAccessPolicyTemplate(prefix1 + "apt")
		ap1 := tf.CreateValidAccessPolicy(prefix1+"ap", "key1")
		a1 := tf.CreateValidAccessor(prefix1 + "a")
		m1 := tf.CreateValidMutator(prefix1 + "m")
		t1 := tf.CreateValidTransformer(prefix1 + "t")

		// ensure that userstore objects that can change name
		// can update to a differently-cased identical name
		// NOTE: transformers don't allow updates

		c1.Name = prefix1 + "C"
		c1, err := tf.IDPClient.UpdateColumn(tf.Ctx, c1.ID, *c1)
		assert.NoErr(t, err)
		assert.Equal(t, c1.Name, prefix1+"C")

		apt1.Name = prefix1 + "APT"
		apt1, err = tf.TokenizerClient.UpdateAccessPolicyTemplate(tf.Ctx, *apt1)
		assert.NoErr(t, err)
		assert.Equal(t, apt1.Name, prefix1+"APT")

		ap1.Name = prefix1 + "AP"
		ap1, err = tf.TokenizerClient.UpdateAccessPolicy(tf.Ctx, *ap1)
		assert.NoErr(t, err)
		assert.Equal(t, ap1.Name, prefix1+"AP")

		a1.Name = prefix1 + "A"
		a1, err = tf.IDPClient.UpdateAccessor(tf.Ctx, a1.ID, *a1)
		assert.NoErr(t, err)
		assert.Equal(t, a1.Name, prefix1+"A")

		m1 = getUpdatableMutator(m1)
		m1.Name = prefix1 + "M"
		m1, err = tf.IDPClient.UpdateMutator(tf.Ctx, m1.ID, *m1)
		assert.NoErr(t, err)
		assert.Equal(t, m1.Name, prefix1+"M")

		// purpose does not allow name changes

		p1.Name = prefix1 + "P"
		_, err = tf.IDPClient.UpdatePurpose(tf.Ctx, *p1)
		assert.HTTPError(t, err, http.StatusBadRequest)

		// cannot create another userstore object with a matching case-insensitive name,
		// but CreateIfNotExists should return the matching case-insensitive object

		c := *c1
		c.ID = uuid.Nil
		c.Name = prefix1 + "c"
		_, err = tf.IDPClient.CreateColumn(tf.Ctx, c)
		assert.HTTPError(t, err, http.StatusConflict)
		c2, err := tf.IDPClient.CreateColumn(tf.Ctx, c, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, c1.ID, c2.ID)

		p := *p1
		p.ID = uuid.Nil
		_, err = tf.IDPClient.CreatePurpose(tf.Ctx, p)
		assert.HTTPError(t, err, http.StatusConflict)
		p2, err := tf.IDPClient.CreatePurpose(tf.Ctx, p, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, p1.ID, p2.ID)

		apt := *apt1
		apt.ID = uuid.Nil
		apt.Name = prefix1 + "apt"
		_, err = tf.TokenizerClient.CreateAccessPolicyTemplate(tf.Ctx, apt)
		assert.HTTPError(t, err, http.StatusConflict)
		apt2, err := tf.TokenizerClient.CreateAccessPolicyTemplate(tf.Ctx, apt, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, apt1.ID, apt2.ID)

		ap := *ap1
		ap.ID = uuid.Nil
		ap.Name = prefix1 + "ap"
		_, err = tf.TokenizerClient.CreateAccessPolicy(tf.Ctx, ap)
		assert.HTTPError(t, err, http.StatusConflict)
		ap2, err := tf.TokenizerClient.CreateAccessPolicy(tf.Ctx, ap, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, ap1.ID, ap2.ID)

		a := *a1
		a.ID = uuid.Nil
		a.Name = prefix1 + "a"
		_, err = tf.IDPClient.CreateAccessor(tf.Ctx, a)
		assert.HTTPError(t, err, http.StatusConflict)
		a2, err := tf.IDPClient.CreateAccessor(tf.Ctx, a, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, a1.ID, a2.ID)

		m := getUpdatableMutator(m1)
		m.ID = uuid.Nil
		m.Name = prefix1 + "m"
		_, err = tf.IDPClient.CreateMutator(tf.Ctx, *m)
		assert.HTTPError(t, err, http.StatusConflict)
		m2, err := tf.IDPClient.CreateMutator(tf.Ctx, *m, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, m1.ID, m2.ID)

		tfmr := *t1
		tfmr.ID = uuid.Nil
		tfmr.Name = prefix1 + "T"
		_, err = tf.IDPClient.CreateTransformer(tf.Ctx, tfmr)
		assert.HTTPError(t, err, http.StatusConflict)
		tfmr = *t1
		t2, err := tf.IDPClient.CreateTransformer(tf.Ctx, tfmr, idp.IfNotExists())
		assert.NoErr(t, err)
		assert.Equal(t, t1.ID, t2.ID)

		// cannot update another object to have a matching case-insensitive name

		prefix2 := uniqueName("prefix")
		c3 := tf.CreateValidColumn(prefix2+"c", datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)
		apt3 := tf.CreateValidAccessPolicyTemplate(prefix2 + "apt")
		ap3 := tf.CreateValidAccessPolicy(prefix2+"ap", "key1")
		a3 := tf.CreateValidAccessor(prefix2 + "a")
		m3 := tf.CreateValidMutator(prefix2 + "m")

		c3.Name = prefix1 + "c"
		_, err = tf.IDPClient.UpdateColumn(tf.Ctx, c3.ID, *c3)
		assert.HTTPError(t, err, http.StatusConflict)

		apt3.Name = prefix1 + "apt"
		_, err = tf.TokenizerClient.UpdateAccessPolicyTemplate(tf.Ctx, *apt3)
		assert.HTTPError(t, err, http.StatusConflict)

		ap3.Name = prefix1 + "ap"
		_, err = tf.TokenizerClient.UpdateAccessPolicy(tf.Ctx, *ap3)
		assert.HTTPError(t, err, http.StatusConflict)

		a3.Name = prefix1 + "a"
		_, err = tf.IDPClient.UpdateAccessor(tf.Ctx, a3.ID, *a3)
		assert.HTTPError(t, err, http.StatusConflict)

		m3 = getUpdatableMutator(m3)
		m3.Name = prefix1 + "m"
		_, err = tf.IDPClient.UpdateMutator(tf.Ctx, m3.ID, *m3)
		assert.HTTPError(t, err, http.StatusConflict)
	})

	t.Run("test_create_user_invalid", func(t *testing.T) {
		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"jerry": "that's a shame"}, idp.OrganizationID(tf.Company.ID))
		assert.NotNil(t, err)
		assert.True(t, uid.IsNil())
		assert.Contains(t, err.Error(), "Column `jerry` doesn't exist")
	})

	t.Run("test_update_user_conflicts", func(t *testing.T) {
		colName := uniqueName("column")
		tf.CreateValidColumn(colName, datatype.Integer, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeUnique)
		purposeAdditions := []userstore.ResourceID{{Name: "operational"}}

		mutator, err := tf.CreateMutator(
			uniqueName("mutator"),
			policy.AccessPolicyAllowAll.ID,
			[]string{"name", "external_alias", colName},
			[]uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID},
		)
		assert.NoErr(t, err)

		alias := uniqueName("alias")
		_, err = tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"name": "joe", "external_alias": alias, colName: "1"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		// try updating a user to use conflicting values for external_alias and unique column
		anotherAlias := uniqueName("alias")
		user2, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"name": "schmo", "external_alias": anotherAlias, colName: "2"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		updateResp, err := tf.IDPClient.ExecuteMutator(
			tf.Ctx,
			mutator.ID,
			policy.ClientContext{},
			[]any{user2},
			map[string]idp.ValueAndPurposes{
				"name":           {Value: "schmoey", PurposeAdditions: purposeAdditions},
				"external_alias": {Value: alias, PurposeAdditions: purposeAdditions},
				colName:          {Value: "3", PurposeAdditions: purposeAdditions},
			},
		)
		assert.HTTPError(t, err, http.StatusConflict)
		assert.IsNil(t, updateResp)
		assert.Contains(t, err.Error(), fmt.Sprintf(`"error":"user '%v' cannot update unique column 'external_alias' - value '%s' is already in use"`, user2, alias))

		updateResp, err = tf.IDPClient.ExecuteMutator(
			tf.Ctx,
			mutator.ID,
			policy.ClientContext{},
			[]any{user2},
			map[string]idp.ValueAndPurposes{
				"name":           {Value: "schmoey", PurposeAdditions: purposeAdditions},
				"external_alias": {Value: uniqueName("alias"), PurposeAdditions: purposeAdditions},
				colName:          {Value: "1", PurposeAdditions: purposeAdditions},
			},
		)
		assert.HTTPError(t, err, http.StatusConflict)
		assert.IsNil(t, updateResp)
		assert.Contains(t, err.Error(), fmt.Sprintf(`"error":"user '%v' cannot update unique column '%s' - value '1' is already in use"`, user2, colName))

		// user name and external alias should remain unchanged
		getResp, err := tf.IDPClient.GetUser(tf.Ctx, user2)
		assert.NoErr(t, err)
		assert.Equal(t, getResp.Profile["name"], "schmo")
		assert.Equal(t, getResp.Profile["external_alias"], anotherAlias)
		assert.Equal(t, getResp.Profile[colName], "2")

		// test a user with no original value for external alias or unique column
		user3, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"name": "bill"}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		updateResp, err = tf.IDPClient.ExecuteMutator(
			tf.Ctx,
			mutator.ID,
			policy.ClientContext{},
			[]any{user3},
			map[string]idp.ValueAndPurposes{
				"name":           {Value: "billy", PurposeAdditions: purposeAdditions},
				"external_alias": {Value: alias, PurposeAdditions: purposeAdditions},
				colName:          {Value: "3", PurposeAdditions: purposeAdditions},
			},
		)
		assert.NotNil(t, err)
		assert.IsNil(t, updateResp)
		assert.Contains(t, err.Error(), fmt.Sprintf(`"error":"user '%v' cannot update unique column 'external_alias' - value '%s' is already in use"`, user3, alias))

		// user name, external alias, and unique column should remain unchanged
		getResp, err = tf.IDPClient.GetUser(tf.Ctx, user3)
		assert.NoErr(t, err)
		assert.Equal(t, getResp.Profile["name"], "bill")
		assert.IsNil(t, getResp.Profile["external_alias"])
		assert.IsNil(t, getResp.Profile[colName])
	})

	t.Run("test_user_object", func(t *testing.T) {
		uid := uuid.Must(uuid.NewV4())
		userID, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{}, idp.UserID(uid), idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		assert.Equal(t, userID, uid)

		userObj, err := tf.AuthzClient.GetObject(tf.Ctx, userID)
		assert.NoErr(t, err)
		assert.Equal(t, userObj.ID, userID)

		_, err = tf.AuthzClient.CreateObject(tf.Ctx, userID, authz.UserObjectTypeID, "", authz.OrganizationID(tf.Company.ID), authz.IfNotExists())
		assert.NoErr(t, err)
	})

	t.Run("test_user_edge", func(t *testing.T) {
		userID, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)

		ot1, err := tf.AuthzClient.CreateObjectType(tf.Ctx, uuid.Must(uuid.NewV4()), uniqueName("object_type"))
		assert.NoErr(t, err)

		obj, err := tf.AuthzClient.CreateObject(tf.Ctx, uuid.Must(uuid.NewV4()), ot1.ID, "")
		assert.NoErr(t, err)

		et1, err := tf.AuthzClient.CreateEdgeType(tf.Ctx, uuid.Must(uuid.NewV4()), ot1.ID, authz.UserObjectTypeID, uniqueName("edge_type"), authz.Attributes{})
		assert.NoErr(t, err)

		// Ensure we can create an edge between a user and an object.
		edge, err := tf.AuthzClient.CreateEdge(tf.Ctx, uuid.Must(uuid.NewV4()), obj.ID, userID, et1.ID)
		assert.NoErr(t, err)

		gotEdge, err := tf.AuthzClient.GetEdge(tf.Ctx, edge.ID)
		assert.NoErr(t, err)
		assert.Equal(t, edge, gotEdge)

		// Ensure that deleting the user deletes the edge.
		err = tf.IDPClient.DeleteUser(tf.Ctx, userID)
		assert.NoErr(t, err)

		_, err = tf.AuthzClient.GetEdge(tf.Ctx, edge.ID)
		assert.NotNil(t, err, assert.Must())
	})

	t.Run("test_externally_issued_token", func(t *testing.T) {
		s := tenantplexstorage.New(tf.Ctx, tf.TenantDB, cachetesthelpers.NewCacheConfig())
		tp, err := s.GetTenantPlex(tf.Ctx, tf.TenantID)
		assert.NoErr(t, err)

		externalIssuer := "https://www.okta.com"
		tp.PlexConfig.ExternalOIDCIssuers = []string{externalIssuer}
		err = s.SaveTenantPlex(tf.Ctx, tp)
		assert.NoErr(t, err)

		ets := externalTokenSource{t: t, issuer: externalIssuer}
		tsOpt := jsonclient.TokenSource(ets)
		extClient, err := idp.NewClient(tf.TenantURL, idp.JSONClient(tsOpt))
		assert.NoErr(t, err)
		extAuthzClient, err := authz.NewClient(tf.TenantURL, authz.JSONClient(tsOpt))
		assert.NoErr(t, err)

		email := "testuser@test.com"
		uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": email}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		columns, err := tf.IDPClient.ListColumns(tf.Ctx)
		assert.NoErr(t, err)

		// verify that a bunch of operations fail
		_, err = extClient.CreateUser(tf.Ctx, userstore.Record{})
		assert.NotNil(t, err)
		_, err = extClient.GetUser(tf.Ctx, uid)
		assert.NotNil(t, err)
		_, err = extClient.GetConsentedPurposesForUser(tf.Ctx, uid, nil)
		assert.NotNil(t, err)
		_, err = extClient.ListAccessPolicies(tf.Ctx, false)
		assert.NotNil(t, err)
		_, err = extClient.ListAccessPolicyTemplates(tf.Ctx, false)
		assert.NotNil(t, err)
		_, err = extClient.ListAccessors(tf.Ctx, false)
		assert.NotNil(t, err)
		_, err = extClient.ListMutators(tf.Ctx, false)
		assert.NotNil(t, err)
		_, err = extClient.ListPurposes(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extClient.ListTransformers(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extClient.CreateColumn(tf.Ctx, userstore.Column{
			Table:        "users",
			Name:         uniqueName("column"),
			DataType:     datatype.String,
			IsArray:      false,
			DefaultValue: "",
			IndexType:    userstore.ColumnIndexTypeIndexed,
			Constraints:  userstore.ColumnConstraints{},
		}, idp.IfNotExists())
		assert.NotNil(t, err)
		_, err = extClient.GetColumn(tf.Ctx, columns.Data[0].ID)
		assert.NotNil(t, err)
		_, err = extClient.UpdateColumn(tf.Ctx, columns.Data[0].ID, columns.Data[0])
		assert.NotNil(t, err)
		err = extClient.DeleteColumn(tf.Ctx, columns.Data[0].ID)
		assert.NotNil(t, err)
		_, err = extAuthzClient.ListObjects(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extAuthzClient.ListEdgeTypes(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extAuthzClient.ListEdges(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extAuthzClient.ListObjectTypes(tf.Ctx)
		assert.NotNil(t, err)
		_, err = extAuthzClient.CreateObject(tf.Ctx, uuid.Must(uuid.NewV4()), authz.UserObjectTypeID, "")
		assert.NotNil(t, err)
		_, err = extAuthzClient.GetObject(tf.Ctx, uid)
		assert.NotNil(t, err)
		err = extAuthzClient.DeleteObject(tf.Ctx, uid)
		assert.NotNil(t, err)

		// verify that ExecuteAccessor works
		_, err = extClient.ExecuteAccessor(tf.Ctx, constants.GetUserAccessorID, policy.ClientContext{}, []any{[]uuid.UUID{uid}})
		assert.NoErr(t, err)

		// verify that ExecuteMutator works
		mutator, err := tf.CreateMutator(
			uniqueName("mutator"),
			policy.AccessPolicyAllowAll.ID,
			[]string{"name"},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
		)
		assert.NoErr(t, err)

		_, err = extClient.ExecuteMutator(
			tf.Ctx,
			mutator.ID,
			policy.ClientContext{},
			[]any{uid},
			map[string]idp.ValueAndPurposes{
				"name": {
					Value:            idp.MutatorColumnCurrentValue,
					PurposeAdditions: []userstore.ResourceID{{Name: "operational"}},
				},
			},
		)
		assert.NoErr(t, err)

		// Create APT and AP that checks the issuer, and accessor that uses it
		aptName := uniqueName("access_policy_template")
		apt, err := tf.TokenizerClient.CreateAccessPolicyTemplate(
			tf.Ctx,
			policy.AccessPolicyTemplate{
				Name: aptName,
				Function: fmt.Sprintf(`function policy(context, params) {
				return context.server.claims.iss === params.issuer;
			} // %s`, aptName),
			},
			idp.IfNotExists(),
		)
		assert.NoErr(tf.T, err)
		ap, err := tf.IDPClient.CreateAccessPolicy(
			tf.Ctx,
			policy.AccessPolicy{
				Name:        uniqueName("access_policy"),
				Description: "Checks that issuer in token matches the expected issuer",
				Components: []policy.AccessPolicyComponent{
					{
						Template:           &userstore.ResourceID{ID: apt.ID},
						TemplateParameters: fmt.Sprintf(`{ "issuer": "%s" }`, externalIssuer),
					},
				},
				PolicyType: policy.PolicyTypeCompositeAnd,
			},
		)
		assert.NoErr(tf.T, err)
		accessor, err := tf.CreateLiveAccessor(
			uniqueName("accessor"),
			ap.ID,
			[]string{"email"},
			[]uuid.UUID{policy.TransformerPassthrough.ID},
			[]string{"operational"},
		)
		assert.NoErr(tf.T, err)

		// Verify that the accessor works and returns the user's email when using the externally issued token
		ret, err := extClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.True(tf.T, len(ret.Data) == 1)
		assert.Equal(tf.T, ret.Data[0], fmt.Sprintf("{\"email\":\"%s\"}", email))

		// Verify that the accessor works but returns no data when using a token with the UC token (since AP check fails)
		ret, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(tf.T, err)
		assert.True(tf.T, len(ret.Data) == 0)

		// Now create an accessor that tokenizes values and verify that resolving it using external token works
		accessor2, err := tf.IDPClient.CreateAccessor(tf.Ctx,
			userstore.Accessor{
				Name:               uniqueName("accessor"),
				DataLifeCycleState: userstore.DataLifeCycleStateLive,
				Columns: []userstore.ColumnOutputConfig{{
					Column:            userstore.ResourceID{Name: "email"},
					Transformer:       userstore.ResourceID{ID: policy.TransformerUUID.ID},
					TokenAccessPolicy: userstore.ResourceID{ID: ap.ID},
				}},
				AccessPolicy: userstore.ResourceID{ID: ap.ID},
				SelectorConfig: userstore.UserSelectorConfig{
					WhereClause: "{id} = ?",
				},
				Purposes: []userstore.ResourceID{{Name: "operational"}},
			})
		assert.NoErr(tf.T, err)
		ret, err = extClient.ExecuteAccessor(tf.Ctx, accessor2.ID, policy.ClientContext{}, []any{uid})
		assert.NoErr(t, err)
		assert.True(tf.T, len(ret.Data) == 1)
		var emailMap map[string]string
		assert.NoErr(tf.T, json.Unmarshal([]byte(ret.Data[0]), &emailMap))

		// Verify that the token can be resolved
		resolved, err := extClient.ResolveToken(tf.Ctx, emailMap["email"], policy.ClientContext{}, nil)
		assert.NoErr(t, err)
		assert.Equal(tf.T, resolved, email)
	})

	t.Run("test_user_cleanup", func(t *testing.T) {

		// create array column

		arrayColumn, err := tf.CreateColumn(
			uniqueName("column"),
			datatype.Integer,
			true,
			"",
			userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID},
			userstore.ResourceID{ID: policy.TransformerPassthrough.ID},
			userstore.ColumnIndexTypeIndexed,
		)
		assert.NoErr(t, err)

		// create column manager
		s := storage.NewFromTenantState(tf.Ctx, tf.TenantState)

		cm, err := storage.NewUserstoreColumnManager(tf.Ctx, s)
		assert.NoErr(tf.T, err)

		// create two test users

		uid1, err := tf.IDPClient.CreateUser(
			tf.Ctx,
			userstore.Record{
				"name":  "foo",
				"email": "foo@bar.org",
			},
			idp.OrganizationID(tf.Company.ID),
		)
		assert.NoErr(tf.T, err)

		uid2, err := tf.IDPClient.CreateUser(
			tf.Ctx,
			userstore.Record{
				"name":           "bar",
				"email":          "bar@bar.org",
				arrayColumn.Name: []int{1},
			},
			idp.OrganizationID(tf.Company.ID),
		)
		assert.NoErr(tf.T, err)

		// verify there are no cleanup entries

		pager, err := storage.NewUserCleanupCandidatePaginatorFromOptions()
		assert.NoErr(tf.T, err)
		us := storage.NewUserStorage(tf.Ctx, tf.TenantDB, "", tf.TenantID)
		cleanupCandidates, respFields, err := us.ListUserCleanupCandidatesPaginated(
			tf.Ctx,
			*pager,
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 0)
		assert.False(tf.T, respFields.HasNext)

		// verify call to cleanup users still succeeds if there are no candidates
		remaining, err := us.CleanupUsers(tf.Ctx, cm, 10, false)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, remaining, 0)

		// insert duplicate values in user_column_pre_delete_values

		ccv := storage.ColumnConsentedValue{
			Version:    1,
			ColumnName: "name",
			Value:      "foo",
			Ordering:   1,
			ConsentedPurposes: []storage.ConsentedPurpose{
				{
					Purpose:          constants.OperationalPurposeID,
					RetentionTimeout: userstore.GetRetentionTimeoutIndefinite(),
				},
			},
		}

		value, err := storage.NewUserColumnLiveValue(
			uid1,
			cm.GetColumnByID(column.NameColumnID),
			&ccv,
		)
		assert.NoErr(tf.T, err)

		sim, err := storage.NewSearchIndexManager(tf.Ctx, s)
		assert.NoErr(tf.T, err)

		err = us.InsertUserColumnLiveValues(tf.Ctx, cm, sim, nil, storage.UserColumnLiveValues{*value})
		assert.NoErr(tf.T, err)

		ccv = storage.ColumnConsentedValue{
			Version:    1,
			ColumnName: arrayColumn.Name,
			Value:      1,
			Ordering:   1,
			ConsentedPurposes: []storage.ConsentedPurpose{
				{
					Purpose:          constants.OperationalPurposeID,
					RetentionTimeout: userstore.GetRetentionTimeoutIndefinite(),
				},
			},
		}

		value, err = storage.NewUserColumnLiveValue(
			uid2,
			cm.GetColumnByID(arrayColumn.ID),
			&ccv,
		)
		assert.NoErr(tf.T, err)

		err = us.InsertUserColumnLiveValues(tf.Ctx, cm, sim, nil, storage.UserColumnLiveValues{*value})
		assert.NoErr(tf.T, err)

		// get users

		user1, err := tf.IDPClient.GetUser(tf.Ctx, uid1)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, user1.ID, uid1)
		assert.Equal(tf.T, user1.Profile["name"], "foo")
		assert.Equal(tf.T, user1.Profile["email"], "foo@bar.org")

		user2, err := tf.IDPClient.GetUser(tf.Ctx, uid2)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, user2.ID, uid2)
		assert.Equal(tf.T, user2.Profile["name"], "bar")
		assert.Equal(tf.T, user2.Profile["email"], "bar@bar.org")
		assert.Equal(tf.T, user2.Profile[arrayColumn.Name], "[1]")

		// verify that there are now cleanup entries for users

		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid1)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 1)
		assert.Equal(tf.T, cleanupCandidates[0].UserID, uid1)
		assert.Equal(tf.T, cleanupCandidates[0].CleanupReason, storage.UserCleanupReasonDuplicateValue)

		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid2)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 1)
		assert.Equal(tf.T, cleanupCandidates[0].UserID, uid2)
		assert.Equal(tf.T, cleanupCandidates[0].CleanupReason, storage.UserCleanupReasonDuplicateValue)

		// cleanup users
		remaining, err = us.CleanupUsers(tf.Ctx, cm, 10, false)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, remaining, 0)

		// verify there are no cleanup entries for users
		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid1)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 0)

		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid2)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 0)

		// get users

		user1, err = tf.IDPClient.GetUser(tf.Ctx, uid1)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, user1.ID, uid1)
		assert.Equal(tf.T, user1.Profile["name"], "foo")
		assert.Equal(tf.T, user1.Profile["email"], "foo@bar.org")

		user2, err = tf.IDPClient.GetUser(tf.Ctx, uid2)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, user2.ID, uid2)
		assert.Equal(tf.T, user2.Profile["name"], "bar")
		assert.Equal(tf.T, user2.Profile["email"], "bar@bar.org")
		assert.Equal(tf.T, user2.Profile[arrayColumn.Name], "[1]")

		// verify there are still no cleanup entries for users

		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid1)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 0)

		cleanupCandidates, err = us.ListUserCleanupCandidatesForUserID(tf.Ctx, uid2)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(cleanupCandidates), 0)
	})

	t.Run("test_database_methods", func(t *testing.T) {
		db := userstore.SQLShimDatabase{
			ID:       uuid.Must(uuid.NewV4()),
			Name:     uniqueName("database"),
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "password",
		}
		dbRet, err := tf.IDPClient.CreateDatabase(tf.Ctx, db)
		assert.NoErr(t, err)
		assert.True(t, dbRet.EqualsIgnoringNilIDSchemasAndPassword(db))

		db2 := userstore.SQLShimDatabase{
			ID:       uuid.Must(uuid.NewV4()),
			Name:     uniqueName("database2"),
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "admin",
			Password: "password",
		}
		_, err = tf.IDPClient.CreateDatabase(tf.Ctx, db2)
		assert.NoErr(t, err)

		dbRet, err = tf.IDPClient.GetDatabase(tf.Ctx, db.ID)
		assert.NoErr(t, err)
		assert.True(t, dbRet.EqualsIgnoringNilIDSchemasAndPassword(db))

		name := uniqueName("database1new")
		db.Name = name
		dbRet, err = tf.IDPClient.UpdateDatabase(tf.Ctx, db.ID, db)
		assert.NoErr(t, err)
		assert.True(t, dbRet.EqualsIgnoringNilIDSchemasAndPassword(db))

		dbs, err := tf.IDPClient.ListDatabases(tf.Ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(dbs.Data), 2)
		found := false
		for _, database := range dbs.Data {
			if database.Name == name {
				found = true
				assert.True(t, database.EqualsIgnoringNilIDSchemasAndPassword(db))
			}
		}
		assert.True(t, found)

		err = tf.IDPClient.DeleteDatabase(tf.Ctx, db.ID)
		assert.NoErr(t, err)

		dbs, err = tf.IDPClient.ListDatabases(tf.Ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(dbs.Data), 1)
	})
}

// NOTE: the following tests must use their own test fixtures:

func TestExternalOIDCIssuers(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	issuers, err := tf.IDPClient.GetExternalOIDCIssuers(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(issuers), 0)

	issuers = []string{"https://www.okta.com", "https://www.auth0.com"}
	err = tf.IDPClient.UpdateExternalOIDCIssuers(tf.Ctx, issuers)
	assert.NoErr(t, err)

	issuers, err = tf.IDPClient.GetExternalOIDCIssuers(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, issuers, []string{"https://www.okta.com", "https://www.auth0.com"})
}

func TestCreateUpdateAndDeleteUser(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	// ensure user creation, updating, and deletion succeeds with
	// a mixture of full and partial update columns

	colName := uniqueName("column")
	_, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:     "users",
			Name:      colName,
			DataType:  datatype.String,
			IndexType: userstore.ColumnIndexTypeIndexed,
			IsArray:   true,
			Constraints: userstore.ColumnConstraints{
				PartialUpdates: true,
				UniqueRequired: true,
			},
		},
		idp.IfNotExists(),
	)
	assert.NoErr(t, err)

	// create user with a full and partial update profile value

	profile := userstore.Record{
		"email": "TestCrudUser@nowhere.org",
		colName: []string{"foo", "bar"},
	}

	uid, err := tf.IDPClient.CreateUser(
		tf.Ctx,
		profile,
		idp.OrganizationID(tf.Company.ID),
	)
	assert.NoErr(t, err)

	// update user, ensuring that unspecified columns and previously
	// added partial columns stay the same

	profile = userstore.Record{
		"name":  "Bilbo",
		colName: []string{"baz"},
	}

	resp, err := tf.IDPClient.UpdateUser(
		tf.Ctx,
		uid,
		idp.UpdateUserRequest{Profile: profile},
	)
	assert.NoErr(t, err)
	assert.Equal(t, resp.ID, uid)
	assert.Equal(t, resp.Profile["email"], "TestCrudUser@nowhere.org")
	assert.Equal(t, resp.Profile["name"], "Bilbo")
	assert.Equal(t, resp.Profile[colName], "[\"foo\",\"bar\",\"baz\"]")

	// update user, augmenting returned profile, and perform
	// same validation

	profile = resp.Profile
	profile["email"] = "NewNameForTestCrudUser@nowhere.org"
	profile[colName] = []string{"biz"}

	resp, err = tf.IDPClient.UpdateUser(
		tf.Ctx,
		uid,
		idp.UpdateUserRequest{Profile: profile},
	)
	assert.NoErr(t, err)
	assert.Equal(t, resp.ID, uid)
	assert.Equal(t, resp.Profile["email"], "NewNameForTestCrudUser@nowhere.org")
	assert.Equal(t, resp.Profile["name"], "Bilbo")
	assert.Equal(t, resp.Profile[colName], "[\"foo\",\"bar\",\"baz\",\"biz\"]")

	// delete the user

	assert.NoErr(t, tf.IDPClient.DeleteUser(tf.Ctx, uid))

	// create and then update user with no specified partial update value

	profile = userstore.Record{"email": "AnotherUser@nowhere.org"}
	uid, err = tf.IDPClient.CreateUser(
		tf.Ctx,
		profile,
		idp.OrganizationID(tf.Company.ID),
	)
	assert.NoErr(t, err)

	profile = userstore.Record{"name": "Frodo"}
	resp, err = tf.IDPClient.UpdateUser(
		tf.Ctx,
		uid,
		idp.UpdateUserRequest{Profile: profile},
	)
	assert.NoErr(t, err)
	assert.Equal(t, resp.ID, uid)
	assert.Equal(t, resp.Profile["email"], "AnotherUser@nowhere.org")
	assert.Equal(t, resp.Profile["name"], "Frodo")
	assert.IsNil(t, resp.Profile[colName])
}

func TestAccessPolicyInAccessorAndMutator(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	uid, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{"email": "joe@bigcorp.com"}, idp.OrganizationID(tf.Company.ID))
	assert.NoErr(t, err)

	accessPolicy := tf.CreateValidAccessPolicy(uniqueName("access_policy"), "apikey1")

	accessor, err := tf.CreateLiveAccessor(uniqueName("accessor"),
		accessPolicy.ID,
		[]string{"email"},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
		[]string{"operational"})
	assert.NoErr(t, err)

	output, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{"api_key": "apikey1"}, []any{uid})
	assert.NoErr(t, err)
	assert.Equal(t, len(output.Data), 1)

	output, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{"api_key": "apikey2"}, []any{uid})
	assert.NoErr(t, err)
	assert.Equal(t, len(output.Data), 0)

	mutator, err := tf.CreateMutator(uniqueName("mutator"),
		accessPolicy.ID,
		[]string{"email"},
		[]uuid.UUID{policy.TransformerPassthrough.ID})
	assert.NoErr(t, err)

	resp, err := tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{"api_key": "apikey1"}, []any{uid}, map[string]idp.ValueAndPurposes{
		"email": {Value: "new_email@bigcorp.com", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}})
	assert.NoErr(t, err)
	assert.Equal(t, len(resp.UserIDs), 1)

	resp, err = tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{"api_key": "apikey2"}, []any{uid}, map[string]idp.ValueAndPurposes{
		"email": {Value: "new_email2@bigcorp.com", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}})
	assert.NoErr(t, err)
	assert.Equal(t, len(resp.UserIDs), 0)

	featureflags.EnableFlagsForTest(featureflags.GlobalAccessPolicies)

	accessPolicy, err = tf.IDPClient.GetAccessPolicy(tf.Ctx, userstore.ResourceID{ID: policy.AccessPolicyGlobalAccessorID})
	assert.NoErr(t, err)
	accessPolicy.Components[0].Template = &userstore.ResourceID{ID: policy.AccessPolicyTemplateDenyAll.ID}
	_, err = tf.IDPClient.UpdateAccessPolicy(tf.Ctx, *accessPolicy)
	assert.NoErr(t, err)

	output, err = tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{"api_key": "apikey1"}, []any{uid})
	assert.NoErr(t, err)
	assert.Equal(t, len(output.Data), 0)

	accessPolicy, err = tf.IDPClient.GetAccessPolicy(tf.Ctx, userstore.ResourceID{ID: policy.AccessPolicyGlobalMutatorID})
	assert.NoErr(t, err)
	accessPolicy.Components[0].Template = &userstore.ResourceID{ID: policy.AccessPolicyTemplateDenyAll.ID}
	_, err = tf.IDPClient.UpdateAccessPolicy(tf.Ctx, *accessPolicy)
	assert.NoErr(t, err)

	resp, err = tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{"api_key": "apikey1"}, []any{uid}, map[string]idp.ValueAndPurposes{
		"email": {Value: "new_email3@bigcorp.com", PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}})
	assert.NoErr(t, err)
	assert.Equal(t, len(resp.UserIDs), 0)
}

func TestPythonCodeGen(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)
	testhelpers.RunScript(tf.Ctx, t, "tools/ensure-python-sdk-venv.sh")
	pythonSDK, err := tf.IDPClient.DownloadPythonSDK(tf.Ctx)
	assert.NoErr(t, err)
	fl, err := os.CreateTemp("", "python_codegen.py")
	assert.NoErr(t, err)
	defer os.Remove(fl.Name())
	_, err = fl.WriteString(pythonSDK)
	assert.NoErr(t, err)
	assert.NoErr(t, fl.Close())
	uclog.Infof(tf.Ctx, "Python SDK written to %s", fl.Name())
	testhelpers.RunCommand(tf.Ctx, t, "public-repos/sdk-python/.venv/bin/python", fl.Name())
	testhelpers.RunCommand(tf.Ctx, t, "public-repos/sdk-python/.venv/bin/ruff", "check", "--ignore", "E501", fl.Name())
}

func TestCodegenSDKDownload(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	tf.CreateValidAccessor("test_codegen_sdk_download_accessor")
	tf.CreateValidMutator("test_codegen_sdk_download_mutator")
	_, err := tf.IDPClient.CreatePurpose(tf.Ctx, userstore.Purpose{
		Name:        "test_codegen_sdk_download_purpose_name",
		Description: "purpose_description",
	})
	assert.NoErr(t, err)

	goSDK, err := tf.IDPClient.DownloadGolangSDK(tf.Ctx)
	assert.NoErr(t, err)
	assert.Contains(t, goSDK, "Gettest_codegen_sdk_download_accessorObject")
	assert.Contains(t, goSDK, "UpdateUserForTestCodegenSdkDownloadPurposeNamePurpose")

	pySDK, err := tf.IDPClient.DownloadPythonSDK(tf.Ctx)
	assert.NoErr(t, err)
	assert.Contains(t, pySDK, "Gettest_codegen_sdk_download_accessorObject")
	assert.Contains(t, pySDK, "UpdateUserForTestCodegenSdkDownloadPurposeNamePurpose")
}

func TestCreateUserWithMutator(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	colName := uniqueName("column")
	tf.CreateValidColumn(colName, datatype.String, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeIndexed)

	mutator, err := tf.CreateMutator(uniqueName("mutator"),
		policy.AccessPolicyAllowAll.ID,
		[]string{colName},
		[]uuid.UUID{policy.TransformerPassthrough.ID})
	assert.NoErr(t, err)

	// test invalid region
	_, err = tf.IDPClient.CreateUserWithMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, map[string]idp.ValueAndPurposes{
		colName: {Value: "new value", PurposeAdditions: []userstore.ResourceID{{Name: "marketing"}}}}, idp.DataRegion("asdf"), idp.OrganizationID(tf.Company.ID))
	assert.NotNil(t, err)

	userID := uuid.Must(uuid.NewV4())
	uid, err := tf.IDPClient.CreateUserWithMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, map[string]idp.ValueAndPurposes{
		colName: {Value: "new value", PurposeAdditions: []userstore.ResourceID{{Name: "marketing"}}}}, idp.UserID(userID), idp.OrganizationID(tf.Company.ID))
	assert.Equal(t, uid, userID)
	assert.NoErr(t, err)

	user, err := tf.IDPClient.GetUser(tf.Ctx, uid)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile[colName], nil) // blank since we didn't specify operational as a purpose

	accessor, err := tf.CreateLiveAccessor(uniqueName("accessor"),
		policy.AccessPolicyAllowAll.ID,
		[]string{colName},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
		[]string{"marketing"},
	)
	assert.NoErr(t, err)

	resp, err := tf.IDPClient.ExecuteAccessor(tf.Ctx, accessor.ID, policy.ClientContext{}, []any{uid})
	assert.NoErr(t, err)

	assert.Equal(t, len(resp.Data), 1)
	assert.Equal(t, resp.Data[0], fmt.Sprintf("{\"%s\":\"new value\"}", colName))

	anotherColName := uniqueName("column")
	tf.CreateValidColumn(anotherColName, datatype.Integer, false, "", userstore.ResourceID{ID: policy.AccessPolicyAllowAll.ID}, userstore.ResourceID{ID: policy.TransformerPassthrough.ID}, userstore.ColumnIndexTypeUnique)

	mutator = getUpdatableMutator(mutator)
	mutator.Columns = append(mutator.Columns, userstore.ColumnInputConfig{
		Column:     userstore.ResourceID{Name: anotherColName},
		Normalizer: userstore.ResourceID{ID: policy.TransformerPassthrough.ID}},
	)
	mutator, err = tf.IDPClient.UpdateMutator(tf.Ctx, mutator.ID, *mutator)
	assert.NoErr(t, err)

	_, err = tf.IDPClient.ExecuteMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, []any{uid}, map[string]idp.ValueAndPurposes{
		colName:        {Value: idp.MutatorColumnCurrentValue, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
		anotherColName: {Value: 123, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}})
	assert.NoErr(t, err)

	user, err = tf.IDPClient.GetUser(tf.Ctx, uid)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile[colName], "new value") // should be retrieved now that we added operational purpose
	assert.Equal(t, user.Profile[anotherColName], "123")

	// Try to create a second user with the same value for test_column2 - should fail since it's a unique column
	_, err = tf.IDPClient.CreateUserWithMutator(tf.Ctx, mutator.ID, policy.ClientContext{}, map[string]idp.ValueAndPurposes{
		colName:        {Value: idp.MutatorColumnCurrentValue, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}},
		anotherColName: {Value: 123, PurposeAdditions: []userstore.ResourceID{{Name: "operational"}}}}, idp.OrganizationID(tf.Company.ID))
	assert.NotNil(t, err)

	// Check that the user is deleted
	time.Sleep(1 * time.Second) // need to sleep so that the user is deleted
	var count []int
	assert.IsNil(t, tf.TenantDB.SelectContext(tf.Ctx, "TestUserDeletion", &count, "select count(*) from users where deleted != '0001-01-01 00:00:00'; /* lint-deleted */"), assert.Must())
	assert.Equal(t, count[0], 1)
}

func TestListDefaultEntities(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)

	foundDataTypes, err := tf.IDPClient.ListDataTypes(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultDataTypes()), len(foundDataTypes.Data))

	foundColumns, err := tf.IDPClient.ListColumns(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultColumns()), len(foundColumns.Data))

	foundPurposes, err := tf.IDPClient.ListPurposes(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultPurposes()), len(foundPurposes.Data))

	foundAccessPolicyTemplates, err := tf.TokenizerClient.ListAccessPolicyTemplates(tf.Ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultAccessPolicyTemplates()), len(foundAccessPolicyTemplates.Data))

	foundAccessPolicies, err := tf.TokenizerClient.ListAccessPolicies(tf.Ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultAccessPolicies()), len(foundAccessPolicies.Data))

	foundTransformers, err := tf.TokenizerClient.ListTransformers(tf.Ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultTransformers()), len(foundTransformers.Data))

	foundAccessors, err := tf.IDPClient.ListAccessors(tf.Ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultAccessors()), len(foundAccessors.Data))

	foundMutators, err := tf.IDPClient.ListMutators(tf.Ctx, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(defaults.GetDefaultMutators()), len(foundMutators.Data))
}

// Users are treated implicitly as AuthZ objects, which makes enumeration with pagination
// somewhat more complicated, hence this specific test.
func TestUsersAndObjectsPaginated(t *testing.T) {
	t.Parallel()
	tf := idptesthelpers.NewTestFixture(t)
	objectTypeName := uniqueName("object_type")
	ot1, err := tf.AuthzClient.CreateObjectType(tf.Ctx, uuid.Must(uuid.NewV4()), objectTypeName)
	assert.NoErr(t, err)

	const pageSize = 5
	const n = pageSize*4 + 2
	sortedIDs := make([]uuid.UUID, n)
	userIDs := map[uuid.UUID]bool{}
	for i := range n / 2 {
		userID, err := tf.IDPClient.CreateUser(tf.Ctx, userstore.Record{}, idp.OrganizationID(tf.Company.ID))
		assert.NoErr(t, err)
		sortedIDs[i] = userID
		userIDs[userID] = true
	}
	for i := n / 2; i < n; i++ {
		objID := uuid.Must(uuid.NewV4())
		sortedIDs[i] = objID
		alias := fmt.Sprintf("obj_%v", objID)
		_, err = tf.AuthzClient.CreateObject(tf.Ctx, objID, ot1.ID, alias)
		assert.NoErr(t, err)
	}
	sort.Slice(sortedIDs, func(i, j int) bool { return sortedIDs[i].String() < sortedIDs[j].String() })

	var origIdx, userCount, objCount int
	cursor := pagination.CursorBegin
	for {
		resp, err := tf.AuthzClient.ListObjects(tf.Ctx, authz.Pagination(pagination.Limit(pageSize), pagination.StartingAfter(cursor)))
		assert.NoErr(t, err)

		objs := resp.Data
		for i := range objs {
			if objs[i].TypeID == idpAuthz.PolicyTransformerTypeID ||
				objs[i].TypeID == idpAuthz.PolicyAccessTypeID ||
				objs[i].TypeID == idpAuthz.PolicyAccessTemplateTypeID ||
				objs[i].TypeID == idpAuthz.PoliciesObjectID ||
				objs[i].TypeID == idpAuthz.PoliciesTypeID ||
				objs[i].TypeID == authz.GroupObjectTypeID ||
				objs[i].TypeID == authz.LoginAppObjectTypeID ||
				objs[i].TypeID == authz.UserObjectTypeID && userIDs[objs[i].ID] == false {
				continue
			}
			assert.Equal(t, objs[i].ID, sortedIDs[origIdx])
			assert.True(t,
				objs[i].TypeID == authz.UserObjectTypeID || objs[i].TypeID == ot1.ID,
				assert.Must(),
				assert.Errorf("object has unexpected type ID %v, expected either %v (user) or %v (%s)",
					objs[i].TypeID,
					authz.UserObjectTypeID,
					ot1.ID,
					objectTypeName,
				),
			)
			if objs[i].TypeID == authz.UserObjectTypeID {
				userCount++
			} else if objs[i].TypeID == ot1.ID {
				objCount++
			}
			origIdx++
		}

		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}
	assert.Equal(t, userCount, n/2)
	assert.Equal(t, objCount, n/2)
}
