package uctracemiddleware

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-http-utils/headers"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog/responsewriter"
	"userclouds.com/infra/uctrace"
)

func getConnInfo(r *http.Request) (clientIP string, clientPort *int, peerIP string, peerPort *int) {
	// Get IP addresses according to OpenTelemetry semantic conventions:
	// https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/attributes.md#clientserver-examples-using--networkpeer

	peerParts := strings.Split(r.RemoteAddr, ":")
	peerIP = peerParts[0]
	peerPort = nil
	if p, err := strconv.Atoi(peerParts[1]); err == nil {
		peerPort = &p
	}

	clientIP = peerIP
	// TODO: establish a more reliable way of getting the client's IP when we
	// might have multiple layers of reverse proxies. This current logic will
	// only get the peer IP as seen by the last reverse proxy, which will be
	// incorrect if we have multiple reverse proxies. (It works today because
	// we only have ALB without Cloudfront or anything else in front or
	// behind.)
	if forwardIPsStr := r.Header.Get(headers.XForwardedFor); forwardIPsStr != "" {
		forwardIPs := strings.Split(strings.ReplaceAll(forwardIPsStr, " ", ""), ",")
		clientIP = forwardIPs[len(forwardIPs)-1]
	}

	clientPort = peerPort
	if forwardPortStr := r.Header.Get("X-Forwarded-Port"); forwardPortStr != "" {
		if p, err := strconv.Atoi(forwardPortStr); err == nil {
			clientPort = &p
		}
	}

	return
}

func getServerInfo(r *http.Request) (serverIP string, serverPort *int) {
	addr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return "", nil
	}
	parts := strings.Split(addr.String(), ":")
	serverIP = parts[0]
	serverPort = nil
	if p, err := strconv.Atoi(parts[1]); err == nil {
		serverPort = &p
	}
	return
}

var uuidRegex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func getRequestPath(r *http.Request) string {
	if r.URL.Path != "" {
		return r.URL.Path
	}
	// For some weird reason, sometimes r.URL is empty,
	// particularly when EB sends /healthcheck requests
	return r.RequestURI
}

// getSimplifiedPath takes a request path and replaces UUIDs with "<uuid>" so
// that it is easier to search for traces for a particular route, without
// having IDs in the way
func getSimplifiedPath(path string) string {
	return uuidRegex.ReplaceAllString(path, "<uuid>")
}

// HTTPHandlerTraceMiddleware wraps HTTP handlers with OpenTelemetry tracing
func HTTPHandlerTraceMiddleware() middleware.Middleware {
	otelMiddleware := middleware.Func(otelhttp.NewMiddleware("uctracemiddleware", otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		return r.Method + " " + getSimplifiedPath(getRequestPath(r))
	})))
	addAttributesMiddleware := middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := uctrace.GetCurrentSpan(r.Context())
			// Set standard http attributes: https://github.com/open-telemetry/semantic-conventions/blob/main/docs/http/http-spans.md
			clientIP, clientPort, peerIP, peerPort := getConnInfo(r)
			span.SetAttributes(semconv.ClientAddress(clientIP))
			if clientPort != nil {
				span.SetAttributes(semconv.ClientPort(*clientPort))
			}
			span.SetAttributes(semconv.NetworkPeerAddress(peerIP))
			if peerPort != nil {
				span.SetAttributes(semconv.NetworkPeerPort(*peerPort))
			}
			serverIP, serverPort := getServerInfo(r)
			if serverIP != "" {
				span.SetAttributes(semconv.ServerAddress(serverIP))
			}
			if serverPort != nil {
				span.SetAttributes(semconv.ServerPort(*serverPort))
			}
			span.SetAttributes(semconv.URLScheme(r.URL.Scheme))
			span.SetAttributes(semconv.URLPath(getRequestPath(r)))
			if r.URL.RawQuery != "" {
				span.SetAttributes(semconv.URLQuery(r.URL.RawQuery))
			}
			span.SetAttributes(semconv.HTTPRequestMethodKey.String(r.Method))
			span.SetAttributes(semconv.UserAgentOriginal(r.UserAgent()))
			if cl := r.Header.Get(headers.ContentLength); cl != "" {
				if clInt, err := strconv.Atoi(cl); err == nil {
					span.SetAttributes(semconv.HTTPRequestBodySize(clInt))
				}
			}

			for k, v := range r.Header {
				span.SetAttributes(attribute.String("http.request.header."+k, v[0]))
			}

			if v := r.Header.Get(request.HeaderSDKVersion); v != "" {
				span.SetStringAttribute(uctrace.AttributeSdkVersion, v)
			}

			lrw := responsewriter.NewStatusResponseWriter(w)
			next.ServeHTTP(lrw, r)

			span.SetAttributes(semconv.HTTPResponseBodySize(responsewriter.GetCompressedSize(r.Context())))
			span.SetAttributes(semconv.HTTPResponseStatusCode(lrw.StatusCode))
			if lrw.StatusCode >= 500 {
				span.RecordError(ucerr.Errorf("HTTP %d", lrw.StatusCode))
			}
		})
	})
	return middleware.Func(func(next http.Handler) http.Handler {
		chain := middleware.Chain(otelMiddleware, addAttributesMiddleware).Apply(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			chain.ServeHTTP(w, r)
		})
	})
}
