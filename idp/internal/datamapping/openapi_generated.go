// NOTE: automatically generated file -- DO NOT EDIT

package datamapping

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/idp/datamapping"
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/datamapping/datasource")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Data Sources")
		op.SetDescription("This endpoint lists all Data Sources in a tenant.")
		op.SetTags("Data Sources")
		op.AddReqStructure(new(listDataSourcesParams))
		op.AddRespStructure(new(datamapping.ListDataSourcesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/datamapping/datasource")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Data Source")
		op.SetDescription("This endpoint creates a Data Source.")
		op.SetTags("Data Sources")
		op.AddReqStructure(new(datamapping.CreateDataSourceRequest))
		op.AddRespStructure(new(datamapping.DataSource), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/datamapping/datasource/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Data Source")
		op.SetDescription("This endpoint deletes a Data Source by ID.")
		op.SetTags("Data Sources")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/datamapping/datasource/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Data Source")
		op.SetDescription("This endpoint gets a Data Source by ID.")
		op.SetTags("Data Sources")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(datamapping.DataSource), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/datamapping/datasource/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Data Source")
		op.SetDescription("This endpoint updates a specified Data Source.")
		op.SetTags("Data Sources")
		type UpdateDataSourceRequest struct {
			ID uuid.UUID `path:"id"`
			datamapping.UpdateDataSourceRequest
		}
		op.AddReqStructure(new(UpdateDataSourceRequest))
		op.AddRespStructure(new(datamapping.DataSource), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/datamapping/element")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Data Source Elements")
		op.SetDescription("This endpoint lists all Data Sources in a tenant.")
		op.SetTags("Data Sources")
		op.AddReqStructure(new(listDataSourceElementsParams))
		op.AddRespStructure(new(datamapping.ListDataSourceElementsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/datamapping/element")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Data Source")
		op.SetDescription("This endpoint creates a Data Source.")
		op.SetTags("Data Sources")
		op.AddReqStructure(new(datamapping.CreateDataSourceElementRequest))
		op.AddRespStructure(new(datamapping.DataSourceElement), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/datamapping/element/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Data Source")
		op.SetDescription("This endpoint deletes a Data Source by ID.")
		op.SetTags("Data Sources")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/datamapping/element/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Data Source")
		op.SetDescription("This endpoint gets a Data Source by ID.")
		op.SetTags("Data Sources")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(datamapping.DataSourceElement), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/datamapping/element/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Data Source Element")
		op.SetDescription("This endpoint updates a specified Data Source.")
		op.SetTags("Data Sources")
		type UpdateDataSourceElementRequest struct {
			ID uuid.UUID `path:"id"`
			datamapping.UpdateDataSourceElementRequest
		}
		op.AddReqStructure(new(UpdateDataSourceElementRequest))
		op.AddRespStructure(new(datamapping.DataSourceElement), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
