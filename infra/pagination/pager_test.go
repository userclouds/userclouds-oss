package pagination_test

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type testType struct {
	id   uuid.UUID
	name string
	age  int
}

// GetCursor is part of the pagination.PageableType interface
func (tt testType) GetCursor(k pagination.Key) pagination.Cursor {
	switch k {
	case "name,age,id":
		return pagination.Cursor(
			fmt.Sprintf(
				"name:%s,age:%d,id:%v",
				tt.name,
				tt.age,
				tt.id,
			),
		)
	case "id":
		return pagination.Cursor(
			fmt.Sprintf(
				"id:%v",
				tt.id,
			),
		)
	}
	return pagination.CursorBegin
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (testType) GetPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"id":   pagination.UUIDKeyType,
		"name": pagination.NullableStringKeyType,
		"age":  pagination.NullableIntKeyType,
	}
}

func newTestTypePager(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType testType
	defaultOptions = append(defaultOptions, pagination.ResultType(resultType))
	pager, err := pagination.NewPaginatorFromQuery(query, defaultOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

	return pager, nil
}

func stringPointer(s string) *string {
	return &s
}

func TestCreateFromQuery(t *testing.T) {
	// default version 2 query
	q := pagination.QueryParams{}
	q.Version = stringPointer("2")
	p, err := newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version2)
	assert.Equal(t, p.GetCursor(), pagination.CursorBegin)
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err := p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// default query
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorBegin)
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// forward ascending query with empty cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer("")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorBegin)
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// forward descending query with empty cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer("")
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorBegin)
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id DESC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// backward ascending query with end cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer("end")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorEnd)
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// backward descending query with end cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer("end")
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorEnd)
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id DESC")
	assert.Equal(t, p.GetWhereClause(), "")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 0)

	// forward ascending query with non-begin cursor
	testID := uuid.Must(uuid.NewV4())
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("id:%v", testID))
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("id:%v", testID)))
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND (id > $1)")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 1)
	assert.Equal(t, fields[0], testID)

	// forward descending query with non-begin cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("id:%v", testID))
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("id:%v", testID)))
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id DESC")
	assert.Equal(t, p.GetWhereClause(), " AND (id < $1)")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 1)
	assert.Equal(t, fields[0], testID)

	// backward ascending query with non-end cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("id:%v", testID))
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("id:%v", testID)))
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND (id < $1)")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 1)
	assert.Equal(t, fields[0], testID)

	// backward descending query with non-end cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("id:%v", testID))
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("id:%v", testID)))
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id DESC")
	assert.Equal(t, p.GetWhereClause(), " AND (id > $1)")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 1)
	assert.Equal(t, fields[0], testID)

	// forward ascending query with specified filter
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.Filter = stringPointer(fmt.Sprintf("('id',EQ,'%v')", testID))
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.CursorBegin)
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND (id = $1)")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 1)
	assert.Equal(t, fields[0], testID)

	// forward ascending query with non-begin cursor and specified filter
	testID2 := uuid.Must(uuid.NewV4())
	testID3 := uuid.Must(uuid.NewV4())
	testID4 := uuid.Must(uuid.NewV4())
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("id:%v", testID))
	q.Filter = stringPointer(fmt.Sprintf("((('id',GT,'%v'),OR,('id',LT,'%v')),AND,('id',NE,'%v'))", testID2, testID3, testID4))
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("id:%v", testID)))
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND (id > $1) AND (((id > $2) OR (id < $3)) AND (id != $4))")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 4)
	assert.Equal(t, fields[0], testID)
	assert.Equal(t, fields[1], testID2)
	assert.Equal(t, fields[2], testID3)
	assert.Equal(t, fields[3], testID4)

	// forward ascending multi-key query
	var testAge int64 = 10
	testName := "foo"
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID))
	q.SortKey = stringPointer("name,age,id")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID)))
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "name ASC, age ASC, id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "name ASC, age ASC, id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND ((id > $3 AND age = $2 AND name = $1) OR (age > $2 AND name = $1) OR (name > $1))")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 3)
	assert.Equal(t, fields[0], testName)
	assert.Equal(t, fields[1], testAge)
	assert.Equal(t, fields[2], testID)

	// forward descending multi-key query
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID))
	q.SortKey = stringPointer("name,age,id")
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID)))
	assert.True(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "name DESC, age DESC, id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "name DESC, age DESC, id DESC")
	assert.Equal(t, p.GetWhereClause(), " AND ((id < $3 AND age = $2 AND name = $1) OR ((age < $2 OR age IS NULL) AND name = $1) OR ((name < $1 OR name IS NULL)))")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 3)
	assert.Equal(t, fields[0], testName)
	assert.Equal(t, fields[1], testAge)

	// backward ascending multi-key query
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID))
	q.SortKey = stringPointer("name,age,id")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID)))
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "name DESC, age DESC, id DESC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "name ASC, age ASC, id ASC")
	assert.Equal(t, p.GetWhereClause(), " AND ((id < $3 AND age = $2 AND name = $1) OR ((age < $2 OR age IS NULL) AND name = $1) OR ((name < $1 OR name IS NULL)))")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 3)
	assert.Equal(t, fields[0], testName)
	assert.Equal(t, fields[1], testAge)
	assert.Equal(t, fields[2], testID)

	// backward descending multi-key query
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID))
	q.SortKey = stringPointer("name,age,id")
	q.SortOrder = stringPointer("descending")
	p, err = newTestTypePager(q)
	assert.NoErr(t, err)
	assert.Equal(t, p.GetVersion(), pagination.Version3)
	assert.Equal(t, p.GetCursor(), pagination.Cursor(fmt.Sprintf("name:%s,age:%d,id:%v", testName, testAge, testID)))
	assert.False(t, p.IsForward())
	assert.Equal(t, p.GetInnerOrderByClause(), "name ASC, age ASC, id ASC")
	assert.Equal(t, p.GetLimit(), pagination.DefaultLimit)
	assert.Equal(t, p.GetOuterOrderByClause(), "name DESC, age DESC, id DESC")
	assert.Equal(t, p.GetWhereClause(), " AND ((id > $3 AND age = $2 AND name = $1) OR (age > $2 AND name = $1) OR (name > $1))")
	fields, err = p.GetQueryFields()
	assert.NoErr(t, err)
	assert.Equal(t, len(fields), 3)
	assert.Equal(t, fields[0], testName)
	assert.Equal(t, fields[1], testAge)
	assert.Equal(t, fields[2], testID)

	// failures

	// deprecated version
	q = pagination.QueryParams{}
	q.Version = stringPointer("1")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// legacy cursor format
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(testID.String())
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(testID.String())
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// start and end cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer("end")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer("")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// bad cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer("unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer("unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// another bad cursor
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer("id:unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer("id:unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// bad filter key
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	_, err = newTestTypePager(q, pagination.Filter(fmt.Sprintf("('foo',EQ,'%v')", testID)))
	assert.NotNil(t, err)

	// bad filter value
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	_, err = newTestTypePager(q, pagination.Filter("('id',EQ,'foo')"))
	assert.NotNil(t, err)

	// bad filter query

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	_, err = newTestTypePager(q, pagination.Filter(fmt.Sprintf("'id',EQ,'%v'", testID)))
	assert.NotNil(t, err)

	// bad order by key
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.StartingAfter = stringPointer(fmt.Sprintf("bad_key:%v", testID))
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("bad_key:%v", testID))
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// empty limit
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.Limit = stringPointer("")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// invalid limit
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.Limit = stringPointer("unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// negative limit
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.Limit = stringPointer("-1")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// excessive limit
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.Limit = stringPointer("1000000")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// bad sort key
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.SortKey = stringPointer("name")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.SortKey = stringPointer("unsupported,id")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.SortKey = stringPointer("")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// bad sort order
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.SortOrder = stringPointer("unsupported")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.SortOrder = stringPointer("")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// bad version
	q = pagination.QueryParams{}
	q.Version = stringPointer("4")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	q = pagination.QueryParams{}
	q.Version = stringPointer("")
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)

	// forward and reverse
	q = pagination.QueryParams{}
	q.Version = stringPointer("3")
	q.EndingBefore = stringPointer(fmt.Sprintf("id:%v", testID))
	q.StartingAfter = stringPointer(fmt.Sprintf("id:%v", testID))
	_, err = newTestTypePager(q)
	assert.NotNil(t, err)
}
