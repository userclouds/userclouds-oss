// package serviceconfig breaks an import cycle between companyconfig -> worker client -> worker -> companyconfig
// TODO (sgarrity 6/23): once we move "PlexConfig" (the per-tenant config for the service) from companyconfig to tenantdb,
//   we should move this back to plex.config like every other service

package serviceconfig

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctrace"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/ucsentry"
)

// ServiceConfig defines configuration for plex
// Each Plex service loads one service instance config + N tenant configs (multitenant).
//
// There are some uncertain/gray areas, mostly because they affect both IDP and Plex:
// - Auto account merging by email (e.g. social, username+password IFF emails is verified, etc)
// - Password strength
// - User profile schema
// The reason these are uncertain is that the IDPs need to be in sync with Plex on this, or
// in some cases the IDPs are actually the source of truth. But Plex needs to also sync these settings,
// since they affect UI and behavior in Plex itself. Supporting multiple IDPs may make
// some of these settings impossible, to the point where you can only support some features if all
// IDPs support it (and their settings are perfectly sync'd).
type ServiceConfig struct {
	MountPoint               service.Endpoint  `yaml:"svc_listener" json:"svc_listener"`
	InternalServerMountPoint *service.Endpoint `yaml:"internal_server,omitempty" json:"internal_server" validate:"allownil"`

	StaticAssetsPath string `yaml:"static_assets_path" json:"static_assets_path" validate:"notempty"`

	ConsoleURL string `yaml:"console_url,omitempty" json:"console_url,omitempty"`
	// CompanyDB is the config for the company database
	CompanyDB         ucdb.Config          `yaml:"company_db,omitempty" json:"company_db"`
	DisableEmail      bool                 `yaml:"disable_email,omitempty" json:"disable_email"`
	Log               logtransports.Config `yaml:"logger,omitempty" json:"logger"`
	WorkerClient      *workerclient.Config `yaml:"worker_client,omitempty" json:"worker_client"  validate:"allownil"`
	ConsoleTenantID   uuid.UUID            `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`
	ACME              *acme.Config         `yaml:"acme" json:"acme" validate:"allownil"`
	CacheConfig       *cache.Config        `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing" validate:"allownil"`
}

//go:generate gendbjson ServiceConfig

//go:generate genvalidate ServiceConfig

func (cfg *ServiceConfig) extraValidate() error {
	uv := universe.Current()
	if (uv.IsCloud() || uv.IsDev()) && cfg.ACME == nil {
		return ucerr.Errorf("ACME config is required in %v", uv)
	}
	if !uv.IsContainer() && cfg.WorkerClient == nil {
		return ucerr.Errorf("worker client config is required in non-container environment: %v", uv)
	}
	if cfg.IsConsoleEndpointDefined() {
		if _, err := cfg.GetConsoleEndpoint(); err != nil {
			return ucerr.Wrap(err)
		}
	} else if !uv.IsOnPremOrContainer() {
		return ucerr.Errorf("console endpoint is required in non-container environment: %v", uv)
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

// IsConsoleEndpointDefined returns true if the console endpoint is defined
func (cfg *ServiceConfig) IsConsoleEndpointDefined() bool {
	return cfg.ConsoleURL != ""
}

// GetConsoleEndpoint returns the console endpoint as a service.Endpoint
func (cfg *ServiceConfig) GetConsoleEndpoint() (*service.Endpoint, error) {
	if !cfg.IsConsoleEndpointDefined() {
		return nil, ucerr.New("console URL not configured")
	}
	ep, err := service.NewEndpointFromURLString(cfg.ConsoleURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &ep, nil
}
