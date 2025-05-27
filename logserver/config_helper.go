package logserver

import (
	"context"
	"slices"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/migrate"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/logserver/config"
)

// LoadConfig loads the config for the log server service
func LoadConfig(ctx context.Context) (*config.Config, error) {
	var cfg config.Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.LogServer, &cfg); err != nil {
		return nil, ucerr.Errorf("failed to load log server service config : %w", err)
	}
	// Delete the logserver transport from the config, since we get it from the base env
	// but we actually don't want to use it in the log server.
	for idx, tc := range cfg.Log.Transports {
		if tc.GetType() == logtransports.TransportTypeServer {
			cfg.Log.Transports = slices.Delete(cfg.Log.Transports, idx, idx+1)
			break
		}
	}
	return &cfg, nil
}

// GetServiceData returns information about this "service" used for DB migrations.
func GetServiceData(ctx context.Context) (*migrate.ServiceData, error) {
	cfg, err := LoadConfig(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &migrate.ServiceData{DBCfg: &cfg.DefaultLogDB}, nil
}
