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

// ProvisionDefaultAccessors returns a ProvisionableMaker that can provision default accessors
func ProvisionDefaultAccessors(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		return ProvisionAccessors(ctx, name, pi, defaults.GetDefaultAccessors()...)()
	}
}

// ProvisionAccessors returns a ProvisionableMaker that can provision the specified accessors
func ProvisionAccessors(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	accessors ...*storage.Accessor,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(accessors) == 0 {
			return nil, ucerr.New("no accessors specified")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision accessors with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot provision accessors with nil logDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		var provs []types.Provisionable

		for _, a := range accessors {
			if a == nil {
				return nil, ucerr.New("cannot provision nil accessor")
			}

			if isSoftDeleted, err := s.IsAccessorSoftDeleted(ctx, a.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			provs = append(
				provs,
				newAccessorProvisioner(ctx, name, pi, a),
			)
		}

		return provs, nil
	}
}

// newAccessorProvisioner returns a Provisionable for provisioning an accessor
func newAccessorProvisioner(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	a *storage.Accessor,
) types.Provisionable {
	var provs []types.Provisionable

	name = name + ":Accessor"

	// Serially provision the accessor
	p := newAccessorObjectProvisioner(ctx, name, pi, a)
	wp := types.NewWrappedProvisionable(p, name)
	provs = append(provs, wp)

	// Provision events for the accessor
	p = provisioningLogServer.NewEventProvisioner(
		name,
		pi.LogDB,
		nil,
		provisioningLogServer.ParallelOperations(types.Provision, types.Validate),
		provisioningLogServer.ControlSource(p.(*accessorProvisioner)),
	)
	provs = append(provs, p)

	return types.NewParallelProvisioner(provs, name)
}

// newAccessorObjectProvisioner returns a Provisionable for provisioning an accessor object without events, etc
func newAccessorObjectProvisioner(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	a *storage.Accessor,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
	ap := accessorProvisioner{
		Named:          types.NewNamed(name + ":AccessorObject"),
		Parallelizable: types.AllParallelizable(),
		s:              s,
		a:              a,
		mm:             storage.NewMethodManager(ctx, s),
	}
	return &ap
}

// accessorProvisioner is a Provisionable object used to set up an accessor object
type accessorProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s  *storage.Storage
	mm *storage.MethodManager
	a  *storage.Accessor
}

// GetData implements ControlSource
func (ap *accessorProvisioner) GetData(context.Context) (any, error) {
	return events.GetEventsForAccessor(ap.a.ID, ap.a.Version), nil
}

func (ap *accessorProvisioner) refreshAccessor(ctx context.Context) error {
	if ap.a.ID != constants.GetUserAccessorID {
		return nil
	}

	cm, err := storage.NewUserstoreColumnManager(ctx, ap.s)
	if err != nil {
		return ucerr.Wrap(err)
	}

	columns := cm.GetColumns()
	if len(columns) == 0 {
		return ucerr.New("no columns available for default accessor")
	}

	ap.a.ColumnIDs = make([]uuid.UUID, len(columns))
	ap.a.TransformerIDs = make([]uuid.UUID, len(columns))
	ap.a.TokenAccessPolicyIDs = make([]uuid.UUID, len(columns))
	for i, c := range columns {
		ap.a.ColumnIDs[i] = c.ID
		ap.a.TransformerIDs[i] = policy.TransformerPassthrough.ID
		ap.a.TokenAccessPolicyIDs[i] = uuid.Nil
	}

	return nil
}

// Provision implements Provisionable and provisions the accessor
func (ap *accessorProvisioner) Provision(ctx context.Context) error {
	if err := ap.refreshAccessor(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := ap.mm.SaveAccessor(ctx, ap.a); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate implements Provisionable and validates that the accessor exists
func (ap *accessorProvisioner) Validate(ctx context.Context) error {
	if _, err := ap.s.GetLatestAccessor(ctx, ap.a.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Cleanup implements Provisionable and removes the accessor
func (ap *accessorProvisioner) Cleanup(ctx context.Context) error {
	if err := ap.mm.DeleteAccessor(ctx, ap.a.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
