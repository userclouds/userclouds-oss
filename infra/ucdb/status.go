package ucdb

import (
	"context"

	"userclouds.com/infra/uclog"
)

// Status is the status of the database.
type Status struct {
	Status string `json:"status" yaml:"status"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

// GetDBStatus returns the status of the database.
func GetDBStatus(ctx context.Context, cfg *Config) Status {
	if cfg == nil {
		return Status{Status: "NA"}
	}

	if _, err := newDB(ctx, cfg, 1, 4, nil); err != nil {
		uclog.Errorf(ctx, "failed to connected to DB: %v", err)
		return Status{Status: "Fail", Error: "connect"}
	}
	return Status{Status: "OK"}
}
