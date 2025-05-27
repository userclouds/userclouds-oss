package types

import (
	"context"
	"fmt"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// BatchProvisioner is used to execute Provisionables in a series of batches
type BatchProvisioner struct {
	batches []Provisionable
}

// NewBatchProvisioner create a BatchProvisioner, partitioning a list of Provisionables into batchSize-sized batches
func NewBatchProvisioner(name string, provs []Provisionable, batchSize int) (*BatchProvisioner, error) {
	if batchSize <= 0 {
		return nil, ucerr.Errorf("batchSize must be greater than 0")
	}

	var batches []Provisionable
	for i := 0; i < len(provs); i += batchSize {
		end := min(i+batchSize, len(provs))

		// run each batch separately to ensure we don't open too many DB connections
		batches = append(batches, *NewParallelProvisioner(provs[i:end], fmt.Sprintf("%s:BatchExec(%d)", name, i)))
	}

	return &BatchProvisioner{batches: batches}, nil
}

// Execute executes the specified list of provision operations for each batch
func (bp BatchProvisioner) Execute(ctx context.Context, pos []ProvisionOperation) error {
	if len(pos) == 0 {
		return ucerr.Errorf("no provision operations specified")
	}

	var combinedErrors error
	for i := range bp.batches {
		uclog.Infof(ctx, "Starting execution of provisioning batch %d/%d", i+1, len(bp.batches))
		for _, po := range pos {
			if err := po.Execute(ctx, bp.batches[i]); err != nil {
				combinedErrors = ucerr.Combine(combinedErrors, err)
			}
		}
		uclog.Infof(ctx, "Completed execution of provisioning batch %d/%d", i+1, len(bp.batches))
	}
	return ucerr.Wrap(combinedErrors)
}
