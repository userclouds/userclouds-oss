// NOTE: automatically generated file -- DO NOT EDIT

package invite

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
		op, err := reflector.NewOperationContext(http.MethodPost, "/invite/send")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		op.SetSummary("Send Invite")
		op.SetDescription("This endpoint sends an invite to a user to join a tenant")
		op.SetTags("Invites")
		op.AddReqStructure(new(plex.SendInviteRequest))
		op.AddRespStructure(new(any), openapi.WithHTTPStatus(http.StatusNoContent))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusForbidden))
		op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}
}
