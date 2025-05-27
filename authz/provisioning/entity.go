package provisioning

import (
	"context"
	"database/sql"
	"errors"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/provisioning/types"
)

// EntityAuthZ is a Provisionable object used to setup AuthZ state for a set of passed in types/instances.
type EntityAuthZ struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	types.ProvisionInfo
	objTypes  []authz.ObjectType
	edgeTypes []authz.EdgeType
	objects   []authz.Object
	edges     []authz.Edge
}

// NewEntityAuthZ initialized EntityAuthZ with given set of types and instances
func NewEntityAuthZ(
	name string,
	pi types.ProvisionInfo,
	objTypes []authz.ObjectType,
	edgeTypes []authz.EdgeType,
	objects []authz.Object,
	edges []authz.Edge,
	pos ...types.ProvisionOperation,
) types.Provisionable {
	p := EntityAuthZ{
		Named:          types.NewNamed(name + ":EntityAuthZ"),
		Parallelizable: types.NewParallelizable(pos...),
		ProvisionInfo:  pi,
		objTypes:       objTypes,
		edgeTypes:      edgeTypes,
		objects:        objects,
		edges:          edges,
	}
	return &p
}

func (t *EntityAuthZ) newStorage(ctx context.Context) *internal.Storage {
	return internal.NewStorage(ctx, t.TenantID, t.TenantDB, t.CacheCfg)
}

// Provision implements Provisionable and creates or updates AuthZ types and instances for a
// tenant in the tenant's dedicated DB.
func (t *EntityAuthZ) Provision(ctx context.Context) error {

	// Connect to Tenant's AuthZ DB
	storage := t.newStorage(ctx)

	for _, oT := range t.objTypes {
		if err := provisionObjectType(ctx, storage, oT.ID, oT.TypeName); err != nil {
			return ucerr.Wrap(err)
		}
	}

	for _, o := range t.objects {
		if err := provisionObject(ctx, storage, o.ID, o.TypeID, o.Alias, o.OrganizationID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	for _, eT := range t.edgeTypes {
		if err := provisionEdgeType(ctx, storage, eT.ID, eT.TypeName, eT.SourceObjectTypeID, eT.TargetObjectTypeID, eT.Attributes); err != nil {
			return ucerr.Errorf("failed to provisione edge type %v: %w", eT, err)
		}
	}

	for _, e := range t.edges {
		if _, err := provisionEdge(ctx, storage, e.EdgeTypeID, e.SourceObjectID, e.TargetObjectID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil

}

// Validate validates that a tenant has the given AuthZ object types/edge types provisioned.
func (t *EntityAuthZ) Validate(ctx context.Context) error {

	// Connect to Tenant's AuthZ DB
	storage := t.newStorage(ctx)

	// Validate the object types
	for _, oT := range t.objTypes {
		objectType, err := storage.GetObjectTypeForName(ctx, oT.TypeName)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if objectType.ID != oT.ID || objectType.TypeName != oT.TypeName {
			return ucerr.Errorf("Mistmatched object types (expected %v) (got %v)", oT, objectType)
		}
	}

	// Validate object
	for _, o := range t.objects {
		if err := validateObject(ctx, storage, o.ID, o.TypeID, o.Alias, o.OrganizationID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	// Validate the edge types
	for _, eT := range t.edgeTypes {
		if err := validateEdgeType(ctx, storage, eT.TypeName, eT.SourceObjectTypeID, eT.TargetObjectTypeID); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// Validate the edges
	for _, e := range t.edges {
		if err := validateEdge(ctx, storage, e.EdgeTypeID, e.SourceObjectID, e.TargetObjectID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// Cleanup cleans up the AuthZ objects types/instances
func (t *EntityAuthZ) Cleanup(ctx context.Context) error {
	storage := t.newStorage(ctx)

	for _, e := range t.edges {
		if err := deleteEdge(ctx, storage, e.EdgeTypeID, e.SourceObjectID, e.TargetObjectID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}

	for _, o := range t.objects {
		if err := deleteObject(ctx, storage, o.ID, o.TypeID, o.Alias); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}

	for _, eT := range t.edgeTypes {
		if err := deleteEdgeType(ctx, storage, eT.ID, eT.TypeName, eT.SourceObjectTypeID, eT.TargetObjectTypeID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}

	for _, oT := range t.objTypes {
		if err := deleteObjectType(ctx, storage, oT.ID, oT.TypeName); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
