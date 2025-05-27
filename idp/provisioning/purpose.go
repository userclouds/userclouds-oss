package provisioning

import (
	"context"
	"database/sql"
	"errors"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

// ProvisionDefaultPurposes returns a ProvisionableMaker that can provision purposes
func ProvisionDefaultPurposes(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		return ProvisionPurposes(ctx, name, pi, defaults.GetDefaultPurposes()...)()
	}
}

// ProvisionPurposes returns a ProvisionableMaker that can provision the specified purposes
func ProvisionPurposes(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	purposes ...storage.Purpose,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(purposes) == 0 {
			return nil, ucerr.New("no purposes specified")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision purposes with nil tenantDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		var provs []types.Provisionable
		for _, p := range purposes {
			if isSoftDeleted, err := s.IsPurposeSoftDeleted(ctx, p.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			provs = append(provs, newPurposeProvisioner(ctx, name, pi, p))
		}

		return provs, nil
	}
}

func newPurposeProvisioner(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	p storage.Purpose,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
	return &purposeProvisioner{
		Named:          types.NewNamed(name + ":Purpose"),
		Parallelizable: types.AllParallelizable(),
		s:              s,
		purpose:        p,
	}
}

// purposeProvisioner is a Provisionable object used to set up a purpose
type purposeProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s       *storage.Storage
	purpose storage.Purpose
}

// Provision implements Provisionable and creates a purpose
func (pp *purposeProvisioner) Provision(ctx context.Context) error {
	// TODO: we should be able to eliminate this - existing rules enforce this rule
	// if this purpose was created with a different name, log a warning for now
	if existing, err := pp.s.GetPurposeByName(ctx, pp.purpose.Name); err == nil {
		if existing.ID != pp.purpose.ID {
			uclog.Warningf(ctx, "conflicting IDs for purpose")
			return nil
		}
	}

	// TODO: once we enforce that UserClouds can only provision default purposes,
	//       this should not be necessary
	// if the purpose already exists in the DB, we don't need to do anything
	// since purposes are immutable for the purposes of provisioning
	if existing, err := pp.s.GetPurpose(ctx, pp.purpose.ID); err == nil {
		// TODO (sgarrity 11/23): this is a special case to undo a bug I introduced a few weeks ago
		// when I created new default purposes as "system" when they should not have been.
		if existing.IsSystem == pp.purpose.IsSystem {
			return nil
		}
	}

	if err := pp.s.SavePurpose(ctx, &pp.purpose); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// Validate implements Provisionable and validates that the purpose was created
func (pp *purposeProvisioner) Validate(ctx context.Context) error {
	if _, err := pp.s.GetPurpose(ctx, pp.purpose.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Cleanup implements Provisionable and removes the purpose
func (pp *purposeProvisioner) Cleanup(ctx context.Context) error {
	// If the purpose exists, delete it, but don't error if it doesn't
	if err := pp.s.DeletePurpose(ctx, pp.purpose.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	return nil
}
