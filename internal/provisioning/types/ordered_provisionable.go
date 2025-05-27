package types

import (
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// ProvisionableMaker will return a collection of Provisionables
type ProvisionableMaker func() ([]Provisionable, error)

// OrderedProvisionable wraps a Provisionable. It must have a unique name
// and has a set of dependencies it must come after, with each dependency
// referring to a unique OrderedProvisionable.
type OrderedProvisionable struct {
	name               string
	dependencies       set.Set[string]
	makeProvisionables ProvisionableMaker
	provisionables     []Provisionable
}

// NewOrderedProvisionable is used to create an OrderedProvisionable
func NewOrderedProvisionable(pm ProvisionableMaker) OrderedProvisionable {
	return OrderedProvisionable{
		makeProvisionables: pm,
		dependencies:       set.NewStringSet(),
	}
}

// Named sets the name of the OrderedProvisionable
func (op OrderedProvisionable) Named(name string) OrderedProvisionable {
	op.name = name
	return op
}

// After specifies a list of dependencies that this OrderedProvisionable must come after
func (op OrderedProvisionable) After(dependencies ...string) OrderedProvisionable {
	op.dependencies.Insert(dependencies...)
	return op
}

// Validate ensures that the OrderedProvisionable is valid
func (op OrderedProvisionable) Validate() error {
	if op.name == "" {
		return ucerr.Errorf("name must be non-empty: %v", op)
	}

	return nil
}

// OrderedProvisionables is a collection of OrderedProvisionable instances
type OrderedProvisionables []OrderedProvisionable

func (ops OrderedProvisionables) finalize() (OrderedProvisionables, error) {
	// iterate over OrderedProvisionables, generating Provisionables for each
	// using the associated ProvisionableMaker

	var orderedProvisionables OrderedProvisionables
	for _, op := range ops {
		provisionables, err := op.makeProvisionables()
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if len(provisionables) > 0 {
			op.provisionables = provisionables
		}

		orderedProvisionables = append(orderedProvisionables, op)
	}

	return orderedProvisionables, nil
}

// Validate ensures that each OrderedProvisionable is valid, has a unique name,
// and that each dependency refers to an OrderedProvisionable in the collection.
func (ops OrderedProvisionables) Validate() error {
	if len(ops) == 0 {
		return ucerr.New("no OrderedProvisionables specified")
	}

	uniqueNames := set.NewStringSet()
	for _, op := range ops {
		if err := op.Validate(); err != nil {
			return ucerr.Wrap(err)
		}

		if uniqueNames.Contains(op.name) {
			return ucerr.Errorf("names must be unique: %v", op)
		}
		uniqueNames.Insert(op.name)
	}

	for _, op := range ops {
		if !uniqueNames.IsSupersetOf(op.dependencies) {
			return ucerr.Errorf("OrderedProvisionable contains unrecognized dependencies: %v", op)
		}
	}

	return nil
}
