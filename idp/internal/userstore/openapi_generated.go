// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
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
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/api/accessors")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Execute Accessor")
		op.SetDescription("This endpoint executes a specified accessor (custom read API).")
		op.SetTags("Accessors")
		op.AddReqStructure(new(executeAccessorHandlerRequest))
		op.AddRespStructure(new(idp.ExecuteAccessorResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusTooManyRequests))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/api/consentedpurposes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Purposes for User")
		op.SetDescription("This endpoint lists all consented purposes for a specified user.")
		op.SetTags("Purposes")
		op.AddReqStructure(new(idp.GetConsentedPurposesForUserRequest))
		op.AddRespStructure(new(idp.GetConsentedPurposesForUserResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/api/mutators")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Execute Mutator")
		op.SetDescription("This endpoint executes a specified mutator (custom write API).")
		op.SetTags("Mutators")
		op.AddReqStructure(new(idp.ExecuteMutatorRequest))
		op.AddRespStructure(new(idp.ExecuteMutatorResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusTooManyRequests))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/api/users")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create User With Mutator")
		op.SetDescription("This endpoint creates a user and updates it with the specified mutator.")
		op.SetTags("Users")
		op.AddReqStructure(new(idp.CreateUserWithMutatorRequest))
		op.AddRespStructure(new(uuid.UUID), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/api/users/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete User")
		op.SetDescription("This endpoint deletes a user by ID")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/accessors")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Accessors")
		op.SetDescription("This endpoint lists all accessors in a tenant.")
		op.SetTags("Accessors")
		op.AddReqStructure(new(listAccessorsParams))
		op.AddRespStructure(new(idp.ListAccessorsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/accessors")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Accessor")
		op.SetDescription("This endpoint creates an accessor - a custom read API.")
		op.SetTags("Accessors")
		op.AddReqStructure(new(idp.CreateAccessorRequest))
		op.AddRespStructure(new(userstore.Accessor), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/accessors/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Accessor")
		op.SetDescription("This endpoint deletes an accessor by ID.")
		op.SetTags("Accessors")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/accessors/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Accessor")
		op.SetDescription("This endpoint gets an existing accessor's configuration by ID.")
		op.SetTags("Accessors")
		type GetAccessorParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			GetAccessorParams
		}
		op.AddReqStructure(new(GetAccessorParamsAndPath))
		op.AddRespStructure(new(userstore.Accessor), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/accessors/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Accessor")
		op.SetDescription("This endpoint updates a specified accessor.")
		op.SetTags("Accessors")
		type UpdateAccessorRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateAccessorRequest
		}
		op.AddReqStructure(new(UpdateAccessorRequest))
		op.AddRespStructure(new(userstore.Accessor), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Columns")
		op.SetDescription("This endpoint returns a paginated list of all columns in a tenant.")
		op.SetTags("Columns")
		op.AddReqStructure(new(listColumnsParams))
		op.AddRespStructure(new(idp.ListColumnsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/columns")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Column")
		op.SetDescription("This endpoint creates a new column.")
		op.SetTags("Columns")
		op.AddReqStructure(new(idp.CreateColumnRequest))
		op.AddRespStructure(new(userstore.Column), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/columns/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Column")
		op.SetDescription("This endpoint deletes a column by ID. Note that deleting the column doesn't result in data deletion - it just results in the data being immediately unavailable. To delete the data stored in the column, you need to trigger the garbage collection process on the column which will remove the data after a configurable retention period.")
		op.SetTags("Columns")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Column")
		op.SetDescription("This endpoint gets a column's configuration by ID.")
		op.SetTags("Columns")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(userstore.Column), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/columns/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Column")
		op.SetDescription("This endpoint updates a specified column. Some column characteristics cannot be changed in an Update operation, once the column contains data. A column update may invalidate the accessors defined for it.")
		op.SetTags("Columns")
		type UpdateColumnRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRequest
		}
		op.AddReqStructure(new(UpdateColumnRequest))
		op.AddRespStructure(new(userstore.Column), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns/{id}/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default Live ColumnRetentionDurations for Column")
		op.SetDescription("This endpoint gets the default Live column purpose ColumnRetentionDurations for a tenant column, one for each column purpose. For each retention duration, if the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/columns/{id}/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Live ColumnRetentionDurations for Column")
		op.SetDescription("This endpoint updates all specified Live column purpose ColumnRetentionDurations for a tenant column. For each retention duration, if id is nil and use_default is false, the retention duration will be created; if id is non-nil and use_default is false, the associated retention duration will be updated; or if id is non-nil and use_default is true, the associated retention duration will be deleted. Each column purpose retention duration that has been deleted will fall back to the associated purpose retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationsRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationsRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationsRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/columns/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Live ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint deletes a specific Live column purpose ColumnRetentionDuration for a tenant column. Once the column purpose retention duration has been deleted, it will fall back to the associated purpose retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Live ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint gets a specific Live column purpose ColumnRetentionDuration for a tenant column.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/columns/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Live ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint updates a specific Live column purpose ColumnRetentionDuration for a tenant column.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns/{id}/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default SoftDeleted ColumnRetentionDurations for Column")
		op.SetDescription("This endpoint gets the default SoftDeleted column purpose ColumnRetentionDurations for a tenant column, one for each column purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. For each retention duration, if the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/columns/{id}/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update SoftDeleted ColumnRetentionDurations for Column")
		op.SetDescription("This endpoint updates all specified SoftDeleted column purpose ColumnRetentionDurations for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. For each retention duration, if id is nil and use_default is false, the retention duration will be created; if id is non-nil and use_default is false, the associated retention duration will be updated; or if id is non-nil and use_default is true, the associated retention duration will be deleted. Each column purpose retention duration that has been deleted will fall back to the associated purpose retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationsRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationsRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationsRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/columns/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete SoftDeleted ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint deletes a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the column purpose retention duration has been deleted, it will fall back to the associated purpose retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/columns/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get SoftDeleted ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint gets a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/columns/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update SoftDeleted ColumnRetentionDuration for Column")
		op.SetDescription("This endpoint updates a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/datatypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Data Types")
		op.SetDescription("This endpoint returns a paginated list of all data types in a tenant.")
		op.SetTags("Data Types")
		op.AddReqStructure(new(listDataTypesParams))
		op.AddRespStructure(new(idp.ListDataTypesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/datatypes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Data Type")
		op.SetDescription("This endpoint creates a new data type.")
		op.SetTags("Data Types")
		op.AddReqStructure(new(idp.CreateDataTypeRequest))
		op.AddRespStructure(new(userstore.ColumnDataType), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/datatypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Data Type")
		op.SetDescription("This endpoint deletes a data type by ID.")
		op.SetTags("Data Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/datatypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Data Type")
		op.SetDescription("This endpoint gets a data type's configuration by ID.")
		op.SetTags("Data Types")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(userstore.ColumnDataType), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/datatypes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Data Type")
		op.SetDescription("This endpoint updates a specified data type.")
		op.SetTags("Data Types")
		type UpdateDataTypeRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateDataTypeRequest
		}
		op.AddReqStructure(new(UpdateDataTypeRequest))
		op.AddRespStructure(new(userstore.ColumnDataType), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default Live ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint gets the default Live ColumnRetentionDuration for a tenant. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		op.AddReqStructure(nil)
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Live ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint creates a Live ColumnRetentionDuration for a tenant.")
		op.SetTags("ColumnRetentionDurations")
		op.AddReqStructure(new(idp.UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/liveretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Live ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint deletes a specific Live ColumnRetentionDuration for a tenant. Once the tenant default retention duration has been deleted, it will fall back to the system default to retain Live data indefinitely.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/liveretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Live ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint gets a specific Live ColumnRetentionDuration for a tenant.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/liveretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Live ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint updates a specific Live ColumnRetentionDuration for a tenant.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/mutators")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Mutators")
		op.SetDescription("This endpoint lists all mutators in a tenant.")
		op.SetTags("Mutators")
		op.AddReqStructure(new(listMutatorsParams))
		op.AddRespStructure(new(idp.ListMutatorsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/mutators")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Mutator")
		op.SetDescription("This endpoint creates a mutator.")
		op.SetTags("Mutators")
		op.AddReqStructure(new(idp.CreateMutatorRequest))
		op.AddRespStructure(new(userstore.Mutator), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/mutators/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Mutator")
		op.SetDescription("This endpoint deletes a mutator by ID.")
		op.SetTags("Mutators")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/mutators/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Mutator")
		op.SetDescription("This endpoint gets a mutator by ID.")
		op.SetTags("Mutators")
		type GetMutatorParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			GetMutatorParams
		}
		op.AddReqStructure(new(GetMutatorParamsAndPath))
		op.AddRespStructure(new(userstore.Mutator), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/mutators/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Mutator")
		op.SetDescription("This endpoint updates a specified mutator.")
		op.SetTags("Mutators")
		type UpdateMutatorRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateMutatorRequest
		}
		op.AddReqStructure(new(UpdateMutatorRequest))
		op.AddRespStructure(new(userstore.Mutator), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Purposes")
		op.SetDescription("This endpoint returns a paginated list of all purposes in a tenant.")
		op.SetTags("Purposes")
		op.AddReqStructure(new(listPurposesParams))
		op.AddRespStructure(new(idp.ListPurposesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/purposes")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Purpose")
		op.SetDescription("This endpoint creates a purpose.")
		op.SetTags("Purposes")
		op.AddReqStructure(new(idp.CreatePurposeRequest))
		op.AddRespStructure(new(userstore.Purpose), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/purposes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Purpose")
		op.SetDescription("This endpoint deletes a purpose by ID.")
		op.SetTags("Purposes")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Purpose")
		op.SetDescription("This endpoint gets a purpose by ID.")
		op.SetTags("Purposes")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(userstore.Purpose), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/purposes/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Purpose")
		op.SetDescription("This endpoint updates a specified purpose.")
		op.SetTags("Purposes")
		type UpdatePurposeRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdatePurposeRequest
		}
		op.AddReqStructure(new(UpdatePurposeRequest))
		op.AddRespStructure(new(userstore.Purpose), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes/{id}/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default Live ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint gets the default Live ColumnRetentionDuration for a tenant purpose. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/purposes/{id}/liveretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Live ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint creates a Live ColumnRetentionDuration for a tenant purpose.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/purposes/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Live ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint deletes a specific Live ColumnRetentionDuration for a tenant purpose. If the purpose default retention duration has been deleted, it will fall back to the tenant retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Live ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint gets a specific Live ColumnRetentionDuration for a tenant purpose.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/purposes/{id}/liveretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Live ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint updates a specific Live ColumnRetentionDuration for a tenant purpose.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes/{id}/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default SoftDeleted ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint gets the default SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/purposes/{id}/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create SoftDeleted ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint creates a SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/purposes/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete SoftDeleted ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint deletes a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the purpose default retention duration has been deleted, it will fall back to the tenant retention duration.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/purposes/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get SoftDeleted ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint gets a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/purposes/{id}/softdeletedretentiondurations/{id2}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update SoftDeleted ColumnRetentionDuration for Purpose")
		op.SetDescription("This endpoint updates a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID  uuid.UUID `path:"id"`
			ID2 uuid.UUID `path:"id2"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/searchindices")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List User Search Indexes")
		op.SetDescription("This endpoint returns a paginated list of all user search indices in a tenant.")
		op.SetTags("User Search Indices")
		op.AddReqStructure(new(listUserSearchIndicesParams))
		op.AddRespStructure(new(idp.ListUserSearchIndicesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/searchindices")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create User Search Index")
		op.SetDescription("This endpoint creates a new user search index.")
		op.SetTags("User Search Indices")
		op.AddReqStructure(new(idp.CreateUserSearchIndexRequest))
		op.AddRespStructure(new(search.UserSearchIndex), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/searchindices/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete User Search Index")
		op.SetDescription("This endpoint deletes a user search index by ID.")
		op.SetTags("User Search Indices")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/searchindices/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get User Search Index")
		op.SetDescription("This endpoint gets a user search index's configuration by ID.")
		op.SetTags("User Search Indices")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(search.UserSearchIndex), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/searchindices/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update User Search Index")
		op.SetDescription("This endpoint updates a specified user search index.")
		op.SetTags("User Search Indices")
		type UpdateUserSearchIndexRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateUserSearchIndexRequest
		}
		op.AddReqStructure(new(UpdateUserSearchIndexRequest))
		op.AddRespStructure(new(search.UserSearchIndex), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Default SoftDeleted ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint gets the default SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.")
		op.SetTags("ColumnRetentionDurations")
		op.AddReqStructure(nil)
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/userstore/config/softdeletedretentiondurations")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create SoftDeleted ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint creates a SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		op.AddReqStructure(new(idp.UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/userstore/config/softdeletedretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete SoftDeleted ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint deletes a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the tenant default retention duration has been deleted, it will fall back to the system default to not retain deleted data.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/userstore/config/softdeletedretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get SoftDeleted ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint gets a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type RequestAndPath struct {
			ID uuid.UUID `path:"id"`
		}
		op.AddReqStructure(new(RequestAndPath))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/userstore/config/softdeletedretentiondurations/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update SoftDeleted ColumnRetentionDuration for Tenant")
		op.SetDescription("This endpoint updates a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.")
		op.SetTags("ColumnRetentionDurations")
		type UpdateColumnRetentionDurationRequest struct {
			ID uuid.UUID `path:"id"`
			idp.UpdateColumnRetentionDurationRequest
		}
		op.AddReqStructure(new(UpdateColumnRetentionDurationRequest))
		op.AddRespStructure(new(idp.ColumnRetentionDurationResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
