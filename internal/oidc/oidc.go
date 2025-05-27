package oidc

import (
	"net/url"
	"os"
	"strings"

	"userclouds.com/infra/namespace/universe"
)

// IsUsercloudsIssued returns true if the token was issued by Userclouds.
func IsUsercloudsIssued(issuer string) bool {
	issURL, err := url.Parse(issuer)
	if err != nil {
		return false
	}
	uv := universe.Current()
	hostname := issURL.Hostname()
	if allowedHostSuffix := getHostSuffix(uv); allowedHostSuffix != "" {
		return strings.HasSuffix(hostname, allowedHostSuffix)
	}
	return false
}

func getHostSuffix(uv universe.Universe) string {
	if uv.IsCloud() {
		return ".userclouds.com"
	}
	if uv.IsDev() {
		return ".userclouds.tools"
	}
	if uv.IsTestOrCI() || uv.IsContainer() {
		return ".test.userclouds.tools"
	}
	if uv.IsOnPrem() {
		// helm/userclouds-on-prem/templates/_helpers.tpl
		return os.Getenv("UC_ON_PREM_CUSTOMER_DOMAIN")
	}
	return ""
}
