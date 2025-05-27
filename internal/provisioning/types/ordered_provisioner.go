package types

import (
	"fmt"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// OrderedProvisioner can create a Provisionable that will execute in a minimal number of serial
// phases, with the Provisionables of each phase being executed in parallel, based on the specified
// dependencies of each of the OrderedProvisionables associated with the OrderedProvisioner.
type OrderedProvisioner struct {
	name           string
	provisionables OrderedProvisionables
}

// NewOrderedProvisioner creates an OrderedProvisioner with the specified name
func NewOrderedProvisioner(name string) OrderedProvisioner {
	return OrderedProvisioner{
		name:           name + ":OrderedProvisioner",
		provisionables: OrderedProvisionables{},
	}
}

// AddProvisionables adds a collection of OrderedProvisionables that can be scheduled
func (op *OrderedProvisioner) AddProvisionables(provisionables ...OrderedProvisionable) {
	op.provisionables = append(op.provisionables, provisionables...)
}

// CreateProvisionable creates a Provisionable that executes a collection of
// OrderedProvisionables in a series of phases, with the Provisionables of
// each phase being executed in parallel, based on the specified dependencies
// of each OrderedProvisionable
func (op OrderedProvisioner) CreateProvisionable() (Provisionable, error) {
	schedulableProvisionables, err := op.provisionables.finalize()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := schedulableProvisionables.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	scheduledProvisionables := set.NewStringSet()
	scheduledPhases := []Provisionable{}

	for phase := 1; len(schedulableProvisionables) > 0; phase++ {
		var phaseProvisionables OrderedProvisionables
		var unscheduledProvisionables OrderedProvisionables

		// iterate over the schedulable provisionables until no more can be handled by this phase
		for {
			anyScheduled := false

			for _, sp := range schedulableProvisionables {
				if scheduledProvisionables.IsSupersetOf(sp.dependencies) {
					anyScheduled = true
					if len(sp.provisionables) > 0 {
						phaseProvisionables = append(phaseProvisionables, sp)
					} else {
						// a shedulable provisionable that does not produce any
						// provisionables can immediately be merged into this
						// phase

						scheduledProvisionables.Insert(sp.name)
					}
				} else {
					unscheduledProvisionables = append(unscheduledProvisionables, sp)
				}
			}

			if !anyScheduled {
				break
			}

			schedulableProvisionables = unscheduledProvisionables
			unscheduledProvisionables = nil
		}

		if len(phaseProvisionables) > 0 {
			var names []string
			var provs []Provisionable
			for _, pp := range phaseProvisionables {
				names = append(names, pp.name)
				scheduledProvisionables.Insert(pp.name)
				provs = append(provs, pp.provisionables...)
			}
			scheduledPhases = append(scheduledPhases, NewLogMessageProvisioner(fmt.Sprintf("%s: Starting Phase%d %v (%d provisionables)", op.name, phase, names, len(provs))))
			scheduledPhases = append(scheduledPhases, NewParallelProvisioner(provs, fmt.Sprintf("%s:Phase%d", op.name, phase)))
			scheduledPhases = append(scheduledPhases, NewLogMessageProvisioner(fmt.Sprintf("%s: Completed Phase%d", op.name, phase)))
		} else if len(unscheduledProvisionables) > 0 {
			return nil, ucerr.Errorf("could not schedule remaining provisionables: '%v'", unscheduledProvisionables)
		}
	}

	return NewRestrictedParallelProvisioner(scheduledPhases, op.name, Validate), nil
}
