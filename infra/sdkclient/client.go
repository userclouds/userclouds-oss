package sdkclient

import (
	"fmt"
	"strings"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/request"
)

// Client represents a jsonclient that communicates with the UserClouds API
type Client struct {
	*jsonclient.Client
}

// New constructs a new UserClouds SDK client
func New(url, clientName string, opts ...jsonclient.Option) *Client {
	url = strings.TrimSuffix(url, "/")
	opts = append([]jsonclient.Option{
		jsonclient.Header(request.HeaderSDKVersion, sdkVersion),
		jsonclient.HeaderUserAgent(fmt.Sprintf("UserClouds %s Go SDK %s", clientName, sdkVersion)),
		jsonclient.RetryNetworkErrors(retryNetworkErrors),
	}, opts...)
	c := jsonclient.New(url, opts...)
	return &Client{c}
}
