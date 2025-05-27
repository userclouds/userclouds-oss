// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/authz"
	"userclouds.com/infra/uclog"
)

// BuildOpenAPISpec is a generated helper function for building the OpenAPI spec for this service
func BuildOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {

	{
		// Create custom schema mapping for 3rd party type.
		uuidDef := jsonschema.Schema{}
		uuidDef.AddType(jsonschema.String)
		uuidDef.WithFormat("uuid")
		uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

		// Map 3rd party type with your own schema.
		reflector.AddTypeMapping(uuid.UUID{}, uuidDef)
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/checkattribute")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Check Attribute")
		op.SetDescription("This endpoint receives a source object ID, target object ID and attribute. It returns a boolean indicating whether the source object has the attribute permission on the target object.")
		op.SetTags("Permissions")
		op.AddReqStructure(new(CheckAttributeParams))
		op.AddRespStructure(new(authz.CheckAttributeResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/edges")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Edges")
		op.SetDescription("This endpoint returns a paginated list of all edges in a tenant. The list can be filtered to only include edges with a specified organization, source object or target object.")
		op.SetTags("Edges")
		op.AddReqStructure(new(listEdgesParams))
		op.AddRespStructure(new(authz.ListEdgesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authz/edges")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Edge")
		op.SetDescription("This endpoint creates a directed edge of a given type between a source object and target object, both of which are specified by ID.")
		op.SetTags("Edges")
		op.AddReqStructure(new(authz.CreateEdgeRequest))
		op.AddRespStructure(new(authz.Edge), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/edges/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Edge")
		op.SetDescription("This endpoint deletes an edge by ID.")
		op.SetTags("Edges")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/edges/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Edge")
		op.SetDescription("This endpoint gets an edge by ID.")
		op.SetTags("Edges")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(authz.Edge), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/edgetypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Edge Types")
		op.SetDescription("This endpoint returns a paginated list of edge types in a tenant. The list can be filtered to only include edge types with a specified organization, source object type or target object type.")
		op.SetTags("Edge Types")
		op.AddReqStructure(new(listEdgeTypesParams))
		op.AddRespStructure(new(authz.ListEdgeTypesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authz/edgetypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Edge Type")
		op.SetDescription("This endpoint creates a new edge type, complete with name, source object type and target object type. Edges of a given type can only link source objects and target objects of the specified types.")
		op.SetTags("Edge Types")
		op.AddReqStructure(new(authz.CreateEdgeTypeRequest))
		op.AddRespStructure(new(authz.EdgeType), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/edgetypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Edge Type")
		op.SetDescription("This endpoint deletes an edge type by ID. It also deletes all edges which use this edge type.")
		op.SetTags("Edge Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/edgetypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Edge Type")
		op.SetDescription("This endpoint gets an edge type by ID.")
		op.SetTags("Edge Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(authz.EdgeType), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/authz/edgetypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Edge Type")
		op.SetDescription("This endpoint updates an edge type. It is used to adjust the attributes associated with the edge type.")
		op.SetTags("Edge Types")
		type UpdateEdgeTypeRequest struct {
			ID uuid.UUID `path:"id"`
			authz.UpdateEdgeTypeRequest
		}
		op.AddReqStructure(new(UpdateEdgeTypeRequest))
		op.AddRespStructure(new(authz.EdgeType), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/listattributes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Attributes")
		op.SetDescription("This endpoint receives a source object ID and target object ID. It returns a list of attributes that the source object has on the target object.")
		op.SetTags("Permissions")
		op.AddReqStructure(new(listAttributesParams))
		op.AddRespStructure(new([]string), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/listobjectsreachablewithattribute")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Objects Reachable with Attribute")
		op.SetDescription("This endpoint receives a source object ID and attribute. It returns a list of objects reachable from the source object with the attribute.")
		op.SetTags("Permissions")
		op.AddReqStructure(new(listObjectsReachableWithAttributeParams))
		op.AddRespStructure(new(authz.ListObjectsReachableWithAttributeResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/objects")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Objects")
		op.SetDescription("This endpoint returns a paginated list of objects in a tenant. The list can be filtered to only include objects with a specified type, name or organization.")
		op.SetTags("Objects")
		op.AddReqStructure(new(listObjectsParams))
		op.AddRespStructure(new(authz.ListObjectsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authz/objects")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Object")
		op.SetDescription("This endpoint creates an object with a given ID, Type ID, and Alias.")
		op.SetTags("Objects")
		op.AddReqStructure(new(authz.CreateObjectRequest))
		op.AddRespStructure(new(authz.Object), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/objects/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Object")
		op.SetDescription("This endpoint deletes an object by ID. This also deletes all edges that use that object.")
		op.SetTags("Objects")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/objects/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Object")
		op.SetDescription("This endpoint gets an object by ID. If the ID provided is that of a User in the IDP, it returns an object representing the user.")
		op.SetTags("Objects")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(authz.Object), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/authz/objects/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		type UpdateObjectRequest struct {
			ID uuid.UUID `path:"id"`
			authz.UpdateObjectRequest
		}
		op.AddReqStructure(new(UpdateObjectRequest))
		op.AddRespStructure(new(authz.Object), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/objects/{id}/edges")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Edges on Object")
		op.SetDescription("This endpoint deletes all edges associated with an object (specified by ID).")
		op.SetTags("Edges")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/objects/{id}/edges")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Edges on Object")
		op.SetDescription("This endpoint returns a paginated list of edges associated with an object, which is specified by ID. The endpoint lists all incoming and outgoing edges (i.e. all edges where the provided object is a source or target).")
		op.SetTags("Edges")
		type ListEdgesOnObjectParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			ListEdgesOnObjectParams
		}
		op.AddReqStructure(new(ListEdgesOnObjectParamsAndPath))
		op.AddRespStructure(new(authz.ListEdgesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/objecttypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Object Types")
		op.SetDescription("This endpoint returns a paginated list of all object types in a tenant.")
		op.SetTags("Object Types")
		op.AddReqStructure(new(listObjectTypesParams))
		op.AddRespStructure(new(authz.ListObjectTypesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authz/objecttypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Object Type")
		op.SetDescription("This endpoint creates a new object type.")
		op.SetTags("Object Types")
		op.AddReqStructure(new(authz.CreateObjectTypeRequest))
		op.AddRespStructure(new(authz.ObjectType), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/objecttypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Object Type")
		op.SetDescription("This endpoint deletes an object type by ID. It also deletes all objects, edge types and edges which use the object type.")
		op.SetTags("Object Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/objecttypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Object Type")
		op.SetDescription("This endpoint gets an object type by ID.")
		op.SetTags("Object Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(authz.ObjectType), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/organizations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Organizations")
		op.SetDescription("This endpoint returns a paginated list of all organizations in a tenant.")
		op.SetTags("Organizations")
		op.AddReqStructure(new(listOrganizationsParams))
		op.AddRespStructure(new(authz.ListOrganizationsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authz/organizations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Organization")
		op.SetDescription("This endpoint creates an organization.")
		op.SetTags("Organizations")
		op.AddReqStructure(new(authz.CreateOrganizationRequest))
		op.AddRespStructure(new(authz.Organization), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authz/organizations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Organization")
		op.SetDescription("This endpoint deletes an organization by ID.")
		op.SetTags("Organizations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(200))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotImplemented))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authz/organizations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Organization")
		op.SetDescription("This endpoint gets an organization by ID.")
		op.SetTags("Organizations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(authz.Organization), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/authz/organizations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Organization")
		op.SetDescription("This endpoint updates an organization, specified by ID.")
		op.SetTags("Organizations")
		type UpdateOrganizationRequest struct {
			ID uuid.UUID `path:"id"`
			authz.UpdateOrganizationRequest
		}
		op.AddReqStructure(new(UpdateOrganizationRequest))
		op.AddRespStructure(new(authz.Organization), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
