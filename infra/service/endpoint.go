package service

import (
	"fmt"
	"net/url"
	"strconv"

	"userclouds.com/infra/ucerr"
)

// Endpoint describes a service endpoint used for multiple purposes:
// 1. Setting up a listener for handling incoming connections, or
// 2. Describing an externally-reachable endpoint for clients.
type Endpoint struct {
	Protocol string `yaml:"protocol" json:"protocol"`
	Host     string `yaml:"host" json:"host"`
	Port     string `yaml:"port" json:"port"`
}

// NewEndpointFromURLString constructs an Endpoint from a URL string
func NewEndpointFromURLString(s string) (Endpoint, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Endpoint{}, ucerr.Wrap(err)
	}
	return NewEndpointFromURL(u), nil
}

// NewEndpointFromURL constructs an Endpoint from a url.URL
func NewEndpointFromURL(u *url.URL) Endpoint {
	return Endpoint{
		Protocol: u.Scheme,
		Host:     u.Hostname(),
		Port:     u.Port(),
	}
}

// Validate implements Validateable
func (s *Endpoint) Validate() error {
	if s.Protocol != "http" && s.Protocol != "https" {
		return ucerr.Errorf("Endpoint.Protocol must be http or https, not '%s'", s.Protocol)
	}
	if s.Host == "" {
		return ucerr.New("Endpoint.Host can't be empty")
	}
	if s.Port == "" {
		return ucerr.New("Endpoint.Port can't be empty")
	}
	port, err := strconv.ParseInt(s.Port, 10, 32)
	if err != nil {
		return ucerr.Wrap(err)
	} else if port < 0 || port >= 65536 {
		// NOTE: we allow 0 because for configuring a listener it may be valid.
		return ucerr.New("Endpoint.Port must be in the range [0,65536)")
	}
	return nil
}

// HostAndPort simply returns [host]:[port] because we use it many places
func (s *Endpoint) HostAndPort() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

// BaseURL returns [protocol]://[host]:[port] as a convenience, unless
// port is standard for the protocol in which case it is elided.
// We also elide if port is unspecified, indicating default
func (s *Endpoint) BaseURL() string {
	if (s.Protocol == "http" && s.Port == "80") || (s.Protocol == "https" && s.Port == "443") ||
		(s.Port == "0") || (s.Port == "") {
		// Omit port if using default.
		return fmt.Sprintf("%s://%s", s.Protocol, s.Host)
	}
	return fmt.Sprintf("%s://%s", s.Protocol, s.HostAndPort())
}

// URL returns [protocol]://[host]:[port] as a URL object, but removes port if it's the default for a scheme.
func (s *Endpoint) URL() *url.URL {
	if (s.Protocol == "http" && s.Port == "80") || (s.Protocol == "https" && s.Port == "443") {
		// Omit port if using default.
		return &url.URL{
			Scheme: s.Protocol,
			Host:   s.Host,
		}
	}
	return &url.URL{
		Scheme: s.Protocol,
		Host:   s.HostAndPort(),
	}
}
