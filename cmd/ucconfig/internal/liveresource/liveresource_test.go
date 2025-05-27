package liveresource

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
)

func TestTransformValue(t *testing.T) {
	// Testing an array of structs of userstore.ResourceIDs. Should be complicated enough to have
	// somewhat decent coverage...
	val := []userstore.ColumnOutputConfig{{
		Column: userstore.ResourceID{
			ID:   uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794")),
			Name: "demo",
		},
		Transformer: userstore.ResourceID{
			ID:   uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794")),
			Name: "demo",
		},
	}}
	out, err := transformValue(val)
	assert.NoErr(t, err)
	assert.Equal(t, out.([]any)[0].(map[string]any)["column"], "fe20fd48-a006-4ad8-9208-4aad540d8794")
}

type DemoResource struct {
	ID         uuid.UUID `json:"id" yaml:"id"`
	Name       string    `json:"name" yaml:"name"`
	IsSystem   bool      `json:"is_system" yaml:"is_system"`
	StringAttr string    `json:"string_attr" yaml:"string_attr"`
	IntAttr    int       `json:"int_attr" yaml:"int_attr"`
}

func TestGetLiveResourcesForType(t *testing.T) {
	resourceType := resourcetypes.ResourceType{
		TerraformTypeSuffix: "demo",
		ListResources: func(ctx context.Context, client *idp.Client) ([]any, error) {
			return []any{
				DemoResource{
					ID:         uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794")),
					Name:       "demo_resource",
					IsSystem:   true,
					StringAttr: "hello world",
					IntAttr:    7,
				},
			}, nil
		},
	}
	live, err := GetLiveResourcesForType(context.Background(), nil, resourceType)
	assert.NoErr(t, err)
	assert.Equal(t, live, []Resource{{
		TerraformTypeSuffix: "demo",
		ManifestID:          "",
		ResourceUUID:        "fe20fd48-a006-4ad8-9208-4aad540d8794",
		IsSystem:            true,
		Attributes: map[string]any{
			"name":        "demo_resource",
			"string_attr": "hello world",
			"int_attr":    7,
		},
	}})
}

func TestVersionAttributeOmitted(t *testing.T) {
	res, err := MakeLiveResource(context.Background(), *resourcetypes.GetByTerraformTypeSuffix("userstore_accessor"), userstore.Accessor{
		ID:      uuid.Must(uuid.FromString("fe20fd48-a006-4ad8-9208-4aad540d8794")),
		Version: 7,
		Name:    "TestAccessor",
	})
	assert.NoErr(t, err)
	assert.Equal(t, res.Attributes["name"], "TestAccessor")
	assert.Equal(t, res.Attributes["version"], nil)
}
