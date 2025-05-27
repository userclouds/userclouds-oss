package tfstate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
)

func TestCreateState(t *testing.T) {
	resources := []liveresource.Resource{
		{
			TerraformTypeSuffix: "userstore_column",
			ManifestID:          "entry1",
			ResourceUUID:        "fe20fd48-a006-4ad8-9208-4aad540d8794",
		},
		{
			TerraformTypeSuffix: "userstore_column",
			ManifestID:          "entry2",
			ResourceUUID:        "c860a6d7-c632-4f81-8f5f-597290a9f437",
		},
	}
	state, err := CreateState(&resources)
	assert.NoErr(t, err)
	// Set non-random lineage
	state.Lineage = "random-uuid"
	marshalled, err := json.MarshalIndent(state, "", "  ")
	assert.NoErr(t, err)
	assert.Equal(t, string(marshalled), `{
  "version": 4,
  "terraform_version": "1.5.3",
  "serial": 1,
  "lineage": "random-uuid",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "manifestid-entry1",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "fe20fd48-a006-4ad8-9208-4aad540d8794"
          },
          "sensitive_attributes": [],
          "dependencies": []
        }
      ]
    },
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "manifestid-entry2",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "id": "c860a6d7-c632-4f81-8f5f-597290a9f437"
          },
          "sensitive_attributes": [],
          "dependencies": []
        }
      ]
    }
  ],
  "check_results": []
}`)
}

func TestCreateStateWithDependencies(t *testing.T) {
	ctx := context.Background()
	col1ID := uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794"))
	col1, err := liveresource.MakeLiveResource(ctx, *resourcetypes.GetByTerraformTypeSuffix("userstore_column"), userstore.Column{
		ID:        col1ID,
		Name:      "col1",
		IndexType: "none",
	})
	assert.NoErr(t, err)
	col2ID := uuid.Must(uuid.FromString("c860a6d7-c632-4f81-8f5f-597290a9f437"))
	col2, err := liveresource.MakeLiveResource(ctx, *resourcetypes.GetByTerraformTypeSuffix("userstore_column"), userstore.Column{
		ID:        col2ID,
		Name:      "col2",
		IndexType: "none",
	})
	assert.NoErr(t, err)
	accessor, err := liveresource.MakeLiveResource(ctx, *resourcetypes.GetByTerraformTypeSuffix("userstore_accessor"), userstore.Accessor{
		ID: uuid.Must(uuid.FromString("a12b3c4d-5e67-8901-2f34-567890123456")),
		Columns: []userstore.ColumnOutputConfig{
			{Column: userstore.ResourceID{ID: col1ID}},
			{Column: userstore.ResourceID{ID: col2ID}},
		},
	})
	assert.NoErr(t, err)
	resources := []liveresource.Resource{col1, col2, accessor}
	state, err := CreateState(&resources)
	assert.NoErr(t, err)
	// Set non-random lineage
	state.Lineage = "random-uuid"
	marshalled, err := json.MarshalIndent(state, "", "  ")
	assert.NoErr(t, err)
	assert.Equal(t, string(marshalled), `{
  "version": 4,
  "terraform_version": "1.5.3",
  "serial": 1,
  "lineage": "random-uuid",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "unmatched-fe20fd48-a006-4ad8-9208-4aad540d8794",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "data_type": "00000000-0000-0000-0000-000000000000",
            "id": "fe20fd48-a006-4ad8-9208-4aad540d8794",
            "index_type": "none",
            "is_array": false,
            "name": "col1"
          },
          "sensitive_attributes": [],
          "dependencies": []
        }
      ]
    },
    {
      "mode": "managed",
      "type": "userclouds_userstore_column",
      "name": "unmatched-c860a6d7-c632-4f81-8f5f-597290a9f437",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "data_type": "00000000-0000-0000-0000-000000000000",
            "id": "c860a6d7-c632-4f81-8f5f-597290a9f437",
            "index_type": "none",
            "is_array": false,
            "name": "col2"
          },
          "sensitive_attributes": [],
          "dependencies": []
        }
      ]
    },
    {
      "mode": "managed",
      "type": "userclouds_userstore_accessor",
      "name": "unmatched-a12b3c4d-5e67-8901-2f34-567890123456",
      "provider": "provider[\"registry.terraform.io/userclouds/userclouds\"]",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "access_policy": "00000000-0000-0000-0000-000000000000",
            "columns": [
              {
                "column": "fe20fd48-a006-4ad8-9208-4aad540d8794"
              },
              {
                "column": "c860a6d7-c632-4f81-8f5f-597290a9f437"
              }
            ],
            "id": "a12b3c4d-5e67-8901-2f34-567890123456",
            "name": "",
            "purposes": null,
            "selector_config": {}
          },
          "sensitive_attributes": [],
          "dependencies": [
            "userclouds_userstore_column.unmatched-fe20fd48-a006-4ad8-9208-4aad540d8794",
            "userclouds_userstore_column.unmatched-c860a6d7-c632-4f81-8f5f-597290a9f437"
          ]
        }
      ]
    }
  ],
  "check_results": []
}`)
}
