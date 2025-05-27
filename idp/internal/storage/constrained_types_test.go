package storage_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

type Book struct {
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Author      string `json:"author,omitempty" yaml:"author,omitempty"`
	AuthorEmail string `json:"author_email,omitempty" yaml:"author_email,omitempty"`
	Length      int    `json:"length,omitempty" yaml:"length,omitempty"`
}

func NewBookFromCompositeValue(cv userstore.CompositeValue) (*Book, error) {
	b, err := json.Marshal(cv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var v Book
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &v, nil
}

func validateBooks(t *testing.T, vals []Book, cvs []userstore.CompositeValue) {
	t.Helper()

	assert.Equal(t, len(cvs), len(vals))
	for i, val := range vals {
		b, err := NewBookFromCompositeValue(cvs[i])
		assert.NoErr(t, err)
		assert.True(t, val == *b)
	}
}

type BookWithID struct {
	ID          string `json:"id,omitempty" yaml:"id,omitempty"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Author      string `json:"author,omitempty" yaml:"author,omitempty"`
	AuthorEmail string `json:"author_email,omitempty" yaml:"author_email,omitempty"`
	Length      int    `json:"length,omitempty" yaml:"length,omitempty"`
}

func NewBookWithIDFromCompositeValue(cv userstore.CompositeValue) (*BookWithID, error) {
	b, err := json.Marshal(cv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var v BookWithID
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &v, nil
}

func validateBooksWithIDs(t *testing.T, vals []BookWithID, cvs []userstore.CompositeValue) {
	t.Helper()

	assert.Equal(t, len(cvs), len(vals))
	for i, val := range vals {
		b, err := NewBookWithIDFromCompositeValue(cvs[i])
		assert.NoErr(t, err)
		assert.True(t, val == *b)
	}
}

func newBookDataType(includeID bool) column.DataType {
	dt := column.DataType{
		BaseModel:          ucdb.NewBase(),
		Name:               "book",
		Description:        "a book",
		ConcreteDataTypeID: datatype.Composite.ID,
		CompositeAttributes: column.CompositeAttributes{
			Fields: []column.CompositeField{
				{
					DataTypeID:    datatype.String.ID,
					Name:          "Title",
					CamelCaseName: "Title",
					StructName:    "title",
					Required:      true,
				},
				{
					DataTypeID:    datatype.String.ID,
					Name:          "Author",
					CamelCaseName: "Author",
					StructName:    "author",
				},
				{
					DataTypeID:    datatype.Email.ID,
					Name:          "Author_Email",
					CamelCaseName: "AuthorEmail",
					StructName:    "author_email",
				},
				{
					DataTypeID:    datatype.Integer.ID,
					Name:          "Length",
					CamelCaseName: "Length",
					StructName:    "length",
				},
			},
		},
	}

	if includeID {
		dt.CompositeAttributes.IncludeID = true
		dt.CompositeAttributes.Fields = append(
			dt.CompositeAttributes.Fields,
			column.CompositeField{
				DataTypeID:          datatype.String.ID,
				Name:                "ID",
				CamelCaseName:       "ID",
				StructName:          "id",
				Required:            true,
				IgnoreForUniqueness: true,
			},
		)
	}

	return dt
}

func TestCompositeValueSingleValueCoercion(t *testing.T) {
	t.Parallel()

	dt := newBookDataType(false)
	var constraints column.Constraints
	dc, err := column.NewDataCoercer(dt, constraints)
	assert.NoErr(t, err)

	book := Book{Title: "Moby Dick", Author: "Herman Melville", AuthorEmail: "herman@melville.org", Length: 427}

	// convert a struct
	cv, err := dc.ToCompositeValue(book)
	assert.NoErr(t, err)
	b, err := NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, book == *b)

	// convert the string representation of the struct
	bytes, err := json.Marshal(book)
	assert.NoErr(t, err)
	cv, err = dc.ToCompositeValue(string(bytes))
	assert.NoErr(t, err)
	b, err = NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, book == *b)

	// convert the CompositeValue representation of the struct
	cv, err = dc.ToCompositeValue(userstore.CompositeValue{"title": "Moby Dick", "author": "Herman Melville", "author_email": "herman@melville.org", "length": 427})
	assert.NoErr(t, err)
	b, err = NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, book == *b)

	// convert a string representation
	cv, err = dc.ToCompositeValue(`{"title":"Moby Dick","author":"Herman Melville","author_email":"herman@melville.org","length":427}`)
	assert.NoErr(t, err)
	b, err = NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, book == *b)

	// succeed if optional fields are missing
	cv, err = dc.ToCompositeValue(Book{Title: "Moby Dick"})
	assert.NoErr(t, err)
	b, err = NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.Equal(t, b.Title, "Moby Dick")
	assert.Equal(t, b.Author, "")
	assert.Equal(t, b.AuthorEmail, "")
	assert.Equal(t, b.Length, 0)

	// fail if field is not single-value

	_, err = dc.ToCompositeValue([]Book{book, book})
	assert.NotNil(t, err)

	// fail if required field is missing
	_, err = dc.ToCompositeValue(Book{Author: "Herman Melville"})
	assert.NotNil(t, err)

	_, err = dc.ToCompositeValue(userstore.CompositeValue{"author": "Herman Melville"})
	assert.NotNil(t, err)

	// fail if unexpected field is present
	_, err = dc.ToCompositeValue(userstore.CompositeValue{"title": "Moby Dick", "bad_field": "foo"})
	assert.NotNil(t, err)

	// fail if field value is invalid
	_, err = dc.ToCompositeValue(userstore.CompositeValue{"title": "Moby Dick", "author_email": "bad_email"})
	assert.NotNil(t, err)

	_, err = dc.ToCompositeValue(userstore.CompositeValue{"title": "Moby Dick", "length": "bad_length"})
	assert.NotNil(t, err)
}

func TestCompositeValueMultiValueCoercion(t *testing.T) {
	t.Parallel()

	dt := newBookDataType(false)
	var constraints column.Constraints
	dc, err := column.NewDataCoercer(dt, constraints)
	assert.NoErr(t, err)

	books := []Book{
		{Title: "Moby Dick", Author: "Herman Melville", AuthorEmail: "herman@melville.org", Length: 427},
		{Title: "Dune", Author: "Frank Herbert", Length: 896},
	}

	// convert a slice of structs
	cvs, err := dc.ToCompositeValues(books)
	assert.NoErr(t, err)
	validateBooks(t, books, cvs)

	// allow duplicates if unique is not enabled

	dupBooks := []Book{{Title: "foo"}, {Title: "foo"}}
	cvs, err = dc.ToCompositeValues(dupBooks)
	assert.NoErr(t, err)
	validateBooks(t, dupBooks, cvs)

	// convert the string representation of the slice of structs
	bytes, err := json.Marshal(books)
	assert.NoErr(t, err)
	cvs, err = dc.ToCompositeValues(string(bytes))
	assert.NoErr(t, err)
	validateBooks(t, books, cvs)

	// convert a slice of CompositeValues
	cvs, err = dc.ToCompositeValues([]userstore.CompositeValue{
		{"title": "Moby Dick", "author": "Herman Melville", "author_email": "herman@melville.org", "length": 427},
		{"title": "Dune", "author": "Frank Herbert", "length": 896},
	})
	assert.NoErr(t, err)
	validateBooks(t, books, cvs)

	// convert the string representation of a slice of CompositeValues
	cvs, err = dc.ToCompositeValues(
		`[` +
			`{"title": "Moby Dick", "author": "Herman Melville", "author_email": "herman@melville.org", "length": 427}` +
			`,{"title": "Dune", "author": "Frank Herbert", "length": 896}` +
			`]`,
	)
	assert.NoErr(t, err)
	validateBooks(t, books, cvs)

	// fail if field is not multi-value

	_, err = dc.ToCompositeValues(books[0])
	assert.NotNil(t, err)

	// fail if required field is missing

	_, err = dc.ToCompositeValues([]Book{{Title: "Foo"}, {Author: "Bar"}})
	assert.NotNil(t, err)

	// fail if unexpected field is present

	_, err = dc.ToCompositeValues([]userstore.CompositeValue{{"title": "Foo"}, {"title": "Bar", "bad_field": "Baz"}})
	assert.NotNil(t, err)

	// fail if field value is invalid

	_, err = dc.ToCompositeValues([]userstore.CompositeValue{{"title": "Foo"}, {"title": "Bar", "author_email": "bad_email"}})
	assert.NotNil(t, err)
}

func TestCompositeValueUniqueIDConstraint(t *testing.T) {
	t.Parallel()

	dt := newBookDataType(true)
	constraints := column.Constraints{UniqueIDRequired: true}
	dc, err := column.NewDataCoercer(dt, constraints)
	assert.NoErr(t, err)

	// generate ID
	cv, err := dc.ToCompositeValue(BookWithID{Title: "foo"})
	assert.NoErr(t, err)
	bwid, err := NewBookWithIDFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, len(bwid.ID) > 0)
	assert.Equal(t, bwid.Title, "foo")
	assert.Equal(t, bwid.Author, "")
	assert.Equal(t, bwid.AuthorEmail, "")
	assert.Equal(t, bwid.Length, 0)

	// reuse ID
	cv, err = dc.ToCompositeValue(*bwid)
	assert.NoErr(t, err)
	returnedBwid, err := NewBookWithIDFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, *bwid == *returnedBwid)

	// multi-value
	cvs, err := dc.ToCompositeValues([]BookWithID{{ID: "id", Title: "foo"}, {Title: "bar"}})
	assert.NoErr(t, err)
	assert.Equal(t, len(cvs), 2)
	assert.Equal(t, cvs[0]["id"], "id")
	s, ok := cvs[1]["id"].(string)
	assert.True(t, ok)
	assert.True(t, len(s) > 0)

	// fail if ids are not unique
	cvs[1]["id"] = cvs[0]["id"]
	_, err = dc.ToCompositeValues(cvs)
	assert.NotNil(t, err)
}

func TestCompositeValueUniqueConstraint(t *testing.T) {
	t.Parallel()

	dt := newBookDataType(false)
	constraints := column.Constraints{UniqueRequired: true}
	dc, err := column.NewDataCoercer(dt, constraints)
	assert.NoErr(t, err)

	// succeed for single-value case
	cv, err := dc.ToCompositeValue(Book{Title: "title"})
	assert.NoErr(t, err)
	b, err := NewBookFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, *b == Book{Title: "title"})

	// succeed if fields are unique
	uniqueBooks := []Book{{Title: "foo", Author: "bar"}, {Title: "bar", Author: "foo"}}
	cvs, err := dc.ToCompositeValues(uniqueBooks)
	assert.NoErr(t, err)
	validateBooks(t, uniqueBooks, cvs)

	// fail if fields are non-unique
	dupBooks := []Book{{Title: "foo", Author: "foo"}, {Title: "foo", Author: "foo"}}
	_, err = dc.ToCompositeValues(dupBooks)
	assert.NotNil(t, err)
}

func TestCompositeValueUniqueIDAndUniqueConstraints(t *testing.T) {
	t.Parallel()

	dt := newBookDataType(true)
	constraints := column.Constraints{UniqueIDRequired: true, UniqueRequired: true}
	dc, err := column.NewDataCoercer(dt, constraints)
	assert.NoErr(t, err)

	// succeed for single-value case
	cv, err := dc.ToCompositeValue(BookWithID{ID: "foo", Title: "title"})
	assert.NoErr(t, err)
	bwid, err := NewBookWithIDFromCompositeValue(cv)
	assert.NoErr(t, err)
	assert.True(t, *bwid == BookWithID{ID: "foo", Title: "title"})

	// succeed if non-ID fields are unique
	uniqueBooksWithID := []BookWithID{{ID: "foo", Title: "title1"}, {ID: "bar", Title: "title2"}}
	cvs, err := dc.ToCompositeValues(uniqueBooksWithID)
	assert.NoErr(t, err)
	validateBooksWithIDs(t, uniqueBooksWithID, cvs)

	// fail if non-ID fields are not unique
	dupBooksWithID := []BookWithID{{ID: "foo", Title: "title"}, {ID: "bar", Title: "title"}}
	_, err = dc.ToCompositeValues(dupBooksWithID)
	assert.NotNil(t, err)
}

type unique_constraints_test_fixture struct {
	t *testing.T
}

func (tf unique_constraints_test_fixture) getDataType(dataTypeID uuid.UUID) column.DataType {
	tf.t.Helper()
	dt, err := column.GetNativeDataType(dataTypeID)
	assert.NoErr(tf.t, err)
	return *dt
}

func TestOtherTypeUniqueConstraints(t *testing.T) {
	t.Parallel()

	tf := unique_constraints_test_fixture{t: t}

	constraints := column.Constraints{UniqueRequired: true}

	//// DataTypeBoolean

	dc, err := column.NewDataCoercer(tf.getDataType(datatype.Boolean.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToBool(true)
	assert.NoErr(t, err)

	_, err = dc.ToBools([]bool{true, false})
	assert.NoErr(t, err)

	_, err = dc.ToBools([]bool{true, true})
	assert.NotNil(t, err)

	//// DataTypeInteger

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.Integer.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToInt(5)
	assert.NoErr(t, err)

	_, err = dc.ToInts([]int{5, 10})
	assert.NoErr(t, err)

	_, err = dc.ToInts([]int{5, 5})
	assert.NotNil(t, err)

	//// DataTypeString

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.String.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToString("foo")
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"foo", "bar"})
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"foo", "foo"})
	assert.NotNil(t, err)

	// DataTypeEmail

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.Email.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToString("foo@bar.com")
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"foo@bar.com", "bar@bar.com"})
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"foo@bar.com", "foo@bar.com"})
	assert.NotNil(t, err)

	// DataTypeSSN

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.SSN.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToString("111-11-1111")
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"111-11-1111", "222-22-2222"})
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"111-11-1111", "111-11-1111"})
	assert.NotNil(t, err)

	// DataTypePhoneNumber

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.PhoneNumber.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToString("123-456-7890")
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"123-456-7890", "234-567-8901"})
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"123-456-7890", "123-456-7890"})
	assert.NotNil(t, err)

	// DataTypeE164PhoneNumber

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.E164PhoneNumber.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToString("+12345678")
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"+12345678", "+123456789"})
	assert.NoErr(t, err)

	_, err = dc.ToStrings([]string{"+12345678", "+12345678"})
	assert.NotNil(t, err)

	//// DataTypeTimestamp

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.Timestamp.ID), constraints)
	assert.NoErr(t, err)

	fooTime, err := time.Parse(time.DateTime, "2023-09-01 12:00:00")
	assert.NoErr(t, err)
	barTime := fooTime.Add(time.Hour)

	_, err = dc.ToTimestamp(fooTime)
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooTime, barTime})
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooTime, fooTime})
	assert.NotNil(t, err)

	// DataTypeDate

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.Date.ID), constraints)
	assert.NoErr(t, err)

	fooDate, err := time.Parse(time.DateOnly, "2023-01-01")
	assert.NoErr(t, err)
	barDate, err := time.Parse(time.DateOnly, "2023-02-01")
	assert.NoErr(t, err)

	_, err = dc.ToTimestamp(fooDate)
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooDate, barDate})
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooDate, fooDate})
	assert.NotNil(t, err)

	// DataTypeBirthdate

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.Birthdate.ID), constraints)
	assert.NoErr(t, err)

	_, err = dc.ToTimestamp(fooDate)
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooDate, barDate})
	assert.NoErr(t, err)

	_, err = dc.ToTimestamps([]time.Time{fooDate, fooDate})
	assert.NotNil(t, err)

	//// DataTypeUUID

	dc, err = column.NewDataCoercer(tf.getDataType(datatype.UUID.ID), constraints)
	assert.NoErr(t, err)

	fooID := uuid.Must(uuid.NewV4())
	barID := uuid.Must(uuid.NewV4())

	_, err = dc.ToUUID(fooID)
	assert.NoErr(t, err)

	_, err = dc.ToUUIDs([]uuid.UUID{fooID, barID})
	assert.NoErr(t, err)

	_, err = dc.ToUUIDs([]uuid.UUID{fooID, fooID})
	assert.NotNil(t, err)
}
