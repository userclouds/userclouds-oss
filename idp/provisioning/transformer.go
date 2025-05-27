package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"userclouds.com/authz"
	provisioningAuthZ "userclouds.com/authz/provisioning"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/events"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/provisioning/defaults"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
	provisioningLogServer "userclouds.com/logserver/provisioning"
)

// ProvisionDefaultTransformers returns a ProvisionableMaker that can provision default transformers
func ProvisionDefaultTransformers(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision default transformers with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot provision default transformers with nil logDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

		var provs []types.Provisionable
		for _, dt := range defaults.GetDefaultTransformers() {
			if isSoftDeleted, err := s.IsTransformerSoftDeleted(ctx, dt.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			provs = append(
				provs,
				newProvisionerTransformer(
					ctx,
					name,
					pi,
					dt,
					types.Provision, types.Validate,
				),
			)
		}

		return provs, nil
	}
}

// CleanUpTransformers returns a ProvisionableMaker that can clean up transformers
func CleanUpTransformers(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if !types.DeepProvisioning {
			return nil, nil
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot clean up transformers with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot clean up transformers with nil logDB")
		}

		p, err := migrateTransformersFromShadowObjects(ctx, name, pi)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		return []types.Provisionable{p}, nil
	}
}

// migrateTransformersFromShadowObjects - should be removed once we migrate everything (or made more efficient to just look for missing objects)
func migrateTransformersFromShadowObjects(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) (types.Provisionable, error) {
	provs := make([]types.Provisionable, 0)
	name = name + ":MigrateTransformersFromShadowObjects"
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

	// TODO: we will be limited to 1000 results for now - seems unlikely we'll exceed that anytime soon.
	pager, err := storage.NewTransformerPaginatorFromOptions(pagination.Limit(1000))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	ts, respFields, err := s.ListTransformersPaginated(ctx, *pager)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if pager.AdvanceCursor(*respFields) {
		return nil, ucerr.Errorf("exceeded %d transformers, some will not be returned", pager.GetLimit())
	}

	for _, t := range ts {
		p := provisioningAuthZ.NewEntityAuthZ(
			name,
			pi,
			nil,
			nil,
			[]authz.Object{
				{BaseModel: ucdb.NewBaseWithID(t.ID), TypeID: idpAuthz.PolicyTransformerTypeID /* we could set alias = p.Name */},
			},
			nil,
			types.Provision,
			types.Validate,
		)
		provs = append(provs, p)

		p = provisioningLogServer.NewEventProvisioner(name, pi.LogDB, events.GetEventsForTransformer(t.ID, t.Version), provisioningLogServer.ParallelOperations(types.Provision, types.Validate))
		provs = append(provs, p)
	}

	return types.NewParallelProvisioner(provs, name), nil
}

// provisionerTransformer is a Provisionable object used to set up a single transformer
type provisionerTransformer struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s           *storage.Storage
	logDB       *ucdb.DB
	transformer storage.Transformer
}

// newProvisionerTransformer return an initialized Provisionable object for initializing a transformer
func newProvisionerTransformer(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	t storage.Transformer,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	var provs []types.Provisionable

	name = fmt.Sprintf("%s:Transformer(%v)", name, t.ID)

	// Serially provision the transformer
	p := newProvisionerTransformerOnly(ctx, name, pi, t, pos...)
	wp := types.NewWrappedProvisionable(p, name)
	provs = append(provs, wp)

	// Provision the events and AuthZ edges and objects in parallel
	p = provisioningLogServer.NewEventProvisioner(
		name,
		pi.LogDB,
		events.GetEventsForTransformer(t.ID, t.Version),
		provisioningLogServer.ParallelOperations(pos...),
	)
	provs = append(provs, p)

	p = provisioningAuthZ.NewEntityAuthZ(
		name,
		pi,
		nil,
		nil,
		[]authz.Object{
			{BaseModel: ucdb.NewBaseWithID(t.ID), TypeID: idpAuthz.PolicyTransformerTypeID /* we could set alias = t.Name */},
		},
		[]authz.Edge{
			{EdgeTypeID: idpAuthz.PolicyTransformerExistsEdgeTypeID, SourceObjectID: idpAuthz.PoliciesObjectID, TargetObjectID: t.ID},
		},
		pos...,
	)
	provs = append(provs, p)

	return types.NewParallelProvisioner(provs, name)
}

// newProvisionerTransformerOnly return an initialized Provisionable object for initializing a transformer
func newProvisionerTransformerOnly(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	transformer storage.Transformer,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
	pt := provisionerTransformer{
		Named:          types.NewNamed(name + ":TransformerObject"),
		Parallelizable: types.NewParallelizable(pos...),
		s:              s,
		logDB:          pi.LogDB,
		transformer:    transformer,
	}
	return &pt
}

// Provision will provision the transform policy
func (pt *provisionerTransformer) Provision(ctx context.Context) error {
	// Check if the policy is already provisioned
	transformer, err := pt.s.GetLatestTransformer(ctx, pt.transformer.ID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if transformer != nil {
		// TODO: ideally we'd have shared logic for updating a transformer between provisioning
		//       and the IDP transformer handler
		if err := transformer.ValidateProvisioningUpdate(pt.transformer); err != nil {
			uclog.Errorf(ctx, "Transformer '%v' has disallowed provisioning changes", transformer.ID)
			return ucerr.Wrap(err)
		}

		if !transformer.Equals(pt.transformer) {
			uclog.Errorf(ctx, "Transformer changed under same ID: %v", transformer.ID)
		}
	}

	uclog.Debugf(ctx, "Writing out transformer %v", pt.transformer.ID)
	if err := pt.s.SaveTransformer(ctx, &pt.transformer); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate with verify that transformer is correctly provisioned
func (pt *provisionerTransformer) Validate(ctx context.Context) error {
	// Check both accessor paths to ensure you get the right policy
	transformer, err := pt.s.GetLatestTransformer(ctx, pt.transformer.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if transformer == nil {
		return ucerr.Errorf("expected transformer ID %v doesn't exist", pt.transformer.ID)
	}

	if !transformer.Equals(pt.transformer) {
		return ucerr.Errorf("Found policy %v by ID, %+v doesn't match %+v", transformer.ID, transformer, pt.transformer)
	}

	return nil
}

// Cleanup cleans up objects associated with this access policy
func (pt *provisionerTransformer) Cleanup(ctx context.Context) error {
	transformer, err := pt.s.GetLatestTransformer(ctx, pt.transformer.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if !errors.Is(err, sql.ErrNoRows) {
		if err := pt.s.DeleteAllTransformerVersions(ctx, transformer.ID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
