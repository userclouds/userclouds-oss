package workerclient

import (
	"context"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/worker"
)

// TypeHTTP defines the HTTP worker client type
const TypeHTTP Type = "http"

// HTTPClient is a client for talking to our localhost dev sqs service
type HTTPClient struct {
	url string
}

// NewHTTPClient returns a new HTTPClient
func NewHTTPClient(url string) HTTPClient {
	return HTTPClient{url: url}
}

// Send implements Client
func (c HTTPClient) Send(ctx context.Context, msg worker.Message) error {
	msg.SetSourceRegionIfNotSet()
	if err := msg.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	client := jsonclient.New(c.url)
	uclog.Debugf(ctx, "sending message %v to %s", msg.Task, c.url)
	return ucerr.Wrap(client.Post(ctx, "/", msg, nil, jsonclient.BypassRouting()))
}

func (c HTTPClient) String() string {
	return c.url
}
