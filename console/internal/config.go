package internal

import (
	"context"
	"net/url"

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
	"userclouds.com/internal/ucimage"
	"userclouds.com/internal/ucsentry"
)

// Config holds config data for the Console service
type Config struct {
	// TODO: rationalize these YAML names
	MountPoint               service.Endpoint  `yaml:"svc_listener" json:"svc_listener"`
	InternalServerMountPoint *service.Endpoint `yaml:"internal_server,omitempty" json:"internal_server,omitempty" validate:"allownil"`

	ConsoleURL      string    `yaml:"console_url,omitempty" json:"console_url,omitempty"`
	ConsoleTenantID uuid.UUID `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`

	// This is used for the base of tenant domains, like
	// "tenant.userclouds.com" for prod or
	// "tenant.dev.userclouds.tools:3333" for dev
	TenantSubDomain string `yaml:"tenant_sub_domain" json:"tenant_sub_domain" validate:"notempty"`
	// always "http" or "https"
	TenantProtocol string `yaml:"tenant_protocol" json:"tenant_protocol" validate:"notempty"`

	StaticAssetsPath string `yaml:"static_assets_path" json:"static_assets_path" validate:"notempty"`

	// At some point, console will probably be UI only and
	// talk to a company/tenant config service instead of directly
	// talking to the CompanyDB.
	CompanyDB ucdb.Config `yaml:"company_db" json:"company_db"`
	// TODO: this is needed so Console can generate per-tenant IDP DB
	// connection info. Longer term our DB provisioning code should generate this.
	LogDB ucdb.Config          `yaml:"log_db" json:"log_db"`
	Log   logtransports.Config `yaml:"logger" json:"logger"`
	Image *ucimage.Config      `yaml:"image,omitempty" json:"image,omitempty" validate:"allownil"`

	CacheConfig *cache.Config `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`

	WorkerClient      workerclient.Config  `yaml:"worker_client" json:"worker_client"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags,omitempty" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry,omitempty" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"allownil"`

	OnPremSQLShimPorts []int `yaml:"onprem_sqlshim_ports,omitempty" json:"onprem_sqlshim_ports,omitempty"`
}

//go:generate genvalidate Config

func (cfg *Config) extraValidate() error {
	if cfg.TenantProtocol != "http" && cfg.TenantProtocol != "https" {
		return ucerr.Errorf("TenantProtocol must be http or https, not %s", cfg.TenantProtocol)
	}
	consoleURL, err := url.Parse(cfg.ConsoleURL)
	if err != nil {
		return ucerr.Errorf("failed to parse ConsoleURL %s: %w", cfg.ConsoleURL, err)
	}
	if consoleURL.Path != "" {
		return ucerr.Errorf("ConsoleURL must not have a path, got %s", consoleURL.Path)
	}
	uv := universe.Current()
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

// LoadConfig loads the config for the console service
func LoadConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.Console, &cfg); err != nil {
		return nil, ucerr.Errorf("failed to load console service config : %w", err)
	}
	return &cfg, nil
}
