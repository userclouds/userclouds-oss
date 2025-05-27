package config

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/ucsentry"
)

// Config holds config data for the AuthZ service
type Config struct {
	MountPoint               service.Endpoint  `yaml:"svc_listener" json:"svc_listener"`
	InternalServerMountPoint *service.Endpoint `yaml:"internal_server,omitempty" json:"internal_server,omitempty" validate:"allownil"`

	CompanyDB ucdb.Config          `yaml:"company_db" json:"company_db"`
	Log       logtransports.Config `yaml:"logger" json:"logger"`

	ConsoleTenantID   uuid.UUID            `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`
	CacheConfig       *cache.Config        `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags,omitempty" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry,omitempty" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"allownil"`

	// CheckAttribute specific config
	CheckAttributeServiceMap map[uuid.UUID][]RegionalCheckAttributeConfig `yaml:"check_attribute_service_map,omitempty" json:"check_attribute_service_map,omitempty"` // for authz service to read
}

// RegionalCheckAttributeConfig holds checkattribute connection info for a specific region
type RegionalCheckAttributeConfig struct {
	Region      region.MachineRegion `json:"region" yaml:"region"`
	ServiceName string               `json:"service_name" yaml:"service_name" validate:"notempty"`
}

func (cfg *Config) extraValidate() error {
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

// GetHandledTenants returns the list of tenants that are handled by the check attribute service for a given service instance name and region
func (cfg *Config) GetHandledTenants(serviceName string, rg region.MachineRegion) []uuid.UUID {
	handledTenants := make([]uuid.UUID, 0, len(cfg.CheckAttributeServiceMap))
	for tenantID, regionalConfigs := range cfg.CheckAttributeServiceMap {
		for _, regionalCfg := range regionalConfigs {
			if regionalCfg.ServiceName == serviceName && regionalCfg.Region == rg {
				handledTenants = append(handledTenants, tenantID)
				break
			}
		}
	}
	return handledTenants
}

//go:generate genvalidate Config
