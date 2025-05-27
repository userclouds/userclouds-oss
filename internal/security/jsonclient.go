package security

import (
	"context"
	"strings"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra/jsonclient"
)

// PassXForwardedFor is an easy option to allow jsonclient to forward XFF headers
func PassXForwardedFor() jsonclient.Option {
	return jsonclient.PerRequestHeader(func(ctx context.Context) (string, string) {
		sc := GetSecurityStatus(ctx)
		if sc == nil || len(sc.IPs) == 0 {
			return "", ""
		}
		ips := strings.Join(sc.IPs, ",")
		return headers.XForwardedFor, ips
	})
}
