package service

import (
	"fmt"
	"net/http"
	"os"

	"github.com/NYTimes/gziphandler"
	sentryhttp "github.com/getsentry/sentry-go/http"

	"userclouds.com/infra/async"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	loggingmiddleware "userclouds.com/infra/uclog/middleware"
	"userclouds.com/infra/uclog/responsewriter"
	"userclouds.com/infra/uctracemiddleware"
)

// BaseMiddleware is the middleware that should apply to everything in UserClouds
var BaseMiddleware = middleware.Chain(
	async.PanicRecoverMiddleware(),
	SentryMiddleware(),
	request.Middleware(),
	uctracemiddleware.HTTPHandlerTraceMiddleware(),
	loggingmiddleware.HTTPLoggerMiddleware(),
	responsewriter.CompressedSizeLogger(), // NB: these appear backwards because they are unwound in reverse order post-handler
	gzipMiddleware{},
	responsewriter.PreCompressionSizeLogger(),
	middleware.ReadAll(),
)

// this could easily live in a new package if needed, but for a few lines, not moving it yet
type gzipMiddleware struct{}

func (gzipMiddleware) Apply(next http.Handler) http.Handler {
	gzh := gziphandler.GzipHandler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gzh.ServeHTTP(w, r)
	})
}

// SentryMiddleware reports panics to sentry
func SentryMiddleware() middleware.Middleware {
	sh := sentryhttp.New(sentryhttp.Options{
		Repanic: true, // allows our logging to capture this as well
	})
	return middleware.Func(func(next http.Handler) http.Handler {
		// this is marginally janky, but we can't dig into the http.Handler
		// to make per-request decisions here, so this at least lets us make
		// per-environment decisions on this feature
		return sh.Handle(next)
	})
}

// GetMachineName returns the machine/host name for the current service (and pod name if in k8s)
func GetMachineName() string {
	if universe.Current().IsKubernetes() {
		return fmt.Sprintf("%s:%s", kubernetes.GetPodName(), kubernetes.GetNodeName())
	}
	host, err := os.Hostname()
	if err != nil {
		host = "N/A"
	}
	return host
}
