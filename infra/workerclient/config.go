package workerclient

import (
	"fmt"
	"net/url"

	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
)

// Type is the type of worker client we're using
type Type string

// Config defines worker client config
type Config struct {
	Type Type   `yaml:"type" json:"type" validate:"notempty"`
	URL  string `yaml:"url" json:"url"`
}

func (a *Config) extraValidate() error {
	if a.Type == TypeTest {
		return nil
	}
	queueURL := a.GetURL()
	if queueURL == "" {
		return ucerr.Friendlyf(nil, "Config.URL can't be empty for worker client type %s", a.Type)
	}

	if _, err := url.Parse(queueURL); err != nil {
		return ucerr.Friendlyf(err, "Config.URL is not a valid URL: %s", queueURL)
	}
	return nil
}

// GetURL returns the URL to use for the worker client
func (a *Config) GetURL() string {
	if a.Type == TypeHTTP {
		if a.URL != "" {
			return a.URL
		}
		host := kubernetes.GetHostForService(service.Worker)
		return fmt.Sprintf("http://%s", host)
	}
	return a.URL
}

//go:generate genvalidate Config
