package tfconfig

import (
	"strings"
	"testing"

	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/infra/assert"
)

func TestGenConfig(t *testing.T) {
	config := manifest.Manifest{
		Resources: []manifest.Resource{
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry1",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				},
				Attributes: map[string]any{
					"name":       "col1",
					"type":       "uuid",
					"is_array":   false,
					"index_type": "none",
				},
			},
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry2",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "c860a6d7-c632-4f81-8f5f-597290a9f437",
				},
				Attributes: map[string]any{
					"name":       "col2",
					"type":       "string",
					"is_array":   true,
					"index_type": "unique",
				},
			},
		},
	}
	terraform, err := GenConfig(&GenerationContext{
		Manifest:                    &config,
		FQTN:                        "mycompany-prod",
		TFProviderVersionConstraint: ">= 0.0.1",
	})
	assert.NoErr(t, err)
	assert.Equal(t, strings.TrimSpace(terraform), strings.TrimSpace(`
terraform {
  required_providers {
    userclouds = {
      source  = "registry.terraform.io/userclouds/userclouds"
      version = ">= 0.0.1"
    }
  }
}

provider "userclouds" {
}

resource "userclouds_userstore_column" "manifestid-entry1" {
  id         = "fe20fd48-a006-4ad8-9208-4aad540d8794"
  index_type = "none"
  is_array   = false
  name       = "col1"
  type       = "uuid"
}

resource "userclouds_userstore_column" "manifestid-entry2" {
  id         = "c860a6d7-c632-4f81-8f5f-597290a9f437"
  index_type = "unique"
  is_array   = true
  name       = "col2"
  type       = "string"
}`))
}
