package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

func provisionObjectType(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeName string) error {
	objectType := authz.ObjectType{
		BaseModel: ucdb.NewBaseWithID(id),
		TypeName:  typeName,
	}
	uclog.Debugf(ctx, "Provisioning ObjectType ID %v Type Name %s", id, typeName)
	if err := storage.SaveObjectType(ctx, &objectType); err != nil {
		uclog.Errorf(ctx, "Failed to provision Object Type ID %v Type Name %s with %v", id, typeName, err)
		return ucerr.Wrap(err)
	}
	return nil
}

func deleteObjectType(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeName string) error {
	uclog.Debugf(ctx, "Deleting ObjectType ID %v Type Name %s", id, typeName)
	if err := storage.DeleteObjectType(ctx, id); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func provisionEdgeType(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeName string, sourceObjectTypeID, targetObjectTypeID uuid.UUID, attributes authz.Attributes) error {
	edgeType := authz.EdgeType{
		BaseModel:          ucdb.NewBaseWithID(id),
		TypeName:           typeName,
		SourceObjectTypeID: sourceObjectTypeID,
		TargetObjectTypeID: targetObjectTypeID,
		Attributes:         attributes,
	}
	uclog.Debugf(ctx, "Provisioning EdgeType ID %v Type Name %s SourceID %v TargetID %v", id, typeName, sourceObjectTypeID, targetObjectTypeID)
	if err := storage.SaveEdgeType(ctx, &edgeType); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func validateEdgeType(ctx context.Context, storage *internal.Storage, typeName string, sourceObjectTypeID, targetObjectTypeID uuid.UUID) error {
	uclog.Debugf(ctx, "Validating EdgeType Type Name %s SourceID %v TargetID %v", typeName, sourceObjectTypeID, targetObjectTypeID)
	edgeType, err := storage.GetEdgeTypeForName(ctx, typeName)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if edgeType.SourceObjectTypeID != sourceObjectTypeID || edgeType.TargetObjectTypeID != targetObjectTypeID {
		return ucerr.Errorf("'%s' edge has wrong source/target object types", typeName)
	}
	return nil
}

func deleteEdgeType(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeName string, sourceObjectTypeID, targetObjectTypeID uuid.UUID) error {
	uclog.Debugf(ctx, "Deleting EdgeType ID %v Type Name %s SourceID %v TargetID %v", id, typeName, sourceObjectTypeID, targetObjectTypeID)
	if err := storage.DeleteEdgeType(ctx, id); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func provisionObject(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeID uuid.UUID, alias *string, orgID uuid.UUID) error {
	obj := authz.Object{
		BaseModel:      ucdb.NewBaseWithID(id),
		Alias:          alias,
		TypeID:         typeID,
		OrganizationID: orgID,
	}
	a := "nil"
	if alias != nil {
		a = *alias
	}

	uclog.Debugf(ctx, "Provisioning Object ID %v Type ID %v Alias %s", id, typeID, a)

	if err := storage.SaveObject(ctx, &obj); err != nil {
		uclog.Errorf(ctx, "Failed to provision Object ID %v Type ID %v Alias %s with %v", id, typeID, a, err)
		return ucerr.Wrap(err)
	}
	return nil
}

func validateObject(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeID uuid.UUID, alias *string, organizationID uuid.UUID) error {
	aliasExp := ""
	if alias != nil {
		aliasExp = *alias
	}

	uclog.Debugf(ctx, "Validating Object ID %v Type ID %v Alias %s", id, typeID, aliasExp)
	obj, err := storage.GetObject(ctx, id)
	if err != nil {
		return ucerr.Wrap(err)
	}

	aliasObj := ""
	if obj.Alias != nil {
		aliasObj = *obj.Alias
	}
	if obj.TypeID != typeID || aliasExp != aliasObj || obj.OrganizationID != organizationID {
		expectedObj := authz.Object{
			BaseModel:      ucdb.NewBaseWithID(id),
			Alias:          alias,
			TypeID:         typeID,
			OrganizationID: organizationID,
		}

		uclog.Debugf(ctx, "Alias exp %s Alias in %s", aliasExp, aliasObj)
		return ucerr.Errorf("mismatched object in provisioning validation (got %v, expected %v)", obj, expectedObj)
	}
	return nil
}

func deleteObject(ctx context.Context, storage *internal.Storage, id uuid.UUID, typeID uuid.UUID, alias *string) error {
	aliasExp := ""
	if alias != nil {
		aliasExp = *alias
	}
	uclog.Debugf(ctx, "Deleting Object ID %v Type ID %v Alias %s", id, typeID, aliasExp)
	if err := storage.DeleteObject(ctx, id); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func provisionEdge(ctx context.Context, storage *internal.Storage, edgeTypeID, sourceObjectID, targetObjectID uuid.UUID) (uuid.UUID, error) {
	if edge, err := storage.FindEdge(ctx, edgeTypeID, sourceObjectID, targetObjectID); err == nil {
		return edge.ID, nil
	}

	edge := authz.Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     edgeTypeID,
		SourceObjectID: sourceObjectID,
		TargetObjectID: targetObjectID,
	}
	uclog.Debugf(ctx, "Provisioning Edge ID %v Type ID %v Source %v Target %v", edge.ID, edgeTypeID, sourceObjectID, targetObjectID)

	if err := storage.SaveEdge(ctx, &edge); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return edge.ID, nil
}

func validateEdge(ctx context.Context, storage *internal.Storage, edgeTypeID, sourceObjectID, targetObjectID uuid.UUID) error {
	uclog.Debugf(ctx, "Validating Edge Type ID %v Source %v Target %v", edgeTypeID, sourceObjectID, targetObjectID)
	if _, err := storage.FindEdge(ctx, edgeTypeID, sourceObjectID, targetObjectID); err != nil {
		return ucerr.Errorf("During provisioning couldn't find edge of type %v from %v to %v", edgeTypeID, sourceObjectID, targetObjectID)
	}

	return nil
}

func deleteEdge(ctx context.Context, storage *internal.Storage, edgeTypeID, sourceObjectID, targetObjectID uuid.UUID) error {
	uclog.Debugf(ctx, "Deleting Edge Type ID %v Source %v Target %v", edgeTypeID, sourceObjectID, targetObjectID)
	edge, err := storage.FindEdge(ctx, edgeTypeID, sourceObjectID, targetObjectID)

	// The edge maybe already deleted by object cleanup, so this is not an error
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return ucerr.Errorf("During cleanup couldn't find edge of type %v from %v to %v", edgeTypeID, sourceObjectID, targetObjectID)
	}

	if err := storage.DeleteEdge(ctx, edge.ID); err != nil {
		return ucerr.Errorf("During cleanup couldn't delete edge of type %v from %v to %v", edgeTypeID, sourceObjectID, targetObjectID)
	}
	return nil
}

// ReadAuthZObjects is a temporary function that allows IDP to list Authz objects during provisioning without reimplementing pagination
// TODO auth/internal/storage should move to internal/storage/authz and then IDP can instantiate this directly
func ReadAuthZObjects(ctx context.Context, pi types.ProvisionInfo, typeID uuid.UUID) ([]authz.Object, error) {
	storage := internal.NewStorage(ctx, pi.TenantID, pi.TenantDB, pi.CacheCfg)
	authzObjects := []authz.Object{}
	pager, err := authz.NewObjectPaginatorFromOptions(
		pagination.Filter(fmt.Sprintf("('type_id',EQ,'%v')", typeID)),
		pagination.Limit(1500))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	for {

		a, respFields, err := storage.ListObjectsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		authzObjects = append(authzObjects, a...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	uclog.Debugf(ctx, "Read %d authz objects of type %v", len(authzObjects), typeID)
	return authzObjects, nil
}
