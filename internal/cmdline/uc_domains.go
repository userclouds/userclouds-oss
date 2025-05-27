package cmdline

import (
	"fmt"
	"net/url"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

const (
	baseDomain = "userclouds.com"
	devDomain  = "dev.userclouds.tools:3333"
)

// GetDomainForUniverse returns the domain for the given universe.
func GetDomainForUniverse(uv universe.Universe) (string, error) {
	if uv.IsCloud() {
		if uv.IsProd() {
			return baseDomain, nil
		}
		return fmt.Sprintf("%v.%s", uv, baseDomain), nil
	}
	if uv.IsDev() {
		return devDomain, nil
	}
	return "", ucerr.Errorf("universe %v is not supported for GetDomainForUniverse", uv)
}

// GetURLForUniverse returns the URL for the given universe and path and an optional hostname (prefixed to the domain, if n not empty).
func GetURLForUniverse(uv universe.Universe, path string, svc service.Service) (string, error) {
	domain, err := GetDomainForUniverse(uv)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if !svc.IsUndefined() {
		domain = fmt.Sprintf("%v.%s", svc, domain)
	}
	ucURL := url.URL{Host: domain, Path: path, Scheme: "https"}
	return ucURL.String(), nil
}
