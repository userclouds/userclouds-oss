package ucopensearch

import (
	"net/url"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
)

// Config holds config info for OpenSearch
type Config struct {
	URL        string               `yaml:"url" json:"url" validate:"notempty"`
	MaxResults int                  `yaml:"max_results" json:"max_results"`
	Region     region.MachineRegion `yaml:"region" json:"region" validate:"notempty"`
}

func (c *Config) extraValidate() error {
	if c.MaxResults < 0 {
		return ucerr.Friendlyf(nil, "MaxResults '%d' must be non-negative", c.MaxResults)
	}
	if _, err := url.Parse(c.URL); err != nil {
		return ucerr.Errorf("failed to parse OpenSearch URL '%s': %w", c.URL, err)
	}
	return nil
}

//go:generate genvalidate Config
