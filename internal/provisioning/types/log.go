package types

import (
	"context"

	"userclouds.com/infra/uclog"
)

// LogMessageProvisioner is a simple wrapper for outputing the debugging messages in provisioning
type LogMessageProvisioner struct {
	Named
	NoopClose
	Parallelizable
	mesg string
}

// NewLogMessageProvisioner returns a LogMessageProvisioner
func NewLogMessageProvisioner(mesg string) *LogMessageProvisioner {
	return &LogMessageProvisioner{
		Named:          NewNamed("Log " + mesg),
		Parallelizable: NewParallelizable(),
		mesg:           mesg,
	}
}

// Provision implements Provisionable
func (l LogMessageProvisioner) Provision(ctx context.Context) error {
	uclog.Debugf(ctx, "Provisioning: %s", l.mesg)
	return nil
}

// Validate implements Provisionable
func (l LogMessageProvisioner) Validate(ctx context.Context) error {
	uclog.Debugf(ctx, "Validating: %s", l.mesg)
	return nil
}

// Cleanup implements Provisionable
func (l LogMessageProvisioner) Cleanup(ctx context.Context) error {
	uclog.Debugf(ctx, "Cleaning up: %s", l.mesg)
	return nil
}
