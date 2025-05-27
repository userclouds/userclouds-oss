// NOTE: automatically generated file -- DO NOT EDIT

package authn

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/idp"
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/authn/baseprofiles")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List User Base Profiles")
		op.SetDescription("This endpoint returns a paginated list of user base profiles in a tenant. The list can be filtered to only include users inside a specified organization.")
		op.SetTags("Users")
		op.AddReqStructure(new(ListUserBaseProfilesParams))
		op.AddRespStructure(new(idp.ListUserBaseProfilesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/authn/users")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Users")
		op.SetDescription("This endpoint returns a paginated list of users in a tenant. The list can be filtered to only include users inside a specified organization.")
		op.SetTags("Users")
		op.AddReqStructure(new(ListUsersParams))
		op.AddRespStructure(new(idp.ListUsersResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/authn/users")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create User")
		op.SetDescription("This endpoint creates a user.")
		op.SetTags("Users")
		op.AddReqStructure(new(idp.CreateUserAndAuthnRequest))
		op.AddRespStructure(new(idp.UserResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/authn/users/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete User")
		op.SetDescription("This endpoint deletes a user by ID.")
		op.SetTags("Users")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/authn/users/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get User")
		op.SetDescription("This endpoint gets a user by ID.")
		op.SetTags("Users")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.UserResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/authn/users/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update User")
		op.SetDescription("This endpoint updates a specified user.")
		op.SetTags("Users")
		type UpdateUserRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateUserRequest
		}
		op.AddReqStructure(new(UpdateUserRequest))
		op.AddRespStructure(new(idp.UserResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
