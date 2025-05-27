// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o Edge) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	return pagination.CursorBegin
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o Edge) GetPaginationKeys() pagination.KeyTypes {
	// .getPaginationKeys() lets you add additional supported pagination keys
	keyTypes := o.getPaginationKeys()
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// NewEdgePaginatorFromOptions generates a paginator for a Edge
func NewEdgePaginatorFromOptions(
	options ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType Edge
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

// NewEdgePaginatorFromQuery generates a paginator for a Edge
func NewEdgePaginatorFromQuery(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType Edge
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

// NewEdgePaginatorFromRequest generates a paginator and cursor maker for a Edge
func NewEdgePaginatorFromRequest(
	r *http.Request,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType Edge
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
