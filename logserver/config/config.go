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

// Config holds config data for the LogServer service
type Config struct {
	MountPoint               service.Endpoint  `yaml:"svc_listener" json:"svc_listener"`
	InternalServerMountPoint *service.Endpoint `yaml:"internal_server,omitempty" json:"internal_server,omitempty" validate:"allownil"`

	CompanyDB         ucdb.Config          `yaml:"company_db" json:"company_db"`
	DefaultLogDB      ucdb.Config          `yaml:"log_db" json:"log_db"`
	Log               logtransports.Config `yaml:"logger" json:"logger"`
	ConsoleTenantID   uuid.UUID            `yaml:"console_tenant_id" json:"console_tenant_id" validate:"notnil"`
	KinesisAWSRegion  string               `yaml:"kinesis_aws_region,omitempty" json:"kinesis_aws_region,omitempty"`
	CacheConfig       *cache.Config        `yaml:"cache,omitempty" json:"cache,omitempty" validate:"allownil"`
	Sentry            *ucsentry.Config     `yaml:"sentry,omitempty" json:"sentry,omitempty" validate:"allownil"`
	Tracing           *uctrace.Config      `yaml:"tracing,omitempty" json:"tracing,omitempty" validate:"allownil"`
	FeatureFlagConfig *featureflags.Config `yaml:"featureflags,omitempty" json:"featureflags,omitempty" validate:"allownil"`
}

//go:generate genvalidate Config

func (cfg *Config) extraValidate() error {
	uv := universe.Current()
	if uv.IsCloud() || uv.IsOnPrem() || uv.IsDev() {
		if cfg.InternalServerMountPoint == nil {
			return ucerr.Errorf("InternalServerMountPoint must be set in %v", uv)
		}
	}

	// KinesisAWSRegion is optional, but if set, it must be a valid region. if not set, LogServer won't accept raw logs
	if cfg.KinesisAWSRegion != "" {
		if rg := region.FromAWSRegion(cfg.KinesisAWSRegion); !region.IsValid(rg, uv) {
			return ucerr.Errorf("Invalid region for Kinesis: %v", cfg.KinesisAWSRegion)
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
