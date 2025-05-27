package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/config"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/provider"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/resources"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/schemas"
)

func handleSpec(ctx context.Context, specConfig *config.Spec, openapiDir string) {
	specPath := filepath.Join(openapiDir, specConfig.File)
	specText, err := os.ReadFile(specPath)
	if err != nil {
		log.Fatalf("failed to read openapi file from %s: %v", specPath, err)
	}

	var spec openapi3.Spec
	if err := spec.UnmarshalYAML(specText); err != nil {
		log.Fatalf("failed to unmarshal openapi file: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get wd: %v", err)
	}

	err = os.MkdirAll(wd+"/"+specConfig.GeneratedFilePackage, 0755)
	if err != nil {
		log.Fatalf("failed to create generated file package directory: %v", err)
	}

	schemas.GenSchemas(ctx, wd, specConfig, &spec)
	for _, resource := range specConfig.Resources {
		resources.GenResource(ctx, wd, specConfig.GeneratedFilePackage, &spec, &resource)
	}
}

func main() {
	ctx := context.Background()

	if len(os.Args) < 3 {
		log.Fatalf("Usage: genprovider [openapi source directory] [generation config file path]")
	}

	configText, err := os.ReadFile(os.Args[2])
	if err != nil {
		log.Fatalf("failed to read generation config file: %v", err)
	}

	var conf config.GenerationConfig
	err = json.Unmarshal(configText, &conf)
	if err != nil {
		log.Fatalf("failed to unmarshal generation config file: %v", err)
	}

	for _, spec := range conf.Specs {
		handleSpec(ctx, &spec, os.Args[1])
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get wd: %v", err)
	}
	provider.GenProvider(ctx, "provider", wd, &conf)
}
