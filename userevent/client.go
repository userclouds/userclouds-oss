package userevent

import (
	"context"
	"net/url"
	"strings"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// Client represents a client to talk to the UserClouds UserEvent service
type Client struct {
	client *jsonclient.Client
}

// NewClient constructs a new UserEvent client
func NewClient(url string, opts ...jsonclient.Option) (*Client, error) {
	c := &Client{
		client: jsonclient.New(strings.TrimSuffix(url, "/"), opts...),
	}
	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// ListEventsResponse is the response to the ListEvents API
type ListEventsResponse struct {
	Data []UserEvent `json:"data"`
	pagination.ResponseFields
}

// ListEvents gets all user events
func (c *Client) ListEvents(ctx context.Context, opts ...pagination.Option) (*ListEventsResponse, error) {
	pager, err := pagination.ApplyOptions(opts...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var res ListEventsResponse
	query := pager.Query()
	requestURL := url.URL{
		Path:     "/userevent/events",
		RawQuery: query.Encode(),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// ListEventsForUserAlias gets all user events for a specified user
func (c *Client) ListEventsForUserAlias(ctx context.Context, userAlias string, opts ...pagination.Option) (*ListEventsResponse, error) {
	pager, err := pagination.ApplyOptions(opts...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var res ListEventsResponse
	query := pager.Query()
	query.Add("user_alias", userAlias)
	requestURL := url.URL{
		Path:     "/userevent/events",
		RawQuery: query.Encode(),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// ReportEventsRequest is the request accepted by the ReportEvents API
type ReportEventsRequest struct {
	Events []UserEvent `json:"events"`
	// TODO: support for counters and other signal types
}

// ReportEvents reports an array/batch of events
func (c *Client) ReportEvents(ctx context.Context, events []UserEvent) error {
	req := ReportEventsRequest{Events: events}

	requestURL := url.URL{
		Path: "/userevent/events",
	}

	return ucerr.Wrap(c.client.Post(ctx, requestURL.String(), &req, nil))
}
