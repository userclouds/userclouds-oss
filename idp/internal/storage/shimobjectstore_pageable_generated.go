// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o ShimObjectStore) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	// .getCursor() lets you generate a cursor for additional sort keys
	cursor := pagination.CursorBegin
	o.getCursor(k, &cursor)
	return cursor
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o ShimObjectStore) GetPaginationKeys() pagination.KeyTypes {
	// .getPaginationKeys() lets you add additional supported pagination keys
	keyTypes := o.getPaginationKeys()
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// NewShimObjectStorePaginatorFromOptions generates a paginator for a ShimObjectStore
func NewShimObjectStorePaginatorFromOptions(
	options ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ShimObjectStore
	options = append(options, pagination.ResultType(resultType))
	pager, err := pagination.ApplyOptions(options...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

	return pager, nil
}

// NewShimObjectStorePaginatorFromQuery generates a paginator for a ShimObjectStore
func NewShimObjectStorePaginatorFromQuery(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ShimObjectStore
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

// NewShimObjectStorePaginatorFromRequest generates a paginator and cursor maker for a ShimObjectStore
func NewShimObjectStorePaginatorFromRequest(
	r *http.Request,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ShimObjectStore
	defaultOptions = append(defaultOptions, pagination.ResultType(resultType))
	pager, err := pagination.NewPaginatorFromRequest(r, defaultOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

	return pager, nil
}
