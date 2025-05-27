package config

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/dnsclient"
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

// Config describes worker config
type Config struct {
	MountPoint               service.Endpoint     `yaml:"svc_listener" json:"svc_listener"`
	InternalServerMountPoint *service.Endpoint    `yaml:"internal_server,omitempty" json:"internal_server,omitempty" validate:"allownil"`
	Log                      logtransports.Config `yaml:"logger" json:"logger"`

	ConsoleTenantID uuid.UUID   `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`
	CompanyDB       ucdb.Config `yaml:"company_db" json:"company_db"`
	LogDB           ucdb.Config `yaml:"log_db" json:"log_db"` // needed for create tenant / provisioning

	WorkerClient workerclient.Config `yaml:"worker_client" json:"worker_client"`

	DNS dnsclient.Config `yaml:"dns" json:"dns"`

	ACME              *acme.Config         `yaml:"acme" json:"acme" validate:"allownil"`
	CacheConfig       *cache.Config        `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags,omitempty" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry,omitempty" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"allownil"`

	OpenSearchConfig *ucopensearch.Config `yaml:"opensearch,omitempty" json:"opensearch,omitempty" validate:"allownil"`
}

func (cfg *Config) extraValidate() error {
	uv := universe.Current()
	if !uv.IsOnPremOrContainer() && cfg.ACME == nil {
		return ucerr.Errorf("ACME must be set in %v", uv)
	}
	if uv.IsOnPremOrContainer() {
		if cfg.OpenSearchConfig != nil {
			return ucerr.Errorf("OpenSearchConfig is not supported in %v", uv)
		}
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

// LoadConfig loads worker config
func LoadConfig() (*Config, error) {
	ctx := context.Background()
	var cfg Config
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.Worker, &cfg); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &cfg, nil
}
