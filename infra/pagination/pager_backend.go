package pagination

import (
	"fmt"
	"strings"

	"userclouds.com/infra/ucerr"
)

// Split returns the list of keys in the Key
func (k Key) Split() []string {
	return strings.Split(string(k), ",")
}

// GetCursorClause returns the appropriate cursor clause for the paginator settings
func (p Paginator) GetCursorClause(nextParamIndex int, keys []string, keysNullable []bool) (string, int) {
	if p.cursor == CursorBegin || p.cursor == CursorEnd {
		return "", nextParamIndex
	}

	var values []string
	if len(keys) > 0 {
		for keyValue := range strings.SplitSeq(string(p.cursor), ",") {
			keyValueSplit := strings.Split(keyValue, ":")
			values = append(values, keyValueSplit[1])
		}
	} else {
		for keyValue := range strings.SplitSeq(string(p.cursor), ",") {
			keyValueSplit := strings.Split(keyValue, ":")
			key := keyValueSplit[0]
			keys = append(keys, key)
			values = append(values, keyValueSplit[1])
			keysNullable = append(keysNullable, p.isNullableKey(key))
		}
	}

	paramIndices := make([]int, len(values))
	for i, value := range values {
		if value != "" {
			paramIndices[i] = nextParamIndex
			nextParamIndex++
		}
	}

	// Cursor behavior is determined by the direction of pagination
	// and the sort order of pagination. If we are paging forward
	// and the sort order is ascending, we look for results that
	// are logically greater than the value pointed to by the cursor.
	// If we are paging forward but the sort order is descending, we
	// look for results that are logically less than the cursor value.
	// If we are paging backwards and are sorting ascending, we look
	// for results that are logically less than the cursor value. And
	// finally, if we are paging backwards but are sorting descending,
	// we look for results that are logically greater than than the
	// cursor value.
	//
	// If the cursor has a single key, by definition the key must be
	// "id", and must have a type of non-nullable UUID. If we are looking
	// for values greater than the cursor, the clause is of the form:
	//
	//   (id > 'value')
	//
	// If we are looking for values less than the cursor, the clause is of the form:
	//
	//   (id < 'value')
	//
	// If the cursor has multiple keys N, the first N-1 keys can possibly
	// have null values, depending on the key type. The last key must be "id".
	//
	// If we are looking for values greater than the cursor, and all of the
	// cursor values are non-null, the clause is of the form:
	//
	// (
	//    (key1 = 'value1' AND ... AND keyN-1 = 'valueN-1' AND id > 'valueN')
	//    OR (key1 = 'value1' AND ... AND keyN-2 = 'valueN-2' AND keyN-1 > 'valueN-1')
	//    ...
	//    OR (key1 > 'value1')
	// )
	//
	// If the first cursor value is null, the clause is of the following form,
	// with similar behavior extrapolated to other keys having null values:
	//
	// (
	//    (key1 IS NULL AND ... AND keyN-1 = 'valueN-1' AND id > 'valueN')
	//    OR (key1 IS NULL AND ... AND keyN-2 = 'valueN-2' AND keyN-1 > 'valueN-1')
	//    ...
	//    OR (key1 IS NOT NULL)
	// )
	//
	// If the cursor has multiple keys N, the first N-1 keys are nullable, we
	// are looking for values less than the cursor, and all of the cursor values
	// are non-null, the clause is of the form:
	//
	// (
	//    (key1 = 'value1' AND ... AND keyN-1 = 'valueN-1' AND id < 'valueN')
	//    OR (key1 = 'value1' AND ... AND keyN-2 = 'valueN-2' AND (keyN-1 < 'valueN-1' OR keyN-1 IS NULL))
	//    ...
	//    OR (key1 < 'value1' OR key1 IS NULL)
	// )
	//
	// If the cursor has multiple keys N, the first N-1 keys are nullable, we
	// are looking for values less than the cursor, and the first cursor value
	// is null, the clause is of the following form, with similar behavior extrapolated
	// to other keys having null values:
	//
	// (
	//    (key1 IS NULL AND ... AND keyN-1 = 'valueN-1' AND id < 'valueN')
	//    OR (key1 IS NULL AND ... AND keyN-2 = 'valueN-2' AND (keyN-1 < 'valueN-1' OR keyN-1 IS NULL))
	//    ...
	//    OR (key1 IS NULL AND (keyN-1 < 'value1' OR keyN-1 IS NULL))
	// )

	var builder strings.Builder
	joiner := ""
	totalKeys := len(keys)

	if totalKeys > 1 {
		builder.WriteString("(")
	}

	if p.isCursorDirectionForward() {
		for i := totalKeys - 1; i >= 0; i-- {
			if keysNullable[i] && values[i] == "" {
				fmt.Fprintf(&builder, "%s(%s IS NOT NULL", joiner, keys[i])
			} else {
				fmt.Fprintf(&builder, "%s(%s > $%d", joiner, keys[i], paramIndices[i])
			}

			for j := i - 1; j >= 0; j-- {
				if keysNullable[j] && values[j] == "" {
					fmt.Fprintf(&builder, " AND %s IS NULL", keys[j])
				} else {
					fmt.Fprintf(&builder, " AND %s = $%d", keys[j], paramIndices[j])
				}
			}
			builder.WriteString(")")
			joiner = " OR "
		}
	} else {
		for i := totalKeys - 1; i >= 0; i-- {
			if values[i] != "" {
				if keysNullable[i] {
					fmt.Fprintf(&builder, "%s((%s < $%d OR %s IS NULL)", joiner, keys[i], paramIndices[i], keys[i])
				} else {
					fmt.Fprintf(&builder, "%s(%s < $%d", joiner, keys[i], paramIndices[i])
				}
			}

			for j := i - 1; j >= 0; j-- {
				if keysNullable[j] && values[j] == "" {
					fmt.Fprintf(&builder, " AND %s IS NULL", keys[j])
				} else {
					fmt.Fprintf(&builder, " AND %s = $%d", keys[j], paramIndices[j])
				}
			}
			builder.WriteString(")")
			joiner = " OR "
		}
	}

	if totalKeys > 1 {
		builder.WriteString(")")
	}

	return builder.String(), nextParamIndex
}

