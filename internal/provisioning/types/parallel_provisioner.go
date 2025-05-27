package types

import (
	"context"
	"fmt"
	"sync"
	"time"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const numWorkerThreads int = 10

// ParallelProvisioner is a Provisionable that runs a batch of Provisionables in parallel as possible
// The batch may consist of Provisionable that request both serial and parallel execution.
// For example a batch [S1, S2, P1, P2, P3, S3, P4, P5, S5] where "S" indicates serial execution and "P" indicates parallel
// will be executed as follows: [S1] -> [S2] -> [P1 | P2 | P3 | S4 ] -> [P4 | P5 | S5], which means there will be 4 distinct
// stages execution and items within a stage will run in parallel
type ParallelProvisioner struct {
	Named
	Parallelizable
	provs []Provisionable
}

// NewParallelProvisioner returns a ParallelProvisioner that allows all operations
// to be parallelized
func NewParallelProvisioner(provs []Provisionable, name string) *ParallelProvisioner {
	return &ParallelProvisioner{
		Named:          NewNamed(name),
		Parallelizable: AllParallelizable(),
		provs:          provs,
	}
}

// NewRestrictedParallelProvisioner returns a ParallelProvisioner that allows the specified
// operations to be parallelized
func NewRestrictedParallelProvisioner(
	provs []Provisionable,
	name string,
	pos ...ProvisionOperation,
) *ParallelProvisioner {
	return &ParallelProvisioner{
		Named:          NewNamed(name),
		Parallelizable: NewParallelizable(pos...),
		provs:          provs,
	}
}

// Provision implements Provisionable
func (pp ParallelProvisioner) Provision(ctx context.Context) error {
	return ucerr.Wrap(pp.performOperations(ctx, Provision))
}

// Validate implements Provisionable
func (pp ParallelProvisioner) Validate(ctx context.Context) error {
	return ucerr.Wrap(pp.performOperations(ctx, Validate))
}

// Cleanup implements Provisionable
func (pp ParallelProvisioner) Cleanup(ctx context.Context) error {
	return ucerr.Wrap(pp.performOperations(ctx, Cleanup))
}

// Close calls close for each Provisionable
func (pp ParallelProvisioner) Close(ctx context.Context) error {
	return ucerr.Wrap(pp.performOperations(ctx, Close))
}

// getItems returns the list of Provisionable inside ParallelProvisioner so that it can be unrolled for execution
func (pp ParallelProvisioner) getItems() []Provisionable {
	return pp.provs
}

// batchOperations combines all Provisionables in a collection in a set of batches for parallel execution that
// preserves execution order dependencies between items in the collection
func (pp ParallelProvisioner) batchOperations(
	ctx context.Context,
	provsIn []Provisionable,
	po ProvisionOperation,
) [][]Provisionable {
	batches := [][]Provisionable{}
	provs := [][]Provisionable{provsIn}

	batchLengths := " "
	for len(provs) > 0 {
		currentBatch := []Provisionable{}
		nextBatch := [][]Provisionable{}
		for p := range provs {
			provsQ := provs[p]
			pp.batchOperationsInner(ctx, provsQ, &currentBatch, &nextBatch, po)
		}
		if len(currentBatch) > 0 {
			// Perfrom remaining operations in parallel
			batches = append(batches, currentBatch)
			batchLengths = fmt.Sprintf("%s %d", batchLengths, len(currentBatch))
		}
		provs = nextBatch
	}
	uclog.Debugf(ctx, "Divided provisioning work into %d batches with length %s", len(batches), batchLengths)
	return batches
}

func (pp ParallelProvisioner) batchOperationsInner(
	ctx context.Context,
	provsQ []Provisionable,
	currentBatch *[]Provisionable,
	nextBatch *[][]Provisionable,
	po ProvisionOperation,
) {
	for i := range provsQ {
		// If we encounter a Provisionable that needs serial execution - we put it into the current batch and stop
		if !provsQ[i].IsExecutableInParallel(ctx, po) {
			*currentBatch = append(*currentBatch, provsQ[i])
			if i+1 < len(provsQ) {
				*nextBatch = append(*nextBatch, provsQ[i+1:])
			}
			break
		}
		// The current Provisionable is a collection so append all the Provisionables in it to the current batch until one requires
		// a serial Provisionable is encountered
		if p, ok := provsQ[i].(*ParallelProvisioner); ok {
			items := p.getItems()
			// TODO add a check for parallel/serial execution to get rid of Wrappers
			pp.batchOperationsInner(ctx, items, currentBatch, nextBatch, po)
		} else {
			// The current Provisionable can be executed in parallel but is not a collection so add it to the current batch and move to the next one
			*currentBatch = append(*currentBatch, provsQ[i])
		}
	}
}

// performOperations performs given operation on the contained Provisionables, running the operation in parallel if applicable
func (pp ParallelProvisioner) performOperations(ctx context.Context, po ProvisionOperation) error {
	batches := [][]Provisionable{}
	if pp.IsExecutableInParallel(ctx, po) {
		batches = pp.batchOperations(ctx, pp.provs, po)
	} else {
		for i := range pp.provs {
			batchesPerProv := pp.batchOperations(ctx, []Provisionable{pp.provs[i]}, po)
			batches = append(batches, batchesPerProv...)
		}
	}

	// Short circuit if there is nothing to execute in parallel
	if len(batches) != 0 {
		for _, currentBatch := range batches {
			// Perfrom  operations in a batch in parallel
			wg := sync.WaitGroup{}
			var combErr error
			m := sync.Mutex{}
			perThreadLoad := (len(currentBatch) + numWorkerThreads) / numWorkerThreads
			startIndex := 0
			for range numWorkerThreads {
				if startIndex < len(currentBatch) {
					endBatch := min(startIndex+perThreadLoad, len(currentBatch))
					provsThread := currentBatch[startIndex:endBatch]

					startIndex = endBatch

					wg.Add(1)
					go func(provsBatch []Provisionable) {
						for i := range provsBatch {
							startTime := time.Now().UTC()
							uclog.Verbosef(ctx, "Performing op %v on %s", po, provsBatch[i].Name())
							if err := po.Execute(ctx, provsBatch[i]); err != nil {
								uclog.Errorf(ctx, "Failed op %v on %s: %v", po, provsBatch[i].Name(), err)
								m.Lock()
								combErr = ucerr.Combine(combErr, ucerr.Errorf("Provisionable %s failed: %w", provsBatch[i].Name(), err))
								m.Unlock()
							}
							uclog.Verbosef(ctx, "Finished op %v on %s in %v", po, provsBatch[i].Name(), time.Now().UTC().Sub(startTime))
						}
						wg.Done()
					}(provsThread)
				}
			}
			wg.Wait()

			if combErr != nil {
				return ucerr.Wrap(combErr)
			}
		}
	}
	return nil
}
