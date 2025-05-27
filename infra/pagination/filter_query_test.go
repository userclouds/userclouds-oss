package pagination_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
)

type testInt int

var testID = uuid.Must(uuid.NewV4())

var testBool = false

var testNum testInt = 1

var testTimestamp = time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC).UnixMicro()

var testSupportedKeys = pagination.KeyTypes{
	"id":                  pagination.UUIDKeyType,
	"ids":                 pagination.UUIDArrayKeyType,
	"created":             pagination.TimestampKeyType,
	"foo":                 pagination.StringKeyType,
	"bar":                 pagination.StringKeyType,
	"baz":                 pagination.StringKeyType,
	"num":                 pagination.IntKeyType,
	"flag":                pagination.BoolKeyType,
	"metadata->>'format'": pagination.StringKeyType,
	"nullablebool":        pagination.NullableBoolKeyType,
	"nullableint":         pagination.NullableIntKeyType,
	"nullablestring":      pagination.NullableStringKeyType,
	"nullabletimestamp":   pagination.NullableTimestampKeyType,
	"nullableuuid":        pagination.NullableUUIDKeyType,
}

func isValidQuery(s string) error {
	fq, err := pagination.CreateFilterQuery(s)
	if err != nil {
		return err
	}

	return fq.IsValid(testSupportedKeys)
}

func TestFilterParser(t *testing.T) {
	assert.NotNil(t, isValidQuery(`()`))
	assert.NotNil(t, isValidQuery(`foo`))
	assert.NotNil(t, isValidQuery(`('unknown',EQ,'blah')`))
	assert.NotNil(t, isValidQuery(`('foo',EQ,'blah'`))
	assert.NotNil(t, isValidQuery(`'foo',EQ,'blah')`))
	assert.NotNil(t, isValidQuery(`'foo',EQ,'blah'`))
	assert.NotNil(t, isValidQuery(`('foo',foo,'blah')`))
	assert.NotNil(t, isValidQuery(`('foo',AND,'blah')`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),LT,('bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),foo,('bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah')OR,('bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah',OR,('bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),OR('bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),OR,'bar',LK,'blah'))`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,'blah')`))
	assert.NotNil(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,')blah')`))

	assert.NotNil(t, isValidQuery(`('id',EQ,'blah')`))
	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('id',LK,'%v')`, testID)))

	assert.NotNil(t, isValidQuery(`('created',EQ,'blah')`))
	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('created',LK,'%v')`, testTimestamp)))

	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('id',HAS,'%v')`, testID)))
	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('created',HAS,'%v')`, testTimestamp)))
	assert.NotNil(t, isValidQuery(`('foo',HAS,'blah')`))
	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('num',HAS,'%v')`, testNum)))
	assert.NotNil(t, isValidQuery(fmt.Sprintf(`('flag',HAS,'%v')`, testBool)))

	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('nullablebool',EQ,'%v')`, testBool)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('nullableint',EQ,'%v')`, testNum)))
	assert.NoErr(t, isValidQuery(`('nullablestring',EQ,'blah')`))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('nullabletimestamp',EQ,'%v')`, testTimestamp)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('nullableuuid',EQ,'%v')`, testID)))

	assert.NoErr(t, isValidQuery(`('foo',EQ,'blah')`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',GT,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LE,'b\'_%lah'))`))
	assert.NoErr(t, isValidQuery(`((((('foo',EQ,'blah')))),OR,(('bar',LK,'b\'lah')))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,'b\')lah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,')blah'))`))
	assert.NoErr(t, isValidQuery(`((('foo',EQ,'blah'),AND,('bar',LK,'%b\%l_a_\_h%')))`))
	assert.NoErr(t, isValidQuery(`((('foo',EQ,'blah'),AND,('bar',LK,'blah')),OR,('baz',NE,'blah'))`))
	assert.NoErr(t, isValidQuery(`(((('foo',EQ,'blah'))))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,'blah'),OR,('baz',EQ,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),AND,('bar',LK,'blah'),AND,('baz',EQ,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,(('bar',LK,'blah'),AND,('baz',EQ,'blah')))`))
	assert.NoErr(t, isValidQuery(`((('bar',LK,'blah'),AND,('baz',EQ,'blah')),OR,('foo',EQ,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),AND,(('bar',LK,'blah'),OR,('baz',EQ,'blah')))`))
	assert.NoErr(t, isValidQuery(`((('bar',LK,'blah'),OR,('baz',EQ,'blah')),AND,('foo',EQ,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),OR,('bar',LK,'blah'),AND,('baz',EQ,'blah'))`))
	assert.NoErr(t, isValidQuery(`(('foo',EQ,'blah'),AND,('bar',LK,'blah'),OR,('baz',EQ,'blah'))`))

	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('id',NE,'%v')`, testID)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('created',LT,'%v')`, testTimestamp)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('num',EQ,'%v')`, testNum)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('flag',EQ,'%v')`, testBool)))

	assert.NoErr(t, isValidQuery(fmt.Sprintf(`('ids',HAS,'%v')`, testID)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`(('foo',EQ,'blah'),AND,('ids',HAS,'%v'))`, testID)))
	assert.NoErr(t, isValidQuery(fmt.Sprintf(`(('ids',HAS,'%v'),AND,('foo',EQ,'blah'))`, testID)))

	assert.NoErr(t, isValidQuery(`('metadata->>format',EQ,'structured')`))
	assert.NotNil(t, isValidQuery(`('metadata->format',EQ,'structured')`))
}
