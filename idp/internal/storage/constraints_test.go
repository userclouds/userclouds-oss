package storage_test

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
)

type constraintsTestFixture struct {
	userstoreTestFixture
}

func newConstraintsTestFixture(t *testing.T) constraintsTestFixture {
	return constraintsTestFixture{
		userstoreTestFixture: newUserstoreTestFixture(t),
	}
}

func (tf constraintsTestFixture) badField(dt userstore.ResourceID, name string) {
	tf.t.Helper()

	cf := userstore.CompositeField{DataType: dt, Name: name}
	f := column.NewCompositeFieldFromClient(cf)
	assert.NotNil(tf.t, f.Validate())
}

func (tf constraintsTestFixture) goodField(
	dt userstore.ResourceID,
	name string,
	camelCaseName string,
	structName string,
) {
	tf.t.Helper()

	cf := userstore.CompositeField{DataType: dt, Name: name}
	assert.NoErr(tf.t, cf.Validate())
	f := column.NewCompositeFieldFromClient(cf)
	assert.NoErr(tf.t, f.Validate())
	assert.Equal(tf.t, f.DataTypeID, dt.ID)
	assert.Equal(tf.t, f.Name, name)
	assert.Equal(tf.t, f.CamelCaseName, camelCaseName)
	assert.Equal(tf.t, f.StructName, structName)

	cf = f.ToClient()
	assert.NoErr(tf.t, cf.Validate())
	assert.True(tf.t, cf.DataType.EquivalentTo(dt))
	assert.Equal(tf.t, cf.Name, name)
	assert.Equal(tf.t, cf.CamelCaseName, camelCaseName)
	assert.Equal(tf.t, cf.StructName, structName)
}

func (tf constraintsTestFixture) badColumn(
	dataTypeID uuid.UUID,
	isArray bool,
	constraints column.Constraints,
) {
	tf.t.Helper()

	c := tf.buildColumn(dataTypeID, isArray, constraints)
	_, err := tf.cm.SaveColumn(tf.ctx, c)
	assert.NotNil(tf.t, err)
}

func (tf constraintsTestFixture) badDataType(fieldNames ...string) {
	tf.t.Helper()

	dt := userstore.ColumnDataType{
		Name:        fmt.Sprintf("test_data_type_%v", uuid.Must(uuid.NewV4())),
		Description: "test data type",
	}

	for _, fieldName := range fieldNames {
		dt.CompositeAttributes.Fields = append(
			dt.CompositeAttributes.Fields,
			userstore.CompositeField{
				DataType: datatype.String,
				Name:     fieldName,
			},
		)
	}

	_, err := tf.dtm.CreateDataTypeFromClient(tf.ctx, &dt)
	assert.NotNil(tf.t, err)
}

func (tf constraintsTestFixture) buildColumn(
	dataTypeID uuid.UUID,
	isArray bool,
	constraints column.Constraints,
) *storage.Column {
	tf.t.Helper()

	return &storage.Column{
		BaseModel:  ucdb.NewBase(),
		Table:      "users",
		Name:       fmt.Sprintf("test_%v", uuid.Must(uuid.NewV4())),
		DataTypeID: dataTypeID,
		IsArray:    isArray,
		IndexType:  storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed),
		Attributes: storage.ColumnAttributes{
			Constraints: constraints,
		},
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	}
}

func (tf constraintsTestFixture) goodColumn(
	dataTypeID uuid.UUID,
	isArray bool,
	constraints column.Constraints,
) {
	tf.t.Helper()

	c := tf.buildColumn(dataTypeID, isArray, constraints)
	_, err := tf.cm.SaveColumn(tf.ctx, c)
	assert.NoErr(tf.t, err)
}

