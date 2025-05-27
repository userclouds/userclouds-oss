package manifest

import (
	"encoding/json"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
)

func parseAndValidateJSON(jsonManifest string, fqtn string) (Manifest, error) {
	parsed := Manifest{}
	err := json.Unmarshal([]byte(jsonManifest), &parsed)
	if err != nil {
		return Manifest{}, err
	}
	return parsed, parsed.Validate(fqtn)
}

func TestParseManifest(t *testing.T) {
	jsonManifest := `{
		"resources": [
			{
				"uc_terraform_type": "userstore_column",
            	"manifest_id": "fe20fd48-a006-4ad8-9208-4aad540d8794",
            	"resource_uuids": {
            	    "__DEFAULT": "fe20fd48-a006-4ad8-9208-4aad540d8794"
            	},
            	"attributes": {
            	    "name": "id",
            	    "type": "uuid",
            	    "is_array": false,
            	    "index_type": "none"
            	}
			}
		]
	}`
	parsed, err := parseAndValidateJSON(jsonManifest, "mycompany-prod")
	assert.NoErr(t, err)
	assert.Equal(t, len(parsed.Resources), 1)
	assert.Equal(t, parsed.Resources[0].ManifestID, "fe20fd48-a006-4ad8-9208-4aad540d8794")
	assert.Equal(t, parsed.Resources[0].ResourceUUIDs["__DEFAULT"], "fe20fd48-a006-4ad8-9208-4aad540d8794")
	assert.Equal(t, parsed.Resources[0].Attributes["name"], "id")
	assert.Equal(t, parsed.Resources[0].Attributes["type"], "uuid")
	assert.Equal(t, parsed.Resources[0].Attributes["is_array"], false)
	assert.Equal(t, parsed.Resources[0].Attributes["index_type"], "none")
}

func TestRejectsInvalidResourceType(t *testing.T) {
	jsonManifest := `{
		"resources": [
			{
				"manifest_id": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				"resource_uuids": {
					"staging": "fe20fd48-a006-4ad8-9208-4aad540d8794"
				},
				"attributes": {}
			}
		]
	}`
	_, err := parseAndValidateJSON(jsonManifest, "mycompany-staging")
	assert.True(t, err != nil && strings.Contains(err.Error(), "uc_terraform_type \"\" is not a valid userclouds resource type suffix"))
}

func TestRejectsManifestMissingManifestID(t *testing.T) {
	jsonManifest := `{
		"resources": [
			{
				"uc_terraform_type": "userstore_column",
				"resource_uuids": {
					"staging": "fe20fd48-a006-4ad8-9208-4aad540d8794"
				},
				"attributes": {}
			}
		]
	}`
	_, err := parseAndValidateJSON(jsonManifest, "mycompany-staging")
	assert.True(t, err != nil && strings.Contains(err.Error(), "manifest_id is required"))
}

func TestRejectsManifestMissingResourceUUID(t *testing.T) {
	jsonManifest := `{
		"resources": [
			{
				"uc_terraform_type": "userstore_column",
				"manifest_id": "fe20fd48-a006-4ad8-9208-4aad540d8794",
				"resource_uuids": {
					"staging": "fe20fd48-a006-4ad8-9208-4aad540d8794"
				},
				"attributes": {}
			}
		]
	}`
	_, err := parseAndValidateJSON(jsonManifest, "mycompany-prod")
	assert.True(t, err != nil && strings.Contains(err.Error(), "resource_uuids either must include a UUID for tenant \"mycompany-prod\", or it must include a __DEFAULT entry."))
}
