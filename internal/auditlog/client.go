package auditlog

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/security"
)

type options struct {
	paginationOptions []pagination.Option
	jsonclientOptions []jsonclient.Option
}

// Option makes auditlog.Client extensible
type Option interface {
	applyOption(*options)
}

type optionFunc func(*options)

func (o optionFunc) applyOption(opts *options) {
	o(opts)
}

// Pagination is a wrapper around pagination.Option
func Pagination(opt ...pagination.Option) Option {
	return optionFunc(func(opts *options) {
		opts.paginationOptions = append(opts.paginationOptions, opt...)
	})
}

// JSONClient is a wrapper around jsonclient.Option
func JSONClient(opt ...jsonclient.Option) Option {
	return optionFunc(func(opts *options) {
		opts.jsonclientOptions = append(opts.jsonclientOptions, opt...)
	})
}

// Client represents a client to talk to the UserClouds UserEvent service
type Client struct {
	client  *jsonclient.Client
	options options
}

// NewClient constructs a new UserEvent client
func NewClient(url string, opts ...jsonclient.Option) (*Client, error) {
	// NB: we pass this directly here since we're already in internal, vs our service
	// clients where we require internal service callers to set this option
	optsN := append(opts, security.PassXForwardedFor())
	c := &Client{
		client: jsonclient.New(strings.TrimSuffix(url, "/"), optsN...),
	}
	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// ListEntriesResponse is the paginated response struct for listing columns
type ListEntriesResponse struct {
	Data []Entry `json:"data"`
	pagination.ResponseFields
}

// ListEntries gets all user events
func (c *Client) ListEntries(ctx context.Context, opts ...Option) (*ListEntriesResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.applyOption(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	requestURL := url.URL{
		Path: BasePathSegment + EntryPathSegment,
	}

	var res ListEntriesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", requestURL.String(), query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// GetEntry gets all user events
func (c *Client) GetEntry(ctx context.Context, id uuid.UUID) (*Entry, error) {
	var res Entry

	requestURL := url.URL{
		Path: fmt.Sprintf("%s%s/%s", BasePathSegment, EntryPathSegment, id),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// CreateEntry creates a new entry in the audit log
func (c *Client) CreateEntry(ctx context.Context, entry Entry) error {
	requestURL := url.URL{
		Path: BasePathSegment + EntryPathSegment,
	}

	return ucerr.Wrap(c.client.Post(ctx, requestURL.String(), &entry, nil))
}
