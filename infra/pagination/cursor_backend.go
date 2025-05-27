package pagination

// ProcessResults is a generic method for post-processing a collection
// of results. Since we request one more than pageSize results, we shrink the
// collection to the specified page size if we found the extra result, and know
// that there are more results available in that case. We set the previous and
// next cursors using the provided MakeCursor method and the direction of the
// pagination
func ProcessResults[T PageableType](
	results []T,
	current Cursor,
	pageSize int,
	isForward bool,
	sortKey Key,
) ([]T, ResponseFields) {
	var next, prev Cursor
	if isForward {
		next = CursorEnd
		if len(results) > pageSize {
			results = results[:pageSize]
			next = results[pageSize-1].GetCursor(sortKey)
		}

		prev = CursorBegin
		if current != CursorBegin {
			if len(results) > 0 {
				prev = results[0].GetCursor(sortKey)
			}
		}
	} else {
		prev = CursorBegin
		if totalResults := len(results); totalResults > pageSize {
			results = results[totalResults-pageSize : totalResults]
			prev = results[0].GetCursor(sortKey)
		}

		next = CursorEnd
		if current != CursorEnd {
			if totalResults := len(results); totalResults > 0 {
				next = results[totalResults-1].GetCursor(sortKey)
			}
		}
	}

	// TODO: once we want to support long-lived cursors, if we've reached the end of the
	// result set while paginating forward (i.e., len(results) <= pageSize), we would set
	// Next to the last result cursor and Prev to CursorEnd. If we are paginating
	// backwards and hit the bottom of the result set (again, len(results) <= pageSize),
	// we would set Next to CursorBegin, and Prev to the first result (or CursorEnd if
	// there are no results). Long-lived cursor behavior would be turned on or off by a
	// setting in the Paginator.
	return results,
		ResponseFields{
			HasNext: next != CursorEnd,
			Next:    next,
			HasPrev: prev != CursorBegin,
			Prev:    prev,
		}
}
