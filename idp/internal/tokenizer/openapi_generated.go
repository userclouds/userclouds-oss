// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/swaggest/jsonschema-go"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/access")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Access Policies")
		op.SetDescription("This endpoint returns a list of access policies in a tenant. The list can be filtered to only include policies with a specified name or version.")
		op.SetTags("Access Policies")
		op.AddReqStructure(new(listAccessPoliciesParams))
		op.AddRespStructure(new(idp.ListAccessPoliciesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/policies/access")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Access Policy")
		op.SetDescription("This endpoint creates an access policy.")
		op.SetTags("Access Policies")
		op.AddReqStructure(new(tokenizer.CreateAccessPolicyRequest))
		op.AddRespStructure(new(policy.AccessPolicy), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/tokenizer/policies/access/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Access Policy")
		op.SetDescription("This endpoint deletes an access policy by ID.")
		op.SetTags("Access Policies")
		type DeleteAccessPolicyParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			DeleteAccessPolicyParams
		}
		op.AddReqStructure(new(DeleteAccessPolicyParamsAndPath))
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/access/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Access Policy")
		op.SetDescription("This endpoint gets an access policy.")
		op.SetTags("Access Policies")
		type GetAccessPolicyParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			GetAccessPolicyParams
		}
		op.AddReqStructure(new(GetAccessPolicyParamsAndPath))
		op.AddRespStructure(new(policy.AccessPolicy), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/tokenizer/policies/access/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Access Policy")
		op.SetDescription("This endpoint updates a specified access policy.")
		op.SetTags("Access Policies")
		type UpdateAccessPolicyRequest struct {
			ID uuid.UUID `path:"id"`
			tokenizer.UpdateAccessPolicyRequest
		}
		op.AddReqStructure(new(UpdateAccessPolicyRequest))
		op.AddRespStructure(new(policy.AccessPolicy), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/accesstemplate")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Access Policy Templates")
		op.SetDescription("This endpoint returns a paginated list of access policy templates in a tenant. The list can be filtered to only include templates with a specified name or version.")
		op.SetTags("Access Policy Templates")
		op.AddReqStructure(new(listAccessPolicyTemplatesParams))
		op.AddRespStructure(new(idp.ListAccessPolicyTemplatesResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/policies/accesstemplate")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Access Policy Template")
		op.SetDescription("This endpoint creates an access policy template.")
		op.SetTags("Access Policy Templates")
		op.AddReqStructure(new(tokenizer.CreateAccessPolicyTemplateRequest))
		op.AddRespStructure(new(policy.AccessPolicyTemplate), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/tokenizer/policies/accesstemplate/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Access Policy Template")
		op.SetDescription("This endpoint deletes an access policy template by ID.")
		op.SetTags("Access Policy Templates")
		type DeleteAccessPolicyTemplateParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			DeleteAccessPolicyTemplateParams
		}
		op.AddReqStructure(new(DeleteAccessPolicyTemplateParamsAndPath))
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/accesstemplate/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Access Policy Template")
		op.SetDescription("This endpoint gets an access policy template by ID.")
		op.SetTags("Access Policy Templates")
		type GetAccessPolicyTemplateParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			GetAccessPolicyTemplateParams
		}
		op.AddReqStructure(new(GetAccessPolicyTemplateParamsAndPath))
		op.AddRespStructure(new(policy.AccessPolicyTemplate), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/tokenizer/policies/accesstemplate/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Access Policy Template")
		op.SetDescription("This endpoint updates a specified access policy template.")
		op.SetTags("Access Policy Templates")
		type UpdateAccessPolicyTemplateRequest struct {
			ID uuid.UUID `path:"id"`
			tokenizer.UpdateAccessPolicyTemplateRequest
		}
		op.AddReqStructure(new(UpdateAccessPolicyTemplateRequest))
		op.AddRespStructure(new(policy.AccessPolicyTemplate), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/secret")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Secrets")
		op.SetDescription("This endpoint lists all secrets.")
		op.SetTags("Secrets")
		op.AddReqStructure(new(listSecretsParams))
		op.AddRespStructure(new(idp.ListSecretsResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/policies/secret")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Secret")
		op.SetDescription("This endpoint creates a new secret.")
		op.SetTags("Secrets")
		op.AddReqStructure(new(tokenizer.CreateSecretRequest))
		op.AddRespStructure(new(policy.Secret), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/tokenizer/policies/secret/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Secret")
		op.SetDescription("This endpoint deletes a secret.")
		op.SetTags("Secrets")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/transformation")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("List Transformers")
		op.SetDescription("This endpoint returns a paginated list of all transformers in a tenant. The list can be filtered to only include transformers with a specific name.")
		op.SetTags("Transformers")
		op.AddReqStructure(new(listTransformersParams))
		op.AddRespStructure(new(idp.ListTransformersResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/policies/transformation")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Transformer")
		op.SetDescription("This endpoint creates a new transformer.")
		op.SetTags("Transformers")
		op.AddReqStructure(new(tokenizer.CreateTransformerRequest))
		op.AddRespStructure(new(policy.Transformer), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/tokenizer/policies/transformation/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Transformer")
		op.SetDescription("This endpoint deletes a transformer by ID.")
		op.SetTags("Transformers")
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
		op, err := reflector.NewOperationContext(http.MethodGet, "/tokenizer/policies/transformation/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Get Transformer")
		op.SetDescription("This endpoint gets a transformer by ID.")
		op.SetTags("Transformers")
		type GetTransformerParamsAndPath struct {
			ID uuid.UUID `path:"id"`
			GetTransformerParams
		}
		op.AddReqStructure(new(GetTransformerParamsAndPath))
		op.AddRespStructure(new(policy.Transformer), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPut, "/tokenizer/policies/transformation/{id}")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Update Transformer")
		op.SetDescription("This endpoint updates a transformer.")
		op.SetTags("Transformers")
		type UpdateTransformerRequest struct {
			ID uuid.UUID `path:"id"`
			tokenizer.UpdateTransformerRequest
		}
		op.AddReqStructure(new(UpdateTransformerRequest))
		op.AddRespStructure(new(policy.Transformer), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodDelete, "/tokenizer/tokens")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Delete Token")
		op.SetDescription("This endpoint deletes a token by ID.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(deleteTokenParams))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/tokens")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Create Token")
		op.SetDescription("This endpoint creates a token for a piece of data. CreateToken will always generate a unique token. If you want to reuse a token that was already generated, use LookupToken.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(tokenizer.CreateTokenRequest))
		op.AddRespStructure(new(tokenizer.CreateTokenResponse), openapi.WithHTTPStatus(http.StatusCreated))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/tokens/actions/inspect")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Inspect Token")
		op.SetDescription("This endpoint gets a token. It is a primarily a debugging API that allows you to query a token without resolving it.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(tokenizer.InspectTokenRequest))
		op.AddRespStructure(new(tokenizer.InspectTokenResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/tokens/actions/lookup")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Lookup Token")
		op.SetDescription("This endpoint helps you re-use existing tokens. It receives a piece of data and an access policy. It returns existing tokens that match across the full set of parameters. If no token matches, an error is returned.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(tokenizer.LookupTokensRequest))
		op.AddRespStructure(new(tokenizer.LookupTokensResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/tokens/actions/lookuporcreate")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Lookup Or Create Tokens")
		op.SetDescription("This endpoint helps you re-use existing tokens by only creating new tokens when they don't exist already.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(tokenizer.LookupOrCreateTokensRequest))
		op.AddRespStructure(new(tokenizer.LookupOrCreateTokensResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusConflict))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	{
		op, err := reflector.NewOperationContext(http.MethodPost, "/tokenizer/tokens/actions/resolve")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Resolve Token")
		op.SetDescription("This endpoint receives a list of tokens, applies the associated access policy for each token, and returns the associated token data if the conditions of the access policy are met.")
		op.SetTags("Tokens")
		op.AddReqStructure(new(tokenizer.ResolveTokensRequest))
		op.AddRespStructure(new([]tokenizer.ResolveTokenResponse), openapi.WithHTTPStatus(http.StatusOK))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusNotFound))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
