package types_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/provisioning/types"
)

type SimpleProvisionable struct {
	Provisioned bool
	Validated   bool
	Cleanedup   bool
	Closed      bool
	N           string
}

func (s *SimpleProvisionable) Name() string {
	return s.N
}
func (s *SimpleProvisionable) Provision(context.Context) error {
	s.Provisioned = true
	return nil
}
func (s *SimpleProvisionable) Validate(context.Context) error {
	s.Validated = true
	return nil
}
func (s *SimpleProvisionable) Cleanup(context.Context) error {
	s.Cleanedup = true
	return nil
}

// IsExecutableInParallel returns true if this Provisionable can be executed in parallel with other Provisionable at the same level for a given operation
// if this function returns false the Provisionable should executed serially
func (s *SimpleProvisionable) IsExecutableInParallel(ctx context.Context, op types.ProvisionOperation) bool {
	return false
}

// Close cleans up resources that maybe used by a Provisionable object it should be always called once the Provisionable object is no longer needed
func (s *SimpleProvisionable) Close(context.Context) error {
	return nil
}

var results []string = []string{}

var resultsLock sync.Mutex

type TrackingProvisionable struct {
	ID int
	b  int
	P  bool
	n  string
}

func (s *TrackingProvisionable) Name() string {
	return s.n
}

func (s *TrackingProvisionable) Provision(context.Context) error {
	resultsLock.Lock()
	defer resultsLock.Unlock()
	results = append(results, fmt.Sprintf("%d_%d_%v", s.ID, s.b, types.Provision))
	return nil
}
func (s *TrackingProvisionable) Validate(context.Context) error {
	resultsLock.Lock()
	defer resultsLock.Unlock()
	results = append(results, fmt.Sprintf("%d_%d_%v", s.ID, s.b, types.Validate))
	return nil
}
func (s *TrackingProvisionable) Cleanup(context.Context) error {
	resultsLock.Lock()
	defer resultsLock.Unlock()
	results = append(results, fmt.Sprintf("%d_%d_%v", s.ID, s.b, types.Cleanup))
	return nil
}

func (s *TrackingProvisionable) IsExecutableInParallel(ctx context.Context, op types.ProvisionOperation) bool {
	return s.P
}

func (s *TrackingProvisionable) Close(context.Context) error {
	resultsLock.Lock()
	defer resultsLock.Unlock()
	results = append(results, fmt.Sprintf("%d_%d_%v", s.ID, s.b, types.Close))
	return nil
}

func (s *TrackingProvisionable) GetIDForOP(op types.ProvisionOperation) string {
	return fmt.Sprintf("%d_%d_%v", s.ID, s.b, op)
}

func GetIDFromResult(t *testing.T, s string, op types.ProvisionOperation) int {
	sb := strings.Split(s, "_")
	id, err := strconv.Atoi(sb[0])
	assert.NoErr(t, err)
	return id
}

func GetBatchFromResult(t *testing.T, s string, op types.ProvisionOperation) int {
	sb := strings.Split(s, "_")
	id, err := strconv.Atoi(sb[1])
	assert.NoErr(t, err)
	return id
}

func GetOpFromResult(t *testing.T, s string, op types.ProvisionOperation) string {
	sb := strings.Split(s, "_")
	return sb[2]
}
func generateProvisionableObjects(count int, batchid int) []types.Provisionable {
	provs := make([]types.Provisionable, count)
	for i := range provs {
		provs[i] = &TrackingProvisionable{i, batchid, true, fmt.Sprintf("%d", i)}
	}
	return provs
}

type DependencyCheckProvisionable struct {
	provisionDeps []int
	validateDeps  []int
	items         []types.Provisionable
	t             *testing.T
	p             types.Provisionable
}