// GetCursorFields returns the appropriate cursor fields for the paginator settings
func (p Paginator) GetCursorFields(fields []any) ([]any, error) {
	if p.cursor != CursorBegin && p.cursor != CursorEnd {
		for keyValue := range strings.SplitSeq(string(p.cursor), ",") {
			keyValueSplit := strings.Split(keyValue, ":")
			value, err := p.supportedKeys.getValidCursorExactValue(keyValueSplit[0], keyValueSplit[1])
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			if value != nil {
				fields = append(fields, value)
			}
		}
	}

	return fields, nil
}

func (p Paginator) getFilterClause(paramIndex int) string {
	if p.filterQuery != nil {
		s, _ := p.filterQuery.queryString(paramIndex)
		return fmt.Sprintf(" AND %s", s)
	}

	return ""
}

// GetQueryFields returns a list of query fields based on the cursor clause and filter clause in the paginator settings
func (p Paginator) GetQueryFields() (queryFields []any, err error) {
	if !p.hasResultType {
		return nil, ucerr.New("resultType is not set")
	}

	if err := p.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// add cursor fields first
	queryFields, err = p.GetCursorFields(queryFields)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// add filter fields
	if p.filterQuery != nil {
		queryFields, err = p.filterQuery.queryFields(p.supportedKeys, queryFields)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return queryFields, nil
}

// GetInnerOrderByClause returns an inner order by clause based on the paginator settings
func (p Paginator) GetInnerOrderByClause() string {
	return p.getOrderByClause(p.GetInnerOrderByDirection())
}

// GetInnerOrderByDirection returns the inner order by direction based on the paginator settings
func (p Paginator) GetInnerOrderByDirection() string {
	if p.sortOrder == OrderAscending {
		if p.IsForward() {
			return "ASC"
		}

		return "DESC"
	}

	if p.IsForward() {
		return "DESC"
	}

	return "ASC"
}

func (p Paginator) getOrderByClause(direction string) string {
	var builder strings.Builder
	joiner := ""
	for _, sortKey := range p.GetSortKey().Split() {
		fmt.Fprintf(&builder, "%s%s %s", joiner, sortKey, direction)
		joiner = ", "
	}
	return builder.String()
}

// GetOuterOrderByClause returns an outer order by direction based on the paginator settings
func (p Paginator) GetOuterOrderByClause() string {
	return p.getOrderByClause(p.getOuterOrderByDirection())
}

func (p Paginator) getOuterOrderByDirection() string {
	if p.sortOrder == OrderAscending {
		return "ASC"
	}

	return "DESC"
}

// GetWhereClause returns a where clause including cursor and filter parameters based on the paginator settings
func (p Paginator) GetWhereClause() string {
	cursorClause, paramIndex := p.GetCursorClause(1, nil, nil)
	if cursorClause != "" {
		cursorClause = fmt.Sprintf(" AND %s", cursorClause)
	}
	return cursorClause + p.getFilterClause(paramIndex)
}

// IsCachable returns true if the paginator is configured to be default and cached value can be returned
func (p Paginator) IsCachable() bool {
	return p.filter == "" && p.sortKey == "id" && p.sortOrder == OrderAscending
}

func (p Paginator) isCursorDirectionForward() bool {
	if p.sortOrder == OrderAscending {
		return p.IsForward()
	}

	return !p.IsForward()
}

// IsInitialQuery returns true if this is a forward query from the beginning or a reverse query from the end
func (p Paginator) IsInitialQuery() bool {
	if p.IsForward() {
		return p.cursor == CursorBegin
	}

	return p.cursor == CursorEnd
}

// setResultType sets & validates the paginator's result type
func (p *Paginator) setResultType(result PageableType) {
	p.hasResultType = true
	p.supportedKeys = result.GetPaginationKeys()
	p.isKeySupported = func(key string) bool {
		_, found := p.supportedKeys[key]
		return found
	}
	p.isNullableKey = func(key string) bool {
		return p.supportedKeys.isNullableKey(key)
	}
	p.isValidFinalSortKey = func(key string) bool {
		return p.supportedKeys.isValidFinalSortKey(key)
	}
	p.keyValueValidator = func(key string, value string) error {
		if err := p.supportedKeys.isValidCursorExactValue(key, value); err != nil {
			return ucerr.Errorf("cursor key:value pair '%s,%s' is invalid: '%v'", key, value, err)
		}
		return nil
	}
	p.supportedKeysValidator = func() error {
		if err := p.supportedKeys.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if p.filterQuery != nil {
			if err := p.filterQuery.IsValid(p.supportedKeys); err != nil {
				return ucerr.Wrap(err)
			}
		}
		return nil
	}
}
