package provisioning

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/events"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/provisioning/types"
	provisioningLogServer "userclouds.com/logserver/provisioning"
)

// ProvisionDefaultMutators returns a ProvisionableMaker that can provision default mutators
func ProvisionDefaultMutators(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		return ProvisionMutators(ctx, name, pi, defaults.GetDefaultMutators()...)()
	}
}

// ProvisionMutators returns a ProvisionableMaker that can provision the specified mutators
func ProvisionMutators(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	mutators ...*storage.Mutator,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(mutators) == 0 {
			return nil, ucerr.New("no mutators specified")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision mutators with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot provision mutators with nil logDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
		var provs []types.Provisionable

		for _, m := range mutators {
			if m == nil {
				return nil, ucerr.New("cannot provision nil mutator")
			}

			if isSoftDeleted, err := s.IsMutatorSoftDeleted(ctx, m.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				return nil, nil
			}

			provs = append(
				provs,
				newMutatorProvisioner(ctx, name, pi, m),
			)
		}

		return provs, nil
	}
}

// newMutatorProvisioner returns a Provisionable for provisioning a mutator
func newMutatorProvisioner(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	m *storage.Mutator,
) types.Provisionable {
	var provs []types.Provisionable

	name = name + ":Mutator"

	// Serially provision the mutator
	p := newMutatorObjectProvisioner(ctx, name, pi, m)
	wp := types.NewWrappedProvisionable(p, name)
	provs = append(provs, wp)

	// Provision the events
	p = provisioningLogServer.NewEventProvisioner(
		name,
		pi.LogDB,
		nil,
		provisioningLogServer.ParallelOperations(types.Provision, types.Validate),
		provisioningLogServer.ControlSource(p.(*mutatorProvisioner)),
	)
	provs = append(provs, p)

	return types.NewParallelProvisioner(provs, name)
}

// newMutatorObjectProvisioner returns a Provisionable for provisioning a mutator object without events, etc
func newMutatorObjectProvisioner(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	m *storage.Mutator,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
	mp := mutatorProvisioner{
		Named:          types.NewNamed(name + ":MutatorObject"),
		Parallelizable: types.AllParallelizable(),
		s:              s,
		m:              m,
		mm:             storage.NewMethodManager(ctx, s),
	}
	return &mp
}

// mutatorProvisioner is a Provisionable object used to set up an mutator object
type mutatorProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s  *storage.Storage
	mm *storage.MethodManager
	m  *storage.Mutator
}

// GetData implements ControlSource
func (mp *mutatorProvisioner) GetData(context.Context) (any, error) {
	return events.GetEventsForMutator(mp.m.ID, mp.m.Version), nil
}

func (mp *mutatorProvisioner) refreshMutator(ctx context.Context) error {
	if mp.m.ID != constants.UpdateUserMutatorID {
		return nil
	}

	cm, err := storage.NewUserstoreColumnManager(ctx, mp.s)
	if err != nil {
		return ucerr.Wrap(err)
	}

	mp.m.ColumnIDs = []uuid.UUID{}
	mp.m.NormalizerIDs = []uuid.UUID{}

	for _, c := range cm.GetColumns() {
		if !c.Attributes.Immutable {
			mp.m.ColumnIDs = append(mp.m.ColumnIDs, c.ID)
			mp.m.NormalizerIDs = append(mp.m.NormalizerIDs, policy.TransformerPassthrough.ID)
		}
	}

	return nil
}

// Provision implements Provisionable and provisions the mutator
func (mp *mutatorProvisioner) Provision(ctx context.Context) error {
	if err := mp.refreshMutator(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := mp.mm.SaveMutator(ctx, mp.m); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate implements Provisionable and validates that the mutator exists
func (mp *mutatorProvisioner) Validate(ctx context.Context) error {
	if _, err := mp.s.GetLatestMutator(ctx, mp.m.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Cleanup implements Provisionable and removes the mutator
func (mp *mutatorProvisioner) Cleanup(ctx context.Context) error {
	if err := mp.mm.DeleteMutator(ctx, mp.m.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
