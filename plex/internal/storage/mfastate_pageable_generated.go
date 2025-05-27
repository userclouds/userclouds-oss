// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o MFAState) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	return pagination.CursorBegin
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o MFAState) GetPaginationKeys() pagination.KeyTypes {
	keyTypes := pagination.KeyTypes{}
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// NewMFAStatePaginatorFromOptions generates a paginator for a MFAState
func NewMFAStatePaginatorFromOptions(
	options ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MFAState
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

// NewMFAStatePaginatorFromQuery generates a paginator for a MFAState
func NewMFAStatePaginatorFromQuery(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MFAState
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

// NewMFAStatePaginatorFromRequest generates a paginator and cursor maker for a MFAState
func NewMFAStatePaginatorFromRequest(
	r *http.Request,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MFAState
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
