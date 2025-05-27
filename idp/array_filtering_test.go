package idp_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
)

func TestArrayFiltering(t *testing.T) {
	t.Parallel()

	tf := idptesthelpers.NewTestFixture(t)

	// create multi-value composite data type
	addressDataType, err := tf.IDPClient.CreateDataType(
		tf.Ctx,
		userstore.ColumnDataType{
			Name:        "address",
			Description: "test address data type",
			CompositeAttributes: userstore.CompositeAttributes{
				IncludeID: true,
				Fields: []userstore.CompositeField{
					{
						DataType:            datatype.Boolean,
						IgnoreForUniqueness: true,
						Name:                "Billing_Address",
					},
					{
						DataType:            datatype.Boolean,
						IgnoreForUniqueness: true,
						Name:                "Shipping_Address",
					},
					{
						DataType: datatype.String,
						Name:     "Address_Line_1",
					},
					{
						DataType: datatype.String,
						Name:     "City",
					},
					{
						DataType: datatype.String,
						Name:     "State",
					},
					{
						DataType: datatype.String,
						Name:     "Country",
					},
					{
						DataType: datatype.String,
						Name:     "Zip_Code",
					},
				},
			},
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, addressDataType)

	// create multi-value composite column

	addresses, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:     "users",
			Name:      "addresses",
			DataType:  userstore.ResourceID{ID: addressDataType.ID},
			IsArray:   true,
			IndexType: userstore.ColumnIndexTypeIndexed,
			Constraints: userstore.ColumnConstraints{
				ImmutableRequired: true,
				UniqueIDRequired:  true,
				UniqueRequired:    true,
			},
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, addresses)

	// create transformer for string that returns the same string

	transformerName := "name"
	nameTransformer, err := tf.TokenizerClient.CreateTransformer(
		tf.Ctx,
		policy.Transformer{
			Name:           transformerName,
			Description:    "just return the name unchanged",
			InputDataType:  datatype.String,
			OutputDataType: datatype.String,
			TransformType:  policy.TransformTypeTransform,
			Function: fmt.Sprintf(
				`
function transform(data, params) {
	return data;
} // %s
`,
				transformerName,
			),
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, nameTransformer)

	// create transformer for address that filters out addresses that are
	// not marked shipping_address

	transformerName = "shipping_address"
	shippingAddressTransformer, err := tf.TokenizerClient.CreateTransformer(
		tf.Ctx,
		policy.Transformer{
			Name:           transformerName,
			Description:    "filter out if address is not a shipping address",
			InputDataType:  userstore.ResourceID{ID: addressDataType.ID},
			OutputDataType: userstore.ResourceID{ID: addressDataType.ID},
			TransformType:  policy.TransformTypeTransform,
			Function: fmt.Sprintf(
				`
function transform(data, params) {
	if (data.shipping_address === true) {
		return data;
	}
	return "";
} // %s
`,
				transformerName,
			),
		},
		idp.IfNotExists(),
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, shippingAddressTransformer)

	// create accessor that only returns shipping addresses

	shippingAddressesAccessor, err := tf.CreateAccessorWithWhereClause(
		"ShippingAddressesAccessor",
		userstore.DataLifeCycleStateLive,
		[]string{
			"id",
			"name",
			addresses.Name,
		},
		[]uuid.UUID{
			uuid.Nil,
			uuid.Nil,
			uuid.Nil,
		},
		[]uuid.UUID{
			policy.TransformerPassthrough.ID,
			nameTransformer.ID,
			shippingAddressTransformer.ID,
		},
		[]string{"operational"},
		"{addresses}->>'shipping_address' = 'yes'::BOOL",
		policy.AccessPolicyAllowAll.ID,
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, shippingAddressesAccessor)

	// create accessor that returns all addresses

	allAddressesAccessor, err := tf.CreateAccessorWithWhereClause(
		"AllAddressesAccessor",
		userstore.DataLifeCycleStateLive,
		[]string{
			"id",
			"name",
			addresses.Name,
		},
		[]uuid.UUID{
			uuid.Nil,
			uuid.Nil,
			uuid.Nil,
		},
		[]uuid.UUID{
			policy.TransformerPassthrough.ID,
			policy.TransformerPassthrough.ID,
			policy.TransformerPassthrough.ID,
		},
		[]string{"operational"},
		"{id} = ?",
		policy.AccessPolicyAllowAll.ID,
	)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, allAddressesAccessor)

	// generate SDK

	golangSDK, err := tf.IDPClient.DownloadGolangSDK(tf.Ctx)
	assert.NoErr(tf.T, err)
	assert.NotEqual(tf.T, golangSDK, "")

	pythonSDK, err := tf.IDPClient.DownloadPythonSDK(tf.Ctx)
	assert.NoErr(tf.T, err)
	assert.NotEqual(tf.T, pythonSDK, "")

	// create a few users with different address values

	user1, err := tf.IDPClient.CreateUser(
		tf.Ctx,
		userstore.Record{
			"email": "user1@foo.org",
			"name":  "user1",
			"addresses": []userstore.CompositeValue{
				{
					"billing_address": true,
					"address_line_1":  "101 Congress Avenue",
					"city":            "Austin",
					"state":           "TX",
					"zip_code":        "78704",
				},
				{
					"shipping_address": true,
					"address_line_1":   "123 Main Street",
					"city":             "Austin",
					"state":            "TX",
					"zip_code":         "78704",
				},
			},
		},
		idp.OrganizationID(tf.Company.ID),
	)
	assert.NoErr(tf.T, err)
	assert.False(tf.T, user1.IsNil())

	user2, err := tf.IDPClient.CreateUser(
		tf.Ctx,
		userstore.Record{
			"email": "user2@foo.org",
			"name":  "user2",
			"addresses": []userstore.CompositeValue{
				{
					"address_line_1": "101 Congress Avenue",
					"city":           "Austin",
					"state":          "TX",
					"zip_code":       "78704",
				},
				{
					"address_line_1": "123 Main Street",
					"city":           "Austin",
					"state":          "TX",
					"zip_code":       "78704",
				},
			},
		},
		idp.OrganizationID(tf.Company.ID),
	)
	assert.NoErr(tf.T, err)
	assert.False(tf.T, user2.IsNil())

	// use accessor to get all values and confirm response

	resp, err := tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		allAddressesAccessor.ID,
		policy.ClientContext{},
		[]any{user1},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.Data), 1)
	profile := map[string]string{}
	assert.NoErr(tf.T, json.Unmarshal([]byte(resp.Data[0]), &profile))
	assert.Equal(tf.T, profile["id"], user1.String())
	addressValues := []userstore.CompositeValue{}
	assert.NoErr(tf.T, json.Unmarshal([]byte(profile["addresses"]), &addressValues))
	assert.Equal(tf.T, len(addressValues), 2)

	resp, err = tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		allAddressesAccessor.ID,
		policy.ClientContext{},
		[]any{user2},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.Data), 1)
	assert.NoErr(tf.T, json.Unmarshal([]byte(resp.Data[0]), &profile))
	assert.Equal(tf.T, profile["id"], user2.String())
	assert.NoErr(tf.T, json.Unmarshal([]byte(profile["addresses"]), &addressValues))
	assert.Equal(tf.T, len(addressValues), 2)

	// use filtering accessor and confirm response

	resp, err = tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		shippingAddressesAccessor.ID,
		policy.ClientContext{},
		[]any{},
	)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resp.Data), 1)
	assert.NoErr(tf.T, json.Unmarshal([]byte(resp.Data[0]), &profile))
	assert.Equal(tf.T, profile["id"], user1.String())
	assert.NoErr(tf.T, json.Unmarshal([]byte(profile["addresses"]), &addressValues))
	assert.Equal(tf.T, len(addressValues), 1)
	assert.Equal(tf.T, addressValues[0]["shipping_address"], true)
	assert.Equal(tf.T, addressValues[0]["address_line_1"], "123 Main Street")
}
