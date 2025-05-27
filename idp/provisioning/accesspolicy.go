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

// ProvisionDefaultAccessPolicies returns a ProvisionableMaker that can provision default access policies
func ProvisionDefaultAccessPolicies(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision default access policies with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot provision default access policies with nil logDB")
		}

		s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
		var provs []types.Provisionable
		for _, dap := range defaults.GetDefaultAccessPolicies() {
			if isSoftDeleted, err := s.IsAccessPolicySoftDeleted(ctx, dap.ID); err != nil {
				return nil, ucerr.Wrap(err)
			} else if isSoftDeleted {
				continue
			}

			provs = append(
				provs,
				newProvisionerAccessPolicy(
					ctx,
					name,
					pi,
					dap,
					types.Provision, types.Validate,
				),
			)
		}

		return provs, nil
	}
}

// CleanUpAccessPolicies returns a ProvisionableMaker that can clean up access policies
func CleanUpAccessPolicies(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if !types.DeepProvisioning {
			return nil, nil
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot clean up access policies with nil tenantDB")
		}

		if pi.LogDB == nil {
			return nil, ucerr.New("cannot clean up access policies with nil logDB")
		}

		p, err := migrateAccessPoliciesFromShadowObjects(ctx, name, pi)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		return []types.Provisionable{p}, nil
	}
}

// migrateAccessPoliciesFromShadowObjects - should be removed once we migrate everything (or made more efficient to just look for missing objects)
func migrateAccessPoliciesFromShadowObjects(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) (types.Provisionable, error) {
	provs := make([]types.Provisionable, 0)
	name = fmt.Sprintf("%s:MigrateAccessPoliciesFromShadowObjects", name)
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)

	// TODO: we will be limited to 1000 results for now - seems unlikely we'll exceed that anytime soon.
	pager, err := storage.NewAccessPolicyPaginatorFromOptions(pagination.Limit(1000))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	aps, respFields, err := s.ListAccessPoliciesPaginated(ctx, *pager)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if pager.AdvanceCursor(*respFields) {
		return nil, ucerr.Errorf("exceeded %d access policies, some will not be returned", pager.GetLimit())
	}

	for _, ap := range aps {
		provs = append(provs, provisioningAuthZ.NewEntityAuthZ(
			name,
			pi,
			nil,
			nil,
			[]authz.Object{
				{BaseModel: ucdb.NewBaseWithID(ap.ID), TypeID: idpAuthz.PolicyAccessTypeID /* we could set alias = p.Name */},
			},
			nil,
			types.Provision, types.Validate,
		))
		provs = append(provs, provisioningLogServer.NewEventProvisioner(name, pi.LogDB, events.GetEventsForAccessPolicy(ap.ID, ap.Version), provisioningLogServer.ParallelOperations(types.Provision, types.Validate)))
	}
	return types.NewParallelProvisioner(provs, name), nil
}

// provisionerAccessPolicy is a Provisionable object used to set up a single access policy
type provisionerAccessPolicy struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	s     *storage.Storage
	logDB *ucdb.DB
	ap    storage.AccessPolicy
}

// newProvisionerAccessPolicy return an initialized Provisionable object for initializing access policy
func newProvisionerAccessPolicy(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	ap storage.AccessPolicy,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	var provs []types.Provisionable

	// Serially provision the access policy
	p := newProvisionerAccessPolicyOnly(ctx, name, pi, ap, pos...)
	wp := types.NewWrappedProvisionable(p, name)
	provs = append(provs, wp)

	// Provision the events and AuthZ edges and objects in parallel
	p = provisioningLogServer.NewEventProvisioner(
		name,
		pi.LogDB,
		nil,
		provisioningLogServer.ParallelOperations(pos...),
		provisioningLogServer.ControlSource(p.(*provisionerAccessPolicy)),
	)
	provs = append(provs, p)

	p = provisioningAuthZ.NewEntityAuthZ(
		name,
		pi,
		nil,
		nil,
		[]authz.Object{
			{BaseModel: ucdb.NewBaseWithID(ap.ID), TypeID: idpAuthz.PolicyAccessTypeID /* we could set alias = ap.Name */},
		},
		[]authz.Edge{
			{EdgeTypeID: idpAuthz.PolicyAccessExistsEdgeTypeID, SourceObjectID: idpAuthz.PoliciesObjectID, TargetObjectID: ap.ID},
		},
		pos...,
	)

	provs = append(provs, p)

	return types.NewParallelProvisioner(provs, name)
}

// newProvisionerAccessPolicyOnly return an initialized Provisionable object for initializing access policy
func newProvisionerAccessPolicyOnly(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
	ap storage.AccessPolicy,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	s := storage.New(ctx, pi.TenantDB, pi.TenantID, pi.CacheCfg)
	pap := provisionerAccessPolicy{
		Named:          types.NewNamed(name + ":AccessPolicyObject"),
		Parallelizable: types.NewParallelizable(pos...),
		s:              s,
		logDB:          pi.LogDB,
		ap:             ap,
	}
	return &pap
}

// GetData implements ControlSource
func (pap *provisionerAccessPolicy) GetData(context.Context) (any, error) {
	return events.GetEventsForAccessPolicy(pap.ap.ID, pap.ap.Version), nil
}

// Provision creates all objects for given access policy
func (pap *provisionerAccessPolicy) Provision(ctx context.Context) error {
	// Check if the access policy already exists by ID
	lap, err := pap.s.GetLatestAccessPolicy(ctx, pap.ap.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	// Check if the access policy already exists by name
	if lap == nil {
		lap, err = pap.s.GetAccessPolicyByName(ctx, pap.ap.Name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}

	// If the access policy already exists, we don't need to do anything if it is the same and bump the version number otherwise
	if lap != nil {
		if lap.EqualsIgnoringNilID(&pap.ap) {
			pap.ap.Version = lap.Version
			uclog.Debugf(ctx, "Access policy %v already exists, no update necessary", pap.ap.ID)
			return nil
		}
		// TODO: this state doesn't converge, we need to figure out namespacing or another solution
		if lap.ID != pap.ap.ID {
			uclog.Warningf(ctx, "Access policy %v already exists with different ID %v", pap.ap.Name, lap.ID)
			return nil
		}

		pap.ap.Version = lap.Version + 1
	}

	uclog.Debugf(ctx, "Writing out access policy %v version %d", pap.ap.ID, pap.ap.Version)
	if err := pap.s.SaveAccessPolicy(ctx, &pap.ap); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate checks if the access policy was properly provisioned
func (pap *provisionerAccessPolicy) Validate(ctx context.Context) error {
	lap, err := pap.s.GetLatestAccessPolicy(ctx, pap.ap.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if lap == nil {
		return ucerr.Errorf("expected access policy ID %v doesn't exist", pap.ap.ID)
	}

	if !lap.EqualsIgnoringNilID(&pap.ap) {
		return ucerr.Errorf("Found policy %v by ID, %+v doesn't match %+v ", pap.ap.ID, lap, pap.ap)
	}

	return nil
}

// Cleanup cleans up objects associated with this access policy
func (pap *provisionerAccessPolicy) Cleanup(ctx context.Context) error {
	lap, err := pap.s.GetLatestAccessPolicy(ctx, pap.ap.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	if !errors.Is(err, sql.ErrNoRows) {
		if err := pap.s.DeleteAccessPolicyByVersion(ctx, lap.ID, lap.Version); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