func TestConstraints(t *testing.T) {
	t.Parallel()

	tf := newConstraintsTestFixture(t)

	t.Run("test_data_type_fields", func(t *testing.T) {
		// supported data types
		tf.goodField(datatype.Birthdate, "FOO", "FOO", "foo")
		tf.goodField(datatype.Boolean, "FOO", "FOO", "foo")
		tf.goodField(datatype.Date, "FOO", "FOO", "foo")
		tf.goodField(datatype.Email, "FOO", "FOO", "foo")
		tf.goodField(datatype.Integer, "FOO", "FOO", "foo")
		tf.goodField(datatype.E164PhoneNumber, "FOO", "FOO", "foo")
		tf.goodField(datatype.PhoneNumber, "FOO", "FOO", "foo")
		tf.goodField(datatype.SSN, "FOO", "FOO", "foo")
		tf.goodField(datatype.String, "FOO", "FOO", "foo")
		tf.goodField(datatype.Timestamp, "FOO", "FOO", "foo")
		tf.goodField(datatype.UUID, "FOO", "FOO", "foo")

		// different supported name styles
		tf.goodField(datatype.String, "Foo", "Foo", "foo")
		tf.goodField(datatype.String, "Foo1", "Foo1", "foo1")
		tf.goodField(datatype.String, "F1o", "F1o", "f1o")
		tf.goodField(datatype.String, "Foo_Bar_Baz", "FooBarBaz", "foo_bar_baz")
		tf.goodField(datatype.String, "Foo_BAR_1", "FooBAR1", "foo_bar_1")

		// unsupported data types
		tf.badField(userstore.ResourceID{Name: "invalid"}, "FOO")
		tf.badField(userstore.ResourceID{ID: datatype.Composite.ID}, "FOO")

		// unsupported field names
		tf.badField(datatype.String, "foo")
		tf.badField(datatype.String, "")
		tf.badField(datatype.String, "_FOO")
		tf.badField(datatype.String, "_FOO_")
		tf.badField(datatype.String, "FooBar")
		tf.badField(datatype.String, "Foo_bar")
		tf.badField(datatype.String, "1_Foo")
		tf.badField(datatype.String, "Foo_1aB")
	})

	t.Run("test_bad_data_types", func(t *testing.T) {
		tf.badDataType("Foo", "Foo")
		tf.badDataType("Foo", "FOO")
		tf.badDataType("FO_O", "FOO")
		tf.badDataType("FO_O", "Fo_O")
	})

	t.Run("test_native_column_constraints", func(t *testing.T) {
		invalidSingleValueConstraints := []column.Constraints{
			{PartialUpdates: true},
			{UniqueRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validSingleValueConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
		}

		invalidArrayConstraints := []column.Constraints{
			{PartialUpdates: true},
			{UniqueIDRequired: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validArrayConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
			{UniqueRequired: true, PartialUpdates: true},
		}

		for _, dataType := range column.GetNativeDataTypes() {
			for _, constraints := range validSingleValueConstraints {
				tf.goodColumn(dataType.ID, false, constraints)
			}

			for _, constraints := range invalidSingleValueConstraints {
				tf.badColumn(dataType.ID, false, constraints)
			}

			for _, constraints := range validArrayConstraints {
				tf.goodColumn(dataType.ID, true, constraints)
			}

			for _, constraints := range invalidArrayConstraints {
				tf.badColumn(dataType.ID, true, constraints)
			}
		}
	})

	t.Run("test_composite_without_id_column_constraints", func(t *testing.T) {
		dt := userstore.ColumnDataType{
			Name:        fmt.Sprintf("test_data_type_%v", uuid.Must(uuid.NewV4())),
			Description: "test data type",
			CompositeAttributes: userstore.CompositeAttributes{
				Fields: []userstore.CompositeField{
					{
						DataType: datatype.String,
						Name:     "Foo",
					},
				},
			},
		}
		_, err := tf.dtm.CreateDataTypeFromClient(tf.ctx, &dt)
		assert.NoErr(tf.t, err)

		invalidSingleValueConstraints := []column.Constraints{
			{PartialUpdates: true},
			{UniqueIDRequired: true},
			{UniqueRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validSingleValueConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
		}

		invalidArrayConstraints := []column.Constraints{
			{PartialUpdates: true},
			{UniqueIDRequired: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validArrayConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
			{UniqueRequired: true, PartialUpdates: true},
		}

		for _, constraints := range validSingleValueConstraints {
			tf.goodColumn(dt.ID, false, constraints)
		}

		for _, constraints := range invalidSingleValueConstraints {
			tf.badColumn(dt.ID, false, constraints)
		}

		for _, constraints := range validArrayConstraints {
			tf.goodColumn(dt.ID, true, constraints)
		}

		for _, constraints := range invalidArrayConstraints {
			tf.badColumn(dt.ID, true, constraints)
		}
	})

	t.Run("test_composite_with_id_column_constraints", func(t *testing.T) {
		invalidSingleValueConstraints := []column.Constraints{
			{PartialUpdates: true},
			{UniqueRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validSingleValueConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
			{UniqueIDRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
		}

		invalidArrayConstraints := []column.Constraints{
			{PartialUpdates: true},
			{ImmutableRequired: true},
			{ImmutableRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		validArrayConstraints := []column.Constraints{
			{},
			{UniqueRequired: true},
			{UniqueRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true},
			{UniqueIDRequired: true, PartialUpdates: true},
			{UniqueIDRequired: true, UniqueRequired: true},
			{UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, PartialUpdates: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true},
			{ImmutableRequired: true, UniqueIDRequired: true, UniqueRequired: true, PartialUpdates: true},
		}

		for _, constraints := range validSingleValueConstraints {
			tf.goodColumn(datatype.CanonicalAddress.ID, false, constraints)
		}

		for _, constraints := range invalidSingleValueConstraints {
			tf.badColumn(datatype.CanonicalAddress.ID, false, constraints)
		}

		for _, constraints := range validArrayConstraints {
			tf.goodColumn(datatype.CanonicalAddress.ID, true, constraints)
		}

		for _, constraints := range invalidArrayConstraints {
			tf.badColumn(datatype.CanonicalAddress.ID, true, constraints)
		}
	})
}
