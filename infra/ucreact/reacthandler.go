package ucreact

import (
	"context"
	"net/http"
	"net/url"

	"userclouds.com/infra/service"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
)

// MountStaticAssetsHandler mounts a handler that serves static assets from a given local file path via the provided URL path.
func MountStaticAssetsHandler(ctx context.Context, hb *builder.HandlerBuilder, localFilesPath, urlPath string) {
	uclog.Infof(ctx, "Serving static UI assets from %s via %s", localFilesPath, urlPath)
	hb.Handle(urlPath, service.BaseMiddleware.Apply(NewHandler(localFilesPath)))
}

// NewHandler serves a React webpack bundle which is *mostly* static content,
// except that paths which aren't found should get handled by React Router
// (client side routing).
// If a URL would refer to a file in the bundle, serve it, otherwise serve "/"
// which is index.html (the main app entry point). Then, the client JS code
// can figure out how to render the app based on the URL.
func NewHandler(fileServerPath string) http.Handler {
	fs := http.FileServer(http.Dir(fileServerPath))
	return NewFallbackIfNotFoundHandler(fs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Logic borrowed from http.StripPrefix; basically, just serve the root path (index.html).
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = new(url.URL)
		*r2.URL = *r.URL
		r2.URL.Path = "/"
		r2.URL.RawPath = "/"
		fs.ServeHTTP(w, r2)
	}))
}

// NewFallbackIfNotFoundHandler creates a hybrid handler that attempts to serve a
// path with one handler and, if 404 would be returned, throws the response away
// and uses a fallback handler.
func NewFallbackIfNotFoundHandler(h, notFoundHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		shimWriter := &fallbackIfNotFoundWriter{
			rw:              w,
			r:               r,
			header:          http.Header{},
			fallbackHandler: notFoundHandler,
			invokedFallback: false,
			wroteHeader:     false,
		}
		h.ServeHTTP(shimWriter, r)
	})
}

// fallbackIfNotFoundWriter overrides an HTTP response with that of a fallback
// handler if and only if the original response would be 404 (not found).
type fallbackIfNotFoundWriter struct {
	rw              http.ResponseWriter
	r               *http.Request
	header          http.Header
	fallbackHandler http.Handler
	invokedFallback bool
	wroteHeader     bool
}

func (fw *fallbackIfNotFoundWriter) Header() http.Header {
	if fw.wroteHeader && !fw.invokedFallback {
		// Already wrote header, primary handler has NOT (yet) failed;
		// this must be a trailer (or maybe just bad usage); either way, pass through.
		return fw.rw.Header()
	}

	// At this point, either the primary handler failed (404) and we already served
	// the fallback request (in which case the shadow header map won't be used)
	// or we haven't yet failed and the shadow header map will get copied to the final
	// response.
	return fw.header
}

func (fw *fallbackIfNotFoundWriter) WriteHeader(status int) {
	if fw.wroteHeader {
		// Ignore duplicate header write
		uclog.Warningf(fw.r.Context(), "superfluous response.WriteHeader call")
		return
	}
	fw.wroteHeader = true

	if status == http.StatusNotFound {
		// Invoke fallback handler to write directly to fnal output
		fw.fallbackHandler.ServeHTTP(fw.rw, fw.r)
		fw.invokedFallback = true
	} else {
		// Something other than a 404; copy from shadow header map to final output.
		for k, vs := range fw.header {
			for _, v := range vs {
				fw.rw.Header().Add(k, v)
			}
		}
		fw.rw.WriteHeader(status)
	}
}

func (fw *fallbackIfNotFoundWriter) Write(b []byte) (int, error) {
	// Haven't had to invoke the fallback, pass through to actual stream.
	if !fw.invokedFallback {
		return fw.rw.Write(b)
	}

	// Pretend like everything was ok even though we didn't write anything
	// (because the fallback was invoked, and that presumably wrote the response).
	return len(b), nil
}
