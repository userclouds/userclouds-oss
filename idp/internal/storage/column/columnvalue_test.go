package column_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
)

type testFixture struct {
	t *testing.T
}

func (tf testFixture) getDataType(dataTypeID uuid.UUID) column.DataType {
	tf.t.Helper()
	dt, err := column.GetNativeDataType(dataTypeID)
	assert.NoErr(tf.t, err)
	return *dt
}

func TestValueSet(t *testing.T) {
	ctx := context.Background()
	tf := testFixture{t: t}

	var cv column.Value
	var c column.Constraints

	// First test the non-array version of every type
	assert.NoErr(t, cv.Set(tf.getDataType(datatype.String.ID), c, false, "foo"))
	val, err := cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "foo", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Boolean.ID), c, false, true))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "true", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Integer.ID), c, false, 123))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "123", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Timestamp.ID), c, false, "2019-01-01T00:00:00Z"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "2019-01-01T00:00:00Z", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Date.ID), c, false, "2019-01-01"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "2019-01-01", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Birthdate.ID), c, false, "2019-01-01"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "2019-01-01", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.UUID.ID), c, false, "123e4567-e89b-12d3-a456-426655440000"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426655440000", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.SSN.ID), c, false, "123-45-6789"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "123-45-6789", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Email.ID), c, false, "asdf@ghi.com"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "asdf@ghi.com", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.E164PhoneNumber.ID), c, false, "+1234567890"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "+1234567890", val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.PhoneNumber.ID), c, false, "123-456-7890"))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, "123-456-7890", val)

	// Next test the array version of every type
	assert.NoErr(t, cv.Set(tf.getDataType(datatype.String.ID), c, true, []string{"foo", "bar"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["foo","bar"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Boolean.ID), c, true, []bool{true, false}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `[true,false]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Integer.ID), c, true, []int{123, 456}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `[123,456]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Timestamp.ID), c, true, []string{"2019-01-01T00:00:00Z", "2020-01-01T00:00:00Z"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["2019-01-01T00:00:00Z","2020-01-01T00:00:00Z"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Date.ID), c, true, []string{"2019-01-01", "2020-01-01"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["2019-01-01","2020-01-01"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Birthdate.ID), c, true, []string{"2019-01-01", "2020-01-01"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["2019-01-01","2020-01-01"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.UUID.ID), c, true, []string{"123e4567-e89b-12d3-a456-426655440000", "123e4567-e89b-12d3-a456-426655440001"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["123e4567-e89b-12d3-a456-426655440000","123e4567-e89b-12d3-a456-426655440001"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.SSN.ID), c, true, []string{"123-45-6789", "123-45-6790"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["123-45-6789","123-45-6790"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Email.ID), c, true, []string{"asdf@ghi.com", "john@doe.com"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["asdf@ghi.com","john@doe.com"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.E164PhoneNumber.ID), c, true, []string{"+1234567890", "+2345678901"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["+1234567890","+2345678901"]`, val)

	assert.NoErr(t, cv.Set(tf.getDataType(datatype.PhoneNumber.ID), c, true, []string{"123-456-7890", "+1-123-456-7890"}))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, `["123-456-7890","+1-123-456-7890"]`, val)

	// Next test that the regex validation works for a few types
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.Email.ID), c, false, "foo"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.Email.ID), c, false, "foo@"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.E164PhoneNumber.ID), c, false, "foo"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.E164PhoneNumber.ID), c, false, "1234567890"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.PhoneNumber.ID), c, false, "foo"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.PhoneNumber.ID), c, false, "+1-123-456-7890 ext 123"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.PhoneNumber.ID), c, false, "+1-123-456-"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.SSN.ID), c, false, "foo"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.SSN.ID), c, false, "123-45-67890"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.Date.ID), c, false, "foo"))
	assert.NotNil(t, cv.Set(tf.getDataType(datatype.Birthdate.ID), c, false, "foo"))

	// Date types still accept full timestamps
	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Date.ID), c, false, "2010-01-01T00:00:00Z"))
	assert.NoErr(t, cv.Set(tf.getDataType(datatype.Birthdate.ID), c, false, "2010-01-01T00:00:00Z"))
}

func TestCompositeValueSet(t *testing.T) {
	ctx := context.Background()
	var constraints column.Constraints

	dt := column.DataType{
		BaseModel:          ucdb.NewBase(),
		Name:               "test_composite",
		Description:        "test composite data type",
		ConcreteDataTypeID: datatype.Composite.ID,
		CompositeAttributes: column.CompositeAttributes{
			Fields: []column.CompositeField{
				{
					DataTypeID:    datatype.String.ID,
					Name:          "Foo",
					CamelCaseName: "Foo",
					StructName:    "foo",
				},
				{
					DataTypeID:    datatype.Integer.ID,
					Name:          "Bar",
					CamelCaseName: "Bar",
					StructName:    "bar",
				},
				{
					DataTypeID:    datatype.Boolean.ID,
					Name:          "Baz",
					CamelCaseName: "Baz",
					StructName:    "baz",
				},
			},
		},
	}

	singleValue := `{"bar":10,"baz":false,"foo":"a string"}`

	var cv column.Value
	assert.NoErr(t, cv.Set(dt, constraints, false, singleValue))
	val, err := cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, singleValue, val)

	arrayValue := `[{"bar":10,"baz":false,"foo":"a string"},{"bar":20,"baz":true,"foo":"another string"}]`
	assert.NoErr(t, cv.Set(dt, constraints, true, arrayValue))
	val, err = cv.GetString(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, arrayValue, val)
}
