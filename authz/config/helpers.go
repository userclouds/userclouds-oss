package config

import (
	"context"

	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/yamlconfig"
)

// LoadAuthzConfig loads the config for the authz service
func LoadAuthzConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.AuthZ, &cfg); err != nil {
		return nil, ucerr.Errorf("failed to load authz service config : %w", err)
	}
	return &cfg, nil
}

// LoadCheckAttributeConfig loads the config for the checkattribute service
func LoadCheckAttributeConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.CheckAttribute, &cfg); err != nil {
		return nil, ucerr.Errorf("failed to load checkattribute service config : %w", err)
	}
	return &cfg, nil
}