func (s *DependencyCheckProvisionable) Name() string {
	return s.p.Name()
}
func (s *DependencyCheckProvisionable) Provision(ctx context.Context) error {
	for _, i := range s.provisionDeps {
		tp := (s.items[i]).(*TrackingProvisionable)
		expected := tp.GetIDForOP(types.Provision)
		found := false
		resultsLock.Lock()
		for _, r := range results {
			if r == expected {
				found = true
			}
		}
		resultsLock.Unlock()
		assert.True(s.t, found)

	}
	return s.p.Provision(ctx)
}
func (s *DependencyCheckProvisionable) Validate(ctx context.Context) error {
	for _, i := range s.validateDeps {
		tp := (s.items[i]).(*TrackingProvisionable)
		expected := tp.GetIDForOP(types.Provision)
		found := false
		resultsLock.Lock()
		for _, s := range results {
			if s == expected {
				found = true
			}
		}
		resultsLock.Unlock()
		assert.True(s.t, found)
	}
	return s.p.Validate(ctx)
}
func (s *DependencyCheckProvisionable) Cleanup(context.Context) error {
	return nil
}
func (s *DependencyCheckProvisionable) Close(context.Context) error {
	return nil
}

func (s *DependencyCheckProvisionable) IsExecutableInParallel(ctx context.Context, op types.ProvisionOperation) bool {
	return s.p.IsExecutableInParallel(ctx, op)
}

func validateProvisionableObjects(ctx context.Context, t *testing.T, po types.ProvisionOperation, provs []types.Provisionable) {
	t.Helper()

	// Check that every provisionable has run and that no serialized item is preceded by a later item in the same batch
	for resultIndex, result := range results {
		if GetOpFromResult(t, result, po) == string(po) {
			found := false
			for provIndex := range provs {
				if _, ok := (provs[provIndex]).(*types.ParallelProvisioner); ok {
					continue
				}
				tp := (provs[provIndex]).(*TrackingProvisionable)
				if tp.GetIDForOP(po) == result {
					found = true
					if !tp.IsExecutableInParallel(ctx, po) {
						for i := range resultIndex {
							if GetBatchFromResult(t, results[i], po) == GetBatchFromResult(t, result, po) {
								if GetOpFromResult(t, results[i], po) == string(po) {
									assert.True(t, GetIDFromResult(t, results[i], po) < GetIDFromResult(t, result, po))
								}
							}
						}
					}
					continue
				}
			}
			assert.True(t, found)
		}
	}
}

func createBatches(provs []types.Provisionable, batchSize int) [][]types.Provisionable {
	var batches [][]types.Provisionable
	for i := 0; i < len(provs); i += batchSize {
		end := min(i+batchSize, len(provs))

		batches = append(batches, provs[i:end])
	}

	return batches
}

func validateProvisionableObjectsOpOrder(
	ctx context.Context,
	t *testing.T,
	batches [][]types.Provisionable,
	batchSize int,
	opOrder []types.ProvisionOperation,
) {
	t.Helper()

	// Convert the op order into easy to look up numeric map
	opOrderMap := make(map[string]int)
	batchOpCounts := make(map[string]int)
	for i := range opOrder {
		opOrderMap[string(opOrder[i])] = i
		batchOpCounts[string(opOrder[i])] = 0
	}

	opCount := 0
	opIndex := 0
	for _, batch := range batches {
		// Check if the batch was correctly split
		assert.Equal(t, len(batch) <= batchSize, true)
		opCount = opCount + len(batch)

		// For each operation
		for _, op := range opOrder {
			// For length of items in the batch
			for range batch {
				currOp := string(GetOpFromResult(t, results[opIndex], op))
				currOpOrderIndex := opOrderMap[currOp]
				assert.Equal(t, opOrderMap[string(op)], currOpOrderIndex)
				for i, opR := range opOrder {
					if i < currOpOrderIndex {
						assert.Equal(t, batchOpCounts[string(opR)], opCount)
					}
				}
				batchOpCounts[currOp]++
				opIndex++
			}
		}

	}
}

func executeAndValidateProvisionables(t *testing.T, ctx context.Context, provs []types.Provisionable, po types.ProvisionOperation, parallel bool) {
	t.Helper()

	var pp *types.ParallelProvisioner
	if parallel {
		pp = types.NewParallelProvisioner(provs, "test")
	} else {
		pp = types.NewRestrictedParallelProvisioner(provs, "test")
	}

	results = []string{}
	assert.NoErr(t, po.Execute(ctx, pp))
	validateProvisionableObjects(ctx, t, po, provs)
}

