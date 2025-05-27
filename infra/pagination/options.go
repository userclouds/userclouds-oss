package pagination

import "fmt"

// Option defines a method of passing optional args to paginated List APIs
type Option interface {
	apply(*Paginator)
}

type optFunc func(*Paginator)

func (of optFunc) apply(p *Paginator) {
	of(p)
	p.options = append(p.options, of)
}

// EndingBefore iterates the collection's values before (but not including) the value at the cursor.
// It is commonly used to implement "Prev" functionality.
func EndingBefore(cursor Cursor) Option {
	return optFunc(
		func(p *Paginator) {
			p.cursor = cursor
			p.direction = DirectionBackward
			p.backwardDirectionSet = true
		})
}

// Filter specifies a filter clause to use in the pagination query.
//
// A filter query must either be:
//
//  1. A LEAF query, consisting of a key, operator, and value, formatted like:
//
//     ('KEY',OPERATOR,'VALUE')
//
//  2. A NESTED query, formatted like:
//
//     (FILTER_QUERY)
//
//  3. A COMPOSITE query, formatted like:
//
//     (FILTER_QUERY,OPERATOR,FILTER_QUERY,...,OPERATOR,FILTER_QUERY)
//
//     The OPERATORs in a COMPOSITE query must be a LOGICAL OPERATOR. Note that if more than one LOGICAL OPERATOR is present in
//     the COMPOSITE query, standard SQL precedence rules will be followed, with consecutive AND queries grouped together before
//     OR queries. So the query (fee,OR,fie,AND,foe,AND,fum) would be executed as (fee,OR,(fie,AND,foe,AND,fum)).
//
//     For NESTED and COMPOSITE queries, FILTER_QUERY can be a LEAF, NESTED, or COMPOSITE query.
//
// For LEAF queries, KEY must be a valid result type key for type of the result being returned. Valid keys are stored
// in a KeyTypes map in a configured paginator, mapping a key name to a KeyType. Supported KeyTypes include:
//
//	ArrayKeyType             (must be a string value to be searched for in an array of strings)
//	BoolKeyType              (value may be specified as any string that can be parsed by https://pkg.go.dev/strconv#example-ParseBool)
//	IntKeyType               (must be a valid string representation of an int64)
//	NullableBoolKeyType      (value must either be unspecified (i.e., NULL) or a valid BoolKeyType
//	NullableIntKeyType       (value must either be unspecified (i.e., NULL) or a valid IntKeyType
//	NullableStringKeyType    (value must either be unspecified (i.e., NULL) or a valid StringKeyType
//	NullableTimestampKeyType (value must either be unspecified (i.e., NULL) or a valid TimestampKeyType
//	NullableUUIDKeyType      (value must either be unspecified (i.e., NULL) or a valid UUIDKeyType
//	StringKeyType            (string value can only have single-quotes or double-quotes in the string that are escaped with a
//	                          back-slash (i.e., \' or \"))
//	TimestampKeyType         (the number of microseconds since January 1, 1970 UTC)
//	UUIDArrayKeyType         (must be a valid string representation of a UUID)
//	UUIDKeyType              (must be a valid string representation of a UUID)
//
// By default, all result types support "id" as a valid key of KeyType UUIDKeyType. New supported keys can be added
// to a result type by defining the GetPaginationKeys() method of the PageableType interface for the result type,
// adding the keys and associated KeyType for each key that is supported.
//
// For a LEAF query, the OPERATOR must be an ARRAY operator, a COMPARISON operator, or a PATTERN operator.
//
//	ARRAY operators include:
//
//		HAS // ANY
//
//	Only ArrayKeyType and UUIDArrayKeyType supports ARRAY operators.
//
//	COMPARISON operators include:
//
//		EQ  // =
//		GE  // >=
//		GT  // >
//		LE  // <=
//		LT  // <
//		NE  // !=
//
//	All supported KeyTypes other than ArrayKeyType and UUIDArrayKeyType support COMPARISON operators.
//
//	PATTERN operators include:
//
//		LK  // LIKE
//		NL  // NOT LIKE
//
//	Only StringKeyType keys support PATTERN operators. For a PATTERN operator, % matches 0 or more characters. _
//	matches any single character. To match the % or _ characters, the character must be escaped with a \ in the
//	value (i.e., '\%' matches the '%' character, and '\_' matches '_').
//
//	LOGICAL operators include:
//
//		AND // AND
//		OR  // OR
func Filter(filter string) Option {
	return optFunc(
		func(p *Paginator) {
			if filter != "" {
				if p.filter != "" {
					filter = fmt.Sprintf("(%s,AND,%s)", p.filter, filter)
				}
				p.filter = filter
			}
		})
}

// Limit specifies how many results to fetch at once. If unspecified, the default limit will be used.
func Limit(limit int) Option {
	return optFunc(
		func(p *Paginator) {
			p.limit = limit
		})
}

func requestVersion(version Version) Option {
	return optFunc(
		func(p *Paginator) {
			p.version = version
		})
}

// SortKey optionally specifies which field of the collection should be used to sort results in the view.
func SortKey(key Key) Option {
	return optFunc(
		func(p *Paginator) {
			p.sortKey = key
		})
}

// SortOrder optionally specifies which way a view on a collection should be sorted.
func SortOrder(order Order) Option {
	return optFunc(
		func(p *Paginator) {
			p.sortOrder = order
		})
}

// StartingAfter iterates the collection starting after (but not including) the value at the cursor.
// It can be used to fetch the "current" page starting at a cursor, as well as to iterate the next page
// by passing in the cursor of the last item on the page.
func StartingAfter(cursor Cursor) Option {
	return optFunc(
		func(p *Paginator) {
			p.cursor = cursor
			p.direction = DirectionForward
			p.forwardDirectionSet = true
		})
}
