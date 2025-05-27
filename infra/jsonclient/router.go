package jsonclient

import (
	"context"
	"net/http"
)

// this file / interface exists so that we can affinitize our local API client routing
// by injection (since we ship infra/jsonclient with all of our SDKs, but we don't need
// to route this way external to our DCs)

// RequestRouter allows jsonclient to internally reroute requests to specific hosts / clusters / pods
type RequestRouter interface {
	Reroute(context.Context, *http.Request)
}

// Router is jsonclient's RequestRouter
var Router RequestRouter

func init() {
	Router = &defaultRouter{}
}

// defaultRouter implements a no-op RequestRouter
type defaultRouter struct {
}

// Reroute implements RequestRouter
func (d defaultRouter) Reroute(ctx context.Context, r *http.Request) {
	// no-op
}
