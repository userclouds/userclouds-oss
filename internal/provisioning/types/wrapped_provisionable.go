package types

import (
	"context"

	"userclouds.com/infra/ucerr"
)

// WrappedProvisionable wraps a Provisionable with specific restrictions on operation parallelizability
type WrappedProvisionable struct {
	Named
	Parallelizable
	p Provisionable
}

// NewWrappedProvisionable returns an initialized WrappedProvisionable
func NewWrappedProvisionable(p Provisionable, name string, pos ...ProvisionOperation) Provisionable {
	wp := WrappedProvisionable{
		Named:          NewNamed(name + ":Wrap"),
		Parallelizable: NewParallelizable(pos...),
		p:              p,
	}
	return &wp
}

// Provision is part of the Provisionable interface
func (wp *WrappedProvisionable) Provision(ctx context.Context) error {
	return ucerr.Wrap(wp.p.Provision(ctx))
}

// Validate is part of the Provisionable interface
func (wp *WrappedProvisionable) Validate(ctx context.Context) error {
	return ucerr.Wrap(wp.p.Validate(ctx))
}

// Cleanup is part of the Provisionable interface
func (wp *WrappedProvisionable) Cleanup(ctx context.Context) error {
	return ucerr.Wrap(wp.p.Cleanup(ctx))
}

// Close is part of the Provisionable interface
func (wp *WrappedProvisionable) Close(ctx context.Context) error {
	return ucerr.Wrap(wp.p.Close(ctx))
}
