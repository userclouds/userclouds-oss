package datamapping

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/paths"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
)

type options struct {
	paginationOptions []pagination.Option
	jsonclientOptions []jsonclient.Option
}

// Client is the DataMapping client
type Client struct {
	client  *sdkclient.Client
	options options
}

// NewClient constructs a new IDP client
func NewClient(url string, opts ...Option) (*Client, error) {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	c := &Client{
		client:  sdkclient.New(url, "idp", options.jsonclientOptions...),
		options: options,
	}

	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// Option makes datamapping.Client extensible
type Option interface {
	apply(*options)
}

type optFunc func(*options)

func (o optFunc) apply(opts *options) {
	o(opts)
}

// Pagination is a wrapper around pagination.Option
func Pagination(opt ...pagination.Option) Option {
	return optFunc(func(opts *options) {
		opts.paginationOptions = append(opts.paginationOptions, opt...)
	})
}

// JSONClient is a wrapper around jsonclient.Option
func JSONClient(opt ...jsonclient.Option) Option {
	return optFunc(func(opts *options) {
		opts.jsonclientOptions = append(opts.jsonclientOptions, opt...)
	})
}

// CreateDataSourceRequest is the request body for creating a new data source
type CreateDataSourceRequest struct {
	DataSource DataSource `json:"datasource"`
}

// CreateDataSource creates a new data source
func (c *Client) CreateDataSource(ctx context.Context, ds DataSource) (*DataSource, error) {
	req := CreateDataSourceRequest{
		DataSource: ds,
	}

	var resp DataSource
	if err := c.client.Post(ctx, paths.CreateDataSourcePath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateDataSourceRequest is the request body for creating a new data source
type UpdateDataSourceRequest struct {
	DataSource DataSource `json:"datasource"`
}

// UpdateDataSource updates a data source
func (c *Client) UpdateDataSource(ctx context.Context, ds DataSource) (*DataSource, error) {
	req := UpdateDataSourceRequest{
		DataSource: ds,
	}

	var resp DataSource
	if err := c.client.Put(ctx, paths.UpdateDataSourcePath(ds.ID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListDataSourcesResponse is the paginated response from listing data sources.
type ListDataSourcesResponse struct {
	Data []DataSource `json:"data"`
	pagination.ResponseFields
}

// ListDataSources lists all the available data sources
func (c *Client) ListDataSources(ctx context.Context, opts ...Option) (*ListDataSourcesResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	var res ListDataSourcesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListDataSourcesPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// GetDataSource gets a data source by ID
func (c *Client) GetDataSource(ctx context.Context, dataSourceID uuid.UUID) (*DataSource, error) {
	var resp DataSource
	if err := c.client.Get(ctx, paths.GetDataSourcePath(dataSourceID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteDataSource deletes a data source by ID
func (c *Client) DeleteDataSource(ctx context.Context, dataSourceID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteDataSourcePath(dataSourceID), nil))
}

// CreateDataSourceElementRequest is the request body for creating a new data source
type CreateDataSourceElementRequest struct {
	DataSourceElement DataSourceElement `json:"element"`
}

// CreateDataSourceElement creates a new data source
func (c *Client) CreateDataSourceElement(ctx context.Context, dse DataSourceElement) (*DataSourceElement, error) {
	req := CreateDataSourceElementRequest{
		DataSourceElement: dse,
	}

	var resp DataSourceElement
	if err := c.client.Post(ctx, paths.CreateDataSourceElementPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateDataSourceElementRequest is the request body for creating a new data source
type UpdateDataSourceElementRequest struct {
	DataSourceElement DataSourceElement `json:"element"`
}

// UpdateDataSourceElement updates a data source
func (c *Client) UpdateDataSourceElement(ctx context.Context, dse DataSourceElement) (*DataSourceElement, error) {
	req := UpdateDataSourceElementRequest{
		DataSourceElement: dse,
	}

	var resp DataSourceElement
	if err := c.client.Put(ctx, paths.UpdateDataSourceElementPath(dse.ID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListDataSourceElementsResponse is the paginated response from listing data sources.
type ListDataSourceElementsResponse struct {
	Data []DataSourceElement `json:"data"`
	pagination.ResponseFields
}

// ListDataSourceElements lists all the available data source elements
func (c *Client) ListDataSourceElements(ctx context.Context, opts ...Option) (*ListDataSourceElementsResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	var res ListDataSourceElementsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListDataSourceElementsPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// GetDataSourceElement gets a data source element by ID
func (c *Client) GetDataSourceElement(ctx context.Context, dataSourceElementID uuid.UUID) (*DataSourceElement, error) {
	var resp DataSourceElement
	if err := c.client.Get(ctx, paths.GetDataSourceElementPath(dataSourceElementID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteDataSourceElement deletes a data source element by ID
func (c *Client) DeleteDataSourceElement(ctx context.Context, dataSourceElementID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteDataSourceElementPath(dataSourceElementID), nil))
}
