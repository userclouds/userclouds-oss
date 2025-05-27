// NOTE: automatically generated file -- DO NOT EDIT

package logeventmetadata

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o MetricMetadata) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	return pagination.CursorBegin
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o MetricMetadata) GetPaginationKeys() pagination.KeyTypes {
	keyTypes := pagination.KeyTypes{}
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// NewMetricMetadataPaginatorFromOptions generates a paginator for a MetricMetadata
func NewMetricMetadataPaginatorFromOptions(
	options ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MetricMetadata
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

// NewMetricMetadataPaginatorFromQuery generates a paginator for a MetricMetadata
func NewMetricMetadataPaginatorFromQuery(
	query pagination.Query,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MetricMetadata
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

// NewMetricMetadataPaginatorFromRequest generates a paginator and cursor maker for a MetricMetadata
func NewMetricMetadataPaginatorFromRequest(
	r *http.Request,
	defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
	var resultType MetricMetadata
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
