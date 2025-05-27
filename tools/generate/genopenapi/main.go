package genopenapi

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/swaggest/openapi-go/openapi3"

	authzRoutes "userclouds.com/authz/routes"
	idpRoutes "userclouds.com/idp/routes"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	plexRoutes "userclouds.com/plex/routes"
)

const (
	specVersion = "3.0.3"
)

// Run the openapi generator
func Run(ctx context.Context) {
	uclog.Infof(ctx, "Generating OpenAPI Spec v%s", specVersion)
	reflector := openapi3.Reflector{Spec: &openapi3.Spec{Openapi: specVersion}}
	authzRoutes.BuildAuthZOpenAPISpec(ctx, &reflector)
	if err := writeOpenAPISpecToFile(ctx, &reflector, "openapi/authz", "Authorization", "AuthZ OpenAPI Spec", "1.0.0"); err != nil {
		log.Fatal(err.Error())
	}

	reflector = openapi3.Reflector{Spec: &openapi3.Spec{Openapi: specVersion}}
	idpRoutes.BuildAuthNOpenAPISpec(ctx, &reflector)
	if err := writeOpenAPISpecToFile(ctx, &reflector, "openapi/authn", "Authentication", "AuthN OpenAPI Spec", "1.0.0"); err != nil {
		uclog.Fatalf(ctx, "Failed to write AuthN OpenAPI spec: %v", err)
	}

	reflector = openapi3.Reflector{Spec: &openapi3.Spec{Openapi: specVersion}}
	idpRoutes.BuildTokenizerOpenAPISpec(ctx, &reflector)
	if err := writeOpenAPISpecToFile(ctx, &reflector, "openapi/tokenizer", "Tokenizer", "Tokenizer OpenAPI Spec", "1.0.0"); err != nil {
		uclog.Fatalf(ctx, "Failed to write Tokenizer OpenAPI spec: %v", err)
	}

	reflector = openapi3.Reflector{Spec: &openapi3.Spec{Openapi: specVersion}}
	idpRoutes.BuildUserstoreOpenAPISpec(ctx, &reflector)
	if err := writeOpenAPISpecToFile(ctx, &reflector, "openapi/userstore", "User Store", "User Store OpenAPI Spec", "1.0.0"); err != nil {
		uclog.Fatalf(ctx, "Failed to write User Store OpenAPI spec: %v", err)
	}

	reflector = openapi3.Reflector{Spec: &openapi3.Spec{Openapi: specVersion}}
	plexRoutes.BuildOpenAPISpec(ctx, &reflector)
	if err := writeOpenAPISpecToFile(ctx, &reflector, "openapi/plex", "Plex", "Plex OpenAPI Spec", "1.0.0"); err != nil {
		uclog.Fatalf(ctx, "Failed to write Plex OpenAPI spec: %v", err)
	}
}

func writeOpenAPISpecToFile(ctx context.Context, reflector *openapi3.Reflector, baseFilename string, title string, description string, version string) error {
	additionalSections := map[string]any{
		"x-readme": map[string]bool{
			"explorer-enabled": false,
		},
		"servers": []map[string]string{
			{"url": "https://your-tenant-name.tenant.userclouds.com"},
		},
	}
	reflector.Spec.
		WithMapOfAnything(additionalSections).
		Info.
		WithTitle(title).
		WithDescription(description).
		WithVersion(version)

	schemaYAML, err := reflector.Spec.MarshalYAML()
	if err != nil {
		return ucerr.Wrap(err)
	}
	fn := fmt.Sprintf("./%s.yaml", baseFilename)
	fh, err := os.Create(fn)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Writing OpenAPI Spec to %s", fn)
	if _, err := fh.Write(schemaYAML); err != nil {
		return ucerr.Wrap(err)
	}
	if err := fh.Close(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
