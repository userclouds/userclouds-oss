package types

import (
	"context"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// ProvisionOperation defines operations that can performed by a Provisionable
type ProvisionOperation string

// Provision operations
const (
	Provision ProvisionOperation = "provision"
	Validate  ProvisionOperation = "validate"
	Cleanup   ProvisionOperation = "cleanup"
	Close     ProvisionOperation = "close"
)

// Execute executes the ProvisionOperation on the Provisionable
func (po ProvisionOperation) Execute(ctx context.Context, p Provisionable) error {
	switch po {
	case Provision:
		return ucerr.Wrap(p.Provision(ctx))
	case Validate:
		return ucerr.Wrap(p.Validate(ctx))
	case Cleanup:
		return ucerr.Wrap(p.Cleanup(ctx))
	case Close:
		return ucerr.Wrap(p.Close(ctx))
	default:
		return ucerr.Errorf("ProvisionOperation %v not recognized", po)
	}
}

// Provisionable defines an interface for Userclouds resources that can be provisioned independently
type Provisionable interface {
	Name() string
	Provision(context.Context) error
	Validate(context.Context) error
	Cleanup(context.Context) error

	// IsExecutableInParallel returns true if this Provisionable can be executed in parallel with
	// other Provisionables at the same level for a given operation; if this function returns false
	// the Provisionable should executed serially
	IsExecutableInParallel(ctx context.Context, op ProvisionOperation) bool
	// Close cleans up resources that may be used by a Provisionable object, and should be always
	// called once the Provisionable object is no longer needed
	Close(ctx context.Context) error
}

// ControlSource is used to pass data and operation necessity from one provisionable to another
type ControlSource interface {
	GetData(context.Context) (any, error)
}

// The following structs implement portions of the Provisionable interface.
// Implementations of Provisionable can embed these as desired to inherit
// these implementations

// Named provides an implementation of the Name method of Provisionable
type Named struct {
	name string
}

// NewNamed returns a Named instance
func NewNamed(name string) Named {
	return Named{name: name}
}

// Name is part of the Provisionable interface
func (n Named) Name() string {
	return n.name
}

// Parallelizable provides an implementation of the IsExecutableInParallel method of Provisionable
type Parallelizable struct {
	alwaysParallel     bool
	parallelOperations set.Set[string]
}

// AllParallelizable returns a Parallelizable that allows all operations to be parallelized
func AllParallelizable() Parallelizable {
	return Parallelizable{alwaysParallel: true}
}

// NewParallelizable returns a Parallelizable that allows specific operations to be parallelized
func NewParallelizable(pos ...ProvisionOperation) Parallelizable {
	p := Parallelizable{
		parallelOperations: set.NewStringSet(),
	}
	for _, po := range pos {
		p.parallelOperations.Insert(string(po))
	}
	return p
}

// IsExecutableInParallel is part of the Provisionable interface
func (p Parallelizable) IsExecutableInParallel(ctx context.Context, po ProvisionOperation) bool {
	return p.alwaysParallel || p.parallelOperations.Contains(string(po))
}

// NoopClose provides a noop implementation of the Close method of Provisionable
type NoopClose struct{}

// Close is part of the Provisionable interface
func (NoopClose) Close(context.Context) error {
	return nil
}
