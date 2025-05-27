// NOTE: automatically generated file -- DO NOT EDIT

package serviceconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ServiceConfig) Validate() error {
	if err := o.MountPoint.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.InternalServerMountPoint != nil {
		if err := o.InternalServerMountPoint.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.StaticAssetsPath == "" {
		return ucerr.Friendlyf(nil, "ServiceConfig.StaticAssetsPath can't be empty")
	}
	if err := o.CompanyDB.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Log.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.WorkerClient != nil {
		if err := o.WorkerClient.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.ConsoleTenantID.IsNil() {
		return ucerr.Friendlyf(nil, "ServiceConfig.ConsoleTenantID can't be nil")
	}
	if o.ACME != nil {
		if err := o.ACME.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.CacheConfig != nil {
		if err := o.CacheConfig.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.FeatureFlagConfig != nil {
		if err := o.FeatureFlagConfig.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.Sentry != nil {
		if err := o.Sentry.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.Tracing != nil {
		if err := o.Tracing.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