func TestProvisionable(t *testing.T) {
	ctx := context.Background()
	t.Run("TestPerformOperation", func(t *testing.T) {
		eP := SimpleProvisionable{N: "test"}

		assert.NoErr(t, eP.Provision(ctx))
		assert.Equal(t, SimpleProvisionable{true, false, false, false, "test"}, eP)
		assert.NoErr(t, eP.Validate(ctx))
		assert.Equal(t, SimpleProvisionable{true, true, false, false, "test"}, eP)
		assert.NoErr(t, eP.Cleanup(ctx))
		assert.Equal(t, SimpleProvisionable{true, true, true, false, "test"}, eP)

	})

	t.Run("TestPerformOperations", func(t *testing.T) {
		eP := generateProvisionableObjects(100, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)

		eP = generateProvisionableObjects(100, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Validate, true)

		eP = generateProvisionableObjects(100, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Cleanup, true)

		// Try empty array
		executeAndValidateProvisionables(t, ctx, []types.Provisionable{}, types.Cleanup, true)

		// Try length 1
		eP = generateProvisionableObjects(1, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)

		eP = generateProvisionableObjects(4, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)

		eP = generateProvisionableObjects(5, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)

		eP = generateProvisionableObjects(6, 0)
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)
	})

	t.Run("TestPerformOperationsSerial", func(t *testing.T) {
		eP := generateProvisionableObjects(100, 0)
		for i := range eP {
			tp := (eP[i]).(*TrackingProvisionable)
			tp.P = false
		}
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)

		eP = generateProvisionableObjects(100, 0)
		for i := range eP {
			if i%3 == 0 {
				tp := (eP[i]).(*TrackingProvisionable)
				tp.P = false
			}
		}
		executeAndValidateProvisionables(t, ctx, eP, types.Provision, true)
	})

	t.Run("TestPerformOperationsBatches", func(t *testing.T) {

		// Empty ParallelProvisioner
		executeAndValidateProvisionables(t, ctx, []types.Provisionable{}, types.Provision, true)
		assert.Equal(t, 0, len(results))

		// Single batch ParallelProvisioner with 50 items with some randomly marked as serial
		batch := generateProvisionableObjects(50, 0)
		for j := range batch {
			if rand.Intn(100) > 50 {
				tp := (batch[j]).(*TrackingProvisionable)
				tp.P = false
			}
		}
		executeAndValidateProvisionables(t, ctx, batch, types.Provision, true)

		// Create 10 batches of 50 Provisionables each and then mark some items randomly as serial
		results = []string{}
		p := []types.Provisionable{}
		items := []types.Provisionable{}

		for i := 1; i < 10; i++ {
			batch := generateProvisionableObjects(50, i)
			for j := range batch {
				if rand.Intn(100) > 50 {
					tp := (batch[j]).(*TrackingProvisionable)
					tp.P = false
				}
			}
			p = append(p, types.NewParallelProvisioner(batch, fmt.Sprintf("Batch(%d)", i)))
			items = append(items, batch...)
		}

		results = []string{}
		pp := types.NewParallelProvisioner(p, "batch test")
		assert.NoErr(t, types.Provision.Execute(ctx, pp))
		validateProvisionableObjects(ctx, t, types.Provision, items)

		// Create 10 batches containing a mix of other batches and provisionables
		p = []types.Provisionable{}
		items = []types.Provisionable{}

		batchNumber := 0

		for i := 1; i < 10; i++ {
			batch := generateProvisionableBatches(&items, 25, &batchNumber, 0, 10)
			p = append(p, types.NewParallelProvisioner(batch, fmt.Sprintf("Batch(%d)", i)))
		}

		results = []string{}
		pp = types.NewParallelProvisioner(p, "batch test")
		assert.NoErr(t, types.Provision.Execute(ctx, pp))
		validateProvisionableObjects(ctx, t, types.Provision, items)
	})

	t.Run("TestPerformExecuteSerial", func(t *testing.T) {
		// Single batch ParallelProvisioner with 50 items with some randomly marked as serial
		batch := generateProvisionableObjects(50, 0)
		// The items should execute serially regardless of their individual state
		executeAndValidateProvisionables(t, ctx, batch, types.Provision, false)

		// Create 10 batches of 10 Provisionables each and then mark some items randomly as serial
		p := []types.Provisionable{}
		items := []types.Provisionable{}
		batches := [][]types.Provisionable{}

		for i := 1; i < 10; i++ {
			batch := generateProvisionableObjects(10, i)
			for j := range batch {
				if rand.Intn(100) > 50 {
					tp := (batch[j]).(*TrackingProvisionable)
					tp.P = false
				}
			}
			p = append(p, types.NewParallelProvisioner(batch, fmt.Sprintf("Batch(%d)", i)))
			items = append(items, batch...)
			batches = append(batches, batch)
		}
		// The 10 batches should execute in order but items inside them in parallel
		results = []string{}
		pp := types.NewRestrictedParallelProvisioner(p, "batch test")
		assert.NoErr(t, types.Provision.Execute(ctx, pp))
		validateProvisionableObjects(ctx, t, types.Provision, items)
		validateProvisionableObjectsOpOrder(ctx, t, batches, 50, []types.ProvisionOperation{types.Provision})

	})
	t.Run("TestPerformExecuteInBatches", func(t *testing.T) {

		// Single batch with 55 items with some randomly marked as serial
		results = []string{}
		batch := generateProvisionableObjects(55, 0)
		for j := range batch {
			if rand.Intn(100) > 50 {
				tp := (batch[j]).(*TrackingProvisionable)
				tp.P = false
			}
		}
		// Request the batch to be broken up into smaller batches of 10
		ops := []types.ProvisionOperation{types.Provision, types.Validate, types.Close}
		bp, err := types.NewBatchProvisioner("TestPerformExecuteInBatches", batch, 10)
		assert.NoErr(t, err)
		assert.NoErr(t, bp.Execute(ctx, ops))
		validateProvisionableObjects(ctx, t, types.Provision, batch)
		validateProvisionableObjects(ctx, t, types.Validate, batch)
		validateProvisionableObjects(ctx, t, types.Close, batch)
		validateProvisionableObjectsOpOrder(ctx, t, createBatches(batch, 10), 10, ops)

		// Create 55 batches of 10 Provisionables each and then mark some items randomly as serial
		results = []string{}
		p := []types.Provisionable{}
		items := []types.Provisionable{}

		for i := 1; i < 55; i++ {
			batch := generateProvisionableObjects(10, i)
			for j := range batch {
				if rand.Intn(100) > 50 {
					tp := (batch[j]).(*TrackingProvisionable)
					tp.P = false
				}
			}
			p = append(p, types.NewParallelProvisioner(batch, fmt.Sprintf("TestPerformExecuteInBatches Batch(%d)", i)))
			items = append(items, batch...)
		}

		bp, err = types.NewBatchProvisioner("TestPerformExecuteInBatches", p, 10)
		assert.NoErr(t, err)
		assert.NoErr(t, bp.Execute(ctx, ops))
		validateProvisionableObjects(ctx, t, types.Provision, items)
		validateProvisionableObjects(ctx, t, types.Validate, items)
		validateProvisionableObjects(ctx, t, types.Close, items)

	})
}

func generateProvisionableBatches(items *[]types.Provisionable, size int, batchNumber *int, depth int, maxdepth int) []types.Provisionable {
	batch := generateProvisionableObjects(size, *batchNumber)
	*batchNumber++
	for j := range batch {
		if rand.Intn(100) > 50 {
			tp := (batch[j]).(*TrackingProvisionable)
			tp.P = false
		}

		if rand.Intn(100) > 95 && depth < maxdepth {

			p := types.NewParallelProvisioner(generateProvisionableBatches(items, size, batchNumber, depth+1, maxdepth), fmt.Sprintf("Batch(%d)", j))
			batch[j] = p
		}

	}
	*items = append(*items, batch...)

	return batch
}
