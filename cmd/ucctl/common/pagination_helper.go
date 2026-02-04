package common

import (
	"context"

	"userclouds.com/infra/pagination"
)

// PaginatedFetcher is a function that fetches a page of results
// It takes a cursor and returns data, next cursor, hasMore flag, and error
type PaginatedFetcher[T any] func(ctx context.Context, cursor pagination.Cursor) (data []T, next pagination.Cursor, hasMore bool, err error)

// FetchAllPaginated fetches all pages using the provided fetch function
// It handles the pagination loop and accumulates all results
func FetchAllPaginated[T any](ctx context.Context, fetcher PaginatedFetcher[T]) ([]T, error) {
	var allItems []T
	cursor := pagination.CursorBegin

	for {
		items, nextCursor, hasMore, err := fetcher(ctx, cursor)
		if err != nil {
			return nil, err
		}

		allItems = append(allItems, items...)

		if !hasMore {
			break
		}
		cursor = nextCursor
	}

	return allItems, nil
}
