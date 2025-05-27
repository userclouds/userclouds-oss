// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o ObjectType) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	// .getCursor() lets you generate a cursor for additional sort keys
	cursor := pagination.CursorBegin
	o.getCursor(k, &cursor)
	return cursor
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o ObjectType) GetPaginationKeys() pagination.KeyTypes {
	// .getPaginationKeys() lets you add additional supported pagination keys
	keyTypes := o.getPaginationKeys()
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// NewObjectTypePaginatorFromOptions generates a paginator for a ObjectType
func NewObjectTypePaginatorFromOptions(
	options ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ObjectType
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

// NewObjectTypePaginatorFromQuery generates a paginator for a ObjectType
func NewObjectTypePaginatorFromQuery(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ObjectType
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

// NewObjectTypePaginatorFromRequest generates a paginator and cursor maker for a ObjectType
func NewObjectTypePaginatorFromRequest(
	r *http.Request,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType ObjectType
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
