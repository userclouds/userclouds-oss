package idp_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
)

type variationMaker struct {
	baseValues []string
	values     []string
}

func newVariationMaker(baseValues ...string) *variationMaker {
	return &variationMaker{
		baseValues: baseValues,
		values:     baseValues,
	}
}

func (vm *variationMaker) addValue(value string) *variationMaker {
	vm.values = append(vm.values, value)
	return vm
}

func (vm *variationMaker) addVariation(suffix string) *variationMaker {
	for _, baseValue := range vm.baseValues {
		vm.values = append(vm.values, fmt.Sprintf("%s::%s", baseValue, suffix))
	}
	return vm
}

func (vm variationMaker) build() []string {
	return vm.values
}

type valueVariations struct {
	bools      []string
	ints       []string
	strings    []string
	timestamps []string
	uuids      []string
}

var subFieldValueVariations valueVariations

type subFieldTestFixture struct {
	idptesthelpers.TestFixture
	compositeColumn   *userstore.Column
	compositeAccessor *userstore.Accessor
}

func newSubFieldTestFixture(t *testing.T) subFieldTestFixture {
	t.Helper()

	tf := subFieldTestFixture{
		TestFixture: *idptesthelpers.NewTestFixture(t),
	}

	compositeDataType, err := tf.IDPClient.CreateDataType(
		tf.Ctx,
		userstore.ColumnDataType{
			Name:        "composite",
			Description: "test composite data type",
			CompositeAttributes: userstore.CompositeAttributes{
				Fields: []userstore.CompositeField{
					{
						DataType: datatype.Boolean,
						Name:     "Boolean",
					},
					{
						DataType: datatype.Integer,
						Name:     "Integer",
					},
					{
						DataType: datatype.String,
						Name:     "String",
					},
					{
						DataType: datatype.Timestamp,
						Name:     "Timestamp",
					},
					{
						DataType: datatype.UUID,
						Name:     "UUID",
					},
				},
			},
		},
		idp.IfNotExists(),
	)
	assert.NoErr(t, err)
	assert.NotNil(t, compositeDataType)

	compositeColumn, err := tf.IDPClient.CreateColumn(
		tf.Ctx,
		userstore.Column{
			Table:     "users",
			Name:      "composite",
			DataType:  userstore.ResourceID{ID: compositeDataType.ID},
			IsArray:   true,
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
		idp.IfNotExists(),
	)
	assert.NoErr(t, err)
	assert.NotNil(t, compositeColumn)

	compositeAccessor, err := tf.CreateLiveAccessor(
		"CompositeAccessor",
		policy.AccessPolicyAllowAll.ID,
		[]string{compositeColumn.Name},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
		[]string{"operational"},
	)
	assert.NoErr(t, err)
	assert.NotNil(t, compositeAccessor)

	tf.compositeColumn = compositeColumn
	tf.compositeAccessor = compositeAccessor
	return tf
}

func (tf *subFieldTestFixture) testConstants(subFieldName string, constants []string) {
	tf.T.Helper()

	for _, constant := range constants {
		tf.compositeAccessor.SelectorConfig = userstore.UserSelectorConfig{
			WhereClause: fmt.Sprintf("{composite}->>'%s' = %s", subFieldName, constant),
		}
		compositeAccessor, err := tf.IDPClient.UpdateAccessor(tf.Ctx, tf.compositeAccessor.ID, *tf.compositeAccessor)
		assert.NoErr(tf.T, err)
		assert.NotNil(tf.T, compositeAccessor)
		tf.compositeAccessor = compositeAccessor

		resp, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.compositeAccessor.ID,
			policy.ClientContext{},
			[]any{},
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.Data), 0)
	}
}

func (tf *subFieldTestFixture) testVariables(subFieldName string, values ...any) {
	tf.T.Helper()

	tf.compositeAccessor.SelectorConfig = userstore.UserSelectorConfig{
		WhereClause: fmt.Sprintf("{composite}->>'%s' = ?", subFieldName),
	}
	compositeAccessor, err := tf.IDPClient.UpdateAccessor(tf.Ctx, tf.compositeAccessor.ID, *tf.compositeAccessor)
	assert.NoErr(tf.T, err)
	assert.NotNil(tf.T, compositeAccessor)
	tf.compositeAccessor = compositeAccessor

	for _, value := range values {
		resp, err := tf.IDPClient.ExecuteAccessor(
			tf.Ctx,
			tf.compositeAccessor.ID,
			policy.ClientContext{},
			[]any{value},
		)
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.Data), 0)
	}
}

func TestSubFieldSelectors(t *testing.T) {
	t.Parallel()

	tf := newSubFieldTestFixture(t)

	tf.testConstants(`boolean`, subFieldValueVariations.bools)
	tf.testVariables(`boolean`, false, true)
	tf.testConstants(`integer`, subFieldValueVariations.ints)
	tf.testVariables(`integer`, 42, -42, +42)
	tf.testConstants(`string`, subFieldValueVariations.strings)
	tf.testVariables(`string`, `foo`, `fo''o`, `fo'o`)
	tf.testConstants(`timestamp`, subFieldValueVariations.timestamps)
	tf.testVariables(`timestamp`, time.Now().UTC())
	tf.testConstants(`uuid`, subFieldValueVariations.uuids)
	tf.testVariables(`uuid`, uuid.Nil, uuid.Must(uuid.NewV4()))
}

func init() {
	subFieldValueVariations = valueVariations{
		bools: newVariationMaker(
			`false`,
			`'false'`,
			`'fals'`,
			`'fal'`,
			`'fa'`,
			`'f'`,
			`FALSE`,
			`'FALSE'`,
			`'FALS'`,
			`'FAL'`,
			`'FA'`,
			`'F'`,
			`true`,
			`'true'`,
			`'tru'`,
			`'tr'`,
			`'t'`,
			`TRUE`,
			`'TRUE'`,
			`'TRU'`,
			`'TR'`,
			`'T'`,
			`'off'`,
			`'of'`,
			`'OFF'`,
			`'OF'`,
			`'on'`,
			`'ON'`,
			`'no'`,
			`'n'`,
			`'NO'`,
			`'N'`,
			`'yes'`,
			`'ye'`,
			`'y'`,
			`'YES'`,
			`'YE'`,
			`'Y'`,
		).addVariation(`BOOL`).
			addVariation(`BOOLEAN`).
			addValue(`0::BOOL`).
			addValue(`'0'::BOOL`).
			addValue(`0::BOOLEAN`).
			addValue(`'0'::BOOLEAN`).
			addValue(`1::BOOL`).
			addValue(`'1'::BOOL`).
			addValue(`1::BOOLEAN`).
			addValue(`'1'::BOOLEAN`).
			build(),
		ints: newVariationMaker(
			`42`,
			`'42'`,
			`+42`,
			`'+42'`,
			`-42`,
			`'-42'`,
		).addVariation(`INT`).addVariation(`INTEGER`).build(),
		strings: newVariationMaker(
			`'foo'`,
			`'%foo%'`,
			`'fo''o'`,
		).addVariation(`VARCHAR`).build(),
		timestamps: newVariationMaker(
			`'2024-01-01'`,
			`'2024-01-01 00:00:00'`,
		).addVariation(`TIMESTAMP`).build(),
		uuids: newVariationMaker(`'01234456-789A-BCDE-Fabc-def000000000'`).addVariation(`UUID`).build(),
	}
}
