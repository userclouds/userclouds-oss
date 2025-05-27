package responsewriter

import (
	"context"
	"net/http"

	"userclouds.com/infra/middleware"
)

type contextKey string

// Constants for SizeLogger constants
const (
	ctxPre  = contextKey("pregzip")
	ctxPost = contextKey("postgzip")
)

// SizeResponseWriter wraps the response writer to log the size of the response
type SizeResponseWriter struct {
	http.ResponseWriter

	Name  contextKey
	Bytes int
}

// Write implements io.Writer
func (srw *SizeResponseWriter) Write(b []byte) (int, error) {
	srw.Bytes += len(b)
	return srw.ResponseWriter.Write(b)
}

// PreCompressionSizeLogger creates a new middleware that logs the size of the response before compression
func PreCompressionSizeLogger() middleware.Middleware {
	return sizeLogger(ctxPre)
}

// CompressedSizeLogger creates a new middleware that logs the size of the response after compression
func CompressedSizeLogger() middleware.Middleware {
	return sizeLogger(ctxPost)
}

// sizeLogger creates a new middleware that logs the size of the response
func sizeLogger(name contextKey) middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			srw := &SizeResponseWriter{
				ResponseWriter: w,
				Name:           name,
			}

			ctx := context.WithValue(r.Context(), name, srw)

			// NB: this line is very important, because it ensures that the context values set by
			// responsewriter.sizeLogger are available to the rest of the request processing chain
			// (specifically `uclog.Middleware`) after the SizeLogger has finished processing an outbound
			// response. This is tested in `responsewriter.TestGzip`. The alternative to this approach
			// would be 2x `SizeLogger`` middleware, one outside of `uclog.Middleware` to set up the context,
			// and one inside to do the actual measurement (before it's logged)
			*r = *r.WithContext(ctx)

			next.ServeHTTP(srw, r)
		})
	})
}

// GetPreCompressionSize returns the size of the response before compression
func GetPreCompressionSize(ctx context.Context) int {
	return getSize(ctx, ctxPre)
}

// GetCompressedSize returns the size of the response after compression
func GetCompressedSize(ctx context.Context) int {
	return getSize(ctx, ctxPost)
}

// getSize returns the size of the response for the given named response writer
func getSize(ctx context.Context, name contextKey) int {
	srw, ok := ctx.Value(name).(*SizeResponseWriter)
	if !ok {
		return 0
	}
	return srw.Bytes
}
