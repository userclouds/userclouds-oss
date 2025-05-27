package manifest

import (
	"context"
	"testing"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog"
	"userclouds.com/test/testlogtransport"
)

func makeLiveResource(id string, name string) liveresource.Resource {
	return liveresource.Resource{
		TerraformTypeSuffix: "userstore_column",
		ResourceUUID:        id,
		Attributes: map[string]any{
			"name": name,
		},
	}
}

func TestMatchColumnsBasicCase(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	liveResources := []liveresource.Resource{
		makeLiveResource("fe20fd48-a006-4ad8-9208-4aad540d8794", "col1"),
		makeLiveResource("c860a6d7-c632-4f81-8f5f-597290a9f437", "col2"),
	}

	mfest := Manifest{
		Resources: []Resource{
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry1",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				},
				Attributes: map[string]any{
					"name": "col1",
				},
			},
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry2",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "c860a6d7-c632-4f81-8f5f-597290a9f437",
				},
				Attributes: map[string]any{
					"name": "col2",
				},
			},
		},
	}

	err := mfest.MatchLiveResources(context.Background(), &liveResources, "prod")
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 0)
	assert.Equal(t, liveResources[0].ManifestID, "entry1")
	assert.Equal(t, liveResources[1].ManifestID, "entry2")
	// matchColumnsToManifest should have updated the manifest with resource IDs for this specific
	// tenant
	assert.Equal(t, mfest.Resources[0].ResourceUUIDs["prod"], "fe20fd48-a006-4ad8-9208-4aad540d8794")
	assert.Equal(t, mfest.Resources[1].ResourceUUIDs["prod"], "c860a6d7-c632-4f81-8f5f-597290a9f437")
}

func TestMatchColumnsMissingLiveResources(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	mfest := Manifest{
		Resources: []Resource{
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry1",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				},
				Attributes: map[string]any{
					"name": "col1",
				},
			},
		},
	}

	err := mfest.MatchLiveResources(context.Background(), &[]liveresource.Resource{}, "prod")
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 0)
}

func TestMatchColumnsMissingManifestEntries(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	liveResources := []liveresource.Resource{
		makeLiveResource("fe20fd48-a006-4ad8-9208-4aad540d8794", "col1"),
	}
	mfest := Manifest{
		Resources: []Resource{},
	}

	err := mfest.MatchLiveResources(context.Background(), &liveResources, "prod")
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 1)
	assert.Equal(t, liveResources[0].ManifestID, "")
}

func TestMatchColumnsMatchingByName(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	// Create live columns with different IDs from what's in the manifest
	liveResources := []liveresource.Resource{
		makeLiveResource("633fac47-c6c1-4459-93e0-0bb4043e60a0", "col1"),
		makeLiveResource("dc42da22-4c49-459d-9572-3b5db6d61959", "col2"),
	}
	mfest := Manifest{
		Resources: []Resource{
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry1",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				},
				Attributes: map[string]any{
					"name": "col1",
				},
			},
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry2",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "c860a6d7-c632-4f81-8f5f-597290a9f437",
				},
				Attributes: map[string]any{
					"name": "col2",
				},
			},
		},
	}

	err := mfest.MatchLiveResources(context.Background(), &liveResources, "prod")
	assert.NoErr(t, err)
	// We should get warnings logged that the IDs didn't match
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 2)
	// But we should still end up with resolved manifest IDs
	assert.Equal(t, liveResources[0].ManifestID, "entry1")
	assert.Equal(t, liveResources[1].ManifestID, "entry2")
	// matchColumnsToManifest should have updated the manifest with the matched resource IDs
	assert.Equal(t, mfest.Resources[0].ResourceUUIDs["prod"], "633fac47-c6c1-4459-93e0-0bb4043e60a0")
	assert.Equal(t, mfest.Resources[1].ResourceUUIDs["prod"], "dc42da22-4c49-459d-9572-3b5db6d61959")
}

func TestMatchColumnsMatchingByIdPrioritized(t *testing.T) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	liveResources := []liveresource.Resource{
		makeLiveResource("fe20fd48-a006-4ad8-9208-4aad540d8794", "col1"),
		makeLiveResource("c860a6d7-c632-4f81-8f5f-597290a9f437", "col2"),
	}
	// Switch the column names to verify that we are prioritizing matching
	// columns by resource ID, not by name
	mfest := Manifest{
		Resources: []Resource{
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry1",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				},
				Attributes: map[string]any{
					"name": "col2",
				},
			},
			{
				TerraformTypeSuffix: "userstore_column",
				ManifestID:          "entry2",
				ResourceUUIDs: map[string]string{
					"__DEFAULT": "c860a6d7-c632-4f81-8f5f-597290a9f437",
				},
				Attributes: map[string]any{
					"name": "col1",
				},
			},
		},
	}

	err := mfest.MatchLiveResources(context.Background(), &liveResources, "prod")
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 0)
	assert.Equal(t, liveResources[0].ManifestID, "entry1")
	assert.Equal(t, liveResources[1].ManifestID, "entry2")
}
