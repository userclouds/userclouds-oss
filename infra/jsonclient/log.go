package jsonclient

import (
	"context"
)

// Logger specifies a minimal interface to allow jsonclient to log errors.
type Logger interface {
	Debugf(ctx context.Context, format string, args ...any)
	Warningf(ctx context.Context, format string, args ...any)
}

var logger Logger

// RegisterLogger registers a logger to be used by jsonclient.
// Note this could eventually be extended to allow multiple etc,
// but right now this just allows us to break the uclog/jsonclient dependency
func RegisterLogger(l Logger) {
	logger = l
}

func (c *Client) logWarning(ctx context.Context, method, url, errorMsg string, code int) {
	if logger != nil && !c.options.stopLogging {
		logger.Warningf(ctx, "http %s request to URL '%s' returned error response (code %d): %s", method, url, code, errorMsg)
	}
}
