package config

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/logtransports"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctrace"
	"userclouds.com/infra/workerclient"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/internal/ucsentry"
)

// SQLShimConfig holds config info for the sqlshim proxy, specifically which ports to listen on
// NB: currently these need to be mapped from the load balancer manually (in helm chart) and updated here
type SQLShimConfig struct {
	MySQLPorts    []int `yaml:"mysql_ports" json:"mysql_ports"`
	PostgresPorts []int `yaml:"postgres_ports" json:"postgres_ports"`
	//For EKS only, we will do health check on this port
	HealthCheckPort *int `yaml:"health_check_port" json:"health_check_port" validate:"allownil"`
}

//go:generate genvalidate SQLShimConfig

func (c *SQLShimConfig) extraValidate() error {
	if c.MySQLPorts == nil && c.PostgresPorts == nil {
		return ucerr.Errorf("At least one of MySQLPorts or PostgresPorts must be set")
	}
	uv := universe.Current()
	if uv.IsOnPrem() && c.HealthCheckPort == nil {
		// TODO: when we move proxy for our cloud env to run in k8s/EKS we also need to error out if HealthCheckPort is not set
		return ucerr.Errorf("HealthCheckPort must be set in '%v'", uv)
	}
	return nil
}

// Config holds config info for IDP
type Config struct {
	MountPoint               service.Endpoint  `yaml:"svc_listener" json:"svc_listener"`
	SQLShimConfig            *SQLShimConfig    `yaml:"sqlshim_config,omitempty" json:"sqlshim_config,omitempty" validate:"allownil"`
	InternalServerMountPoint *service.Endpoint `yaml:"internal_server,omitempty" json:"internal_server,omitempty" validate:"allownil"`

	CompanyDB ucdb.Config `yaml:"company_db" json:"company_db"`

	Log logtransports.Config `yaml:"logger" json:"logger"`

	ConsoleTenantID   uuid.UUID            `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`
	CacheConfig       *cache.Config        `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags,omitempty" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry,omitempty" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"allownil"`

	DataImportConfig *DataImportConfig    `yaml:"data_import_config,omitempty" json:"data_import_config,omitempty" validate:"allownil"`
	WorkerClient     *workerclient.Config `yaml:"worker_client,omitempty" json:"worker_client,omitempty" validate:"allownil"`

	OpenSearchConfig *ucopensearch.Config `yaml:"opensearch,omitempty" json:"opensearch,omitempty" validate:"allownil"`
}

// GetShimPorts returns the proxy ports for the sqlshim proxies
func (cfg *Config) GetShimPorts() []int {
	ports := []int{}
	if cfg.SQLShimConfig == nil {
		return ports
	}
	if cfg.SQLShimConfig.MySQLPorts != nil {
		ports = append(ports, cfg.SQLShimConfig.MySQLPorts...)
	}
	if cfg.SQLShimConfig.PostgresPorts != nil {
		ports = append(ports, cfg.SQLShimConfig.PostgresPorts...)
	}
	return ports
}

func (cfg *Config) extraValidate() error {
	uv := universe.Current()
	if uv.IsOnPremOrContainer() && cfg.OpenSearchConfig != nil {
		return ucerr.Errorf("OpenSearchConfig is not supported in %v", uv)
	}
	if uv.IsCloud() || uv.IsOnPrem() || uv.IsDev() {
		if cfg.InternalServerMountPoint == nil {
			return ucerr.Errorf("InternalServerMountPoint must be set in %v", uv)
		}
	}
	if uv.IsCloud() {
		if cfg.Sentry == nil {
			return ucerr.Errorf("Sentry must be set in %v", uv)
		}
		if cfg.CacheConfig == nil {
			return ucerr.Errorf("CacheConfig must be set in %v", uv)
		}
	}
	return nil
}

//go:generate genvalidate Config

// DataImportConfig holds config info for data import
type DataImportConfig struct {
	DataImportS3Bucket            string `yaml:"data_import_s3_bucket" json:"data_import_s3_bucket" validate:"notempty"`
	PresignedURLExpirationMinutes int    `yaml:"url_expiration_minutes" json:"url_expiration_minutes"`
}

//go:generate genvalidate DataImportConfig

// LoadConfig loads the config for the userstore (idp) service
func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.IDP, &cfg); err != nil {
		return nil, ucerr.Errorf("failed to load userstore(idp) service config : %w", err)
	}
	return &cfg, nil
}

// LoadDBProxyConfig loads the config for the userstore (idp) service running in DBProxy mode
func LoadDBProxyConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	loadParams := yamlconfig.GetLoadParams(true, false, false)
	if err := yamlconfig.LoadEnv(ctx, "dbproxy", &cfg, loadParams); err != nil {
		return nil, ucerr.Errorf("failed to load userstore(idp) service config : %w", err)
	}
	return &cfg, nil
}

// SearchUpdateConfig is the configuration for updating a search index
type SearchUpdateConfig struct {
	SearchCfg    *ucopensearch.Config
	WorkerClient workerclient.Client
}
