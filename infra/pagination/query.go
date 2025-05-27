package pagination

import (
	"net/http"
	"strconv"

	"userclouds.com/infra/ucerr"
)

// NewPaginatorFromRequest calls NewPaginatorFromQuery, creating a PaginationQuery from the
// request and any specified default pagination options
func NewPaginatorFromRequest(r *http.Request, defaultOptions ...Option) (*Paginator, error) {
	urlValues := r.URL.Query()

	req := QueryParams{}
	if urlValues.Has("ending_before") {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("sort_key") {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("version") {
		v := urlValues.Get("version")
		req.Version = &v
	}

	p, err := NewPaginatorFromQuery(req, defaultOptions...)
	return p, ucerr.Wrap(err)
}

// Query is the interface needed by NewPaginatorFromQuery to create a Paginator
type Query interface {
	getStartingAfter() *string
	getEndingBefore() *string
	getLimit() *string
	getFilter() *string
	getSortKey() *string
	getSortOrder() *string
	getVersion() *string
}

// QueryParams is a struct that implements PaginationQuery, which can be incorporated in other request structs
// for handlers that need to take pagination query parameters
type QueryParams struct {
	StartingAfter *string `json:"starting_after,omitempty" yaml:"starting_after,omitempty" description:"A cursor value after which the returned list will start" query:"starting_after"`
	EndingBefore  *string `json:"ending_before,omitempty" yaml:"ending_before,omitempty" description:"A cursor value before which the returned list will end" query:"ending_before"`
	Limit         *string `json:"limit,omitempty" yaml:"limit,omitempty" description:"The maximum number of results to be returned per page" query:"limit"`
	Filter        *string `json:"filter,omitempty" yaml:"filter,omitempty" description:"A filter clause to use in the pagination query" query:"filter"`
	SortKey       *string `json:"sort_key,omitempty" yaml:"sort_key,omitempty" description:"A comma-delimited list of field names to sort the returned results by - the last field name must be 'id'" query:"sort_key"`
	SortOrder     *string `json:"sort_order,omitempty" yaml:"sort_order,omitempty" description:"The order in which results should be sorted (ascending or descending)" query:"sort_order"`
	Version       *string `json:"version,omitempty" yaml:"version,omitempty" description:"The version of the API to be called" query:"version"`
}

func (p QueryParams) getStartingAfter() *string {
	return p.StartingAfter
}

func (p QueryParams) getEndingBefore() *string {
	return p.EndingBefore
}

func (p QueryParams) getLimit() *string {
	return p.Limit
}

func (p QueryParams) getFilter() *string {
	return p.Filter
}

func (p QueryParams) getSortKey() *string {
	return p.SortKey
}

func (p QueryParams) getSortOrder() *string {
	return p.SortOrder
}

func (p QueryParams) getVersion() *string {
	return p.Version
}

// NewPaginatorFromQuery applies any default options and any additional options from parsing
// the query to produce a Paginator instance, validates that instance, and returns it if valid
func NewPaginatorFromQuery(query Query, defaultOptions ...Option) (*Paginator, error) {
	options := []Option{}

	// since we apply options in order, make sure defaults are applied first
	options = append(options, defaultOptions...)

	if query.getStartingAfter() != nil {
		options = append(options, StartingAfter(Cursor(*query.getStartingAfter())))
	}

	if query.getEndingBefore() != nil {
		options = append(options, EndingBefore(Cursor(*query.getEndingBefore())))
	}

	if query.getLimit() != nil {
		limit, err := strconv.Atoi(*query.getLimit())
		if err != nil {
			return nil, ucerr.Friendlyf(err, "error parsing 'limit' argument")
		}
		options = append(options, Limit(limit))
	}

	if query.getFilter() != nil {
		options = append(options, Filter(*query.getFilter()))
	}

	if query.getSortKey() != nil {
		options = append(options, SortKey(Key(*query.getSortKey())))
	}

	if query.getSortOrder() != nil {
		options = append(options, SortOrder(Order(*query.getSortOrder())))
	}

	if query.getVersion() != nil {
		version, err := strconv.Atoi(*query.getVersion())
		if err != nil {
			return nil, ucerr.Friendlyf(err, "error parsing 'version' argument")
		}
		options = append(options, requestVersion(Version(version)))
	}

	pager, err := ApplyOptions(options...)
	if err != nil {
		return nil, ucerr.Friendlyf(err, "paginator settings are invalid: %s", ucerr.UserFriendlyMessage(err))
	}

	return pager, nil
}

// QueryParamsFromRequest creates a QueryParams struct from the query parameters in an http.Request
func QueryParamsFromRequest(r *http.Request) QueryParams {
	queryParams := QueryParams{}
	urlValues := r.URL.Query()

	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		queryParams.StartingAfter = &v
	}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		queryParams.EndingBefore = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		queryParams.Limit = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		queryParams.Filter = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		queryParams.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		queryParams.SortOrder = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		queryParams.Version = &v
	}

	return queryParams
}
