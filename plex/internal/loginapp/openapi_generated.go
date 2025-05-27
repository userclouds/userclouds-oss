// NOTE: automatically generated file -- DO NOT EDIT

package loginapp

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/infra/uclog"
	"userclouds.com/plex"
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/loginapp/register")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Login Applications")
		op.SetDescription("This endpoint lists all login applications for the tenant. Only tenant admins can list login applications.")
		op.SetTags("Login Application")
		op.AddReqStructure(new(listLoginAppsParams))
		op.AddRespStructure(new([]plex.LoginAppResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/loginapp/register")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Login Application")
		op.SetDescription("This endpoint creates a new login application for the tenant. Only tenant admins can create login applications.")
		op.SetTags("Login Application")
		op.AddReqStructure(new(plex.LoginAppRequest))
		op.AddRespStructure(new(plex.LoginAppResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/loginapp/register/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Login Application")
		op.SetDescription("This endpoint deletes a login application for the tenant. Only tenant admins can delete login applications.")
		op.SetTags("Login Application")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/loginapp/register/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Login Application")
		op.SetDescription("This endpoint retrieves a login application for the tenant. Only tenant admins can get login applications.")
		op.SetTags("Login Application")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(plex.LoginAppResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/loginapp/register/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Login Application")
		op.SetDescription("This endpoint updates a login application for the tenant. Only tenant admins can update login applications.")
		op.SetTags("Login Application")
		type LoginAppRequest struct {
			ID uuid.UUID `path:"id"`
			plex.LoginAppRequest
		}
		op.AddReqStructure(new(LoginAppRequest))
		op.AddRespStructure(new(plex.LoginAppResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
