package tfconfig

import (
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/infra/assert"
)

func TestToHclTokensNil(t *testing.T) {
	tokens, err := toHclTokens(nil, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "null")
}

func TestToHclTokensBoolLiteral(t *testing.T) {
	tokens, err := toHclTokens(true, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "true")
}

func TestToHclTokensIntLiteral(t *testing.T) {
	tokens, err := toHclTokens(-42, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "-42")
}

func TestToHclTokensUintLiteral(t *testing.T) {
	tokens, err := toHclTokens(uint64(42), nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "42")
}

func TestToHclTokensFloatLiteral(t *testing.T) {
	tokens, err := toHclTokens(1.23, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "1.23")
}

func TestToHclTokensStringLiteral(t *testing.T) {
	tokens, err := toHclTokens("function hi() { return `${\"hello\"}`; }", nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "\"function hi() { return `$${\\\"hello\\\"}`; }\"")
}

func TestToHclTokensObject(t *testing.T) {
	tokens, err := toHclTokens(map[string]any{"a": "hello", "b": 2, "c": false}, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `{
  "a" = "hello"
  "b" = 2
  "c" = false
}`)
}

func TestToHclTokensIntArray(t *testing.T) {
	tokens, err := toHclTokens([]int{1, 2, 3}, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), "[1, 2, 3]")
}

func TestToHclTokensObjectArray(t *testing.T) {
	tokens, err := toHclTokens([]map[string]any{{"a": "hi"}}, nil)
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `[{
  "a" = "hi"
}]`)
}

func TestToHclTokensUCManifestID(t *testing.T) {
	tokens, err := toHclTokens(`@UC_MANIFEST_ID("sample").id`, &GenerationContext{
		Manifest: &manifest.Manifest{
			Resources: []manifest.Resource{{
				TerraformTypeSuffix: "resourcetype",
				ManifestID:          "sample",
			}},
		},
	})
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `userclouds_resourcetype.manifestid-sample.id`)
}

func TestToHclTokensUCSystemObject(t *testing.T) {
	tokens, err := toHclTokens(`@UC_SYSTEM_OBJECT("userstore_column", "syscol")`, &GenerationContext{
		LiveResources: &[]liveresource.Resource{{
			TerraformTypeSuffix: "userstore_column",
			ResourceUUID:        "78733010-2a5b-469e-924e-50258db84db9",
			IsSystem:            true,
			Attributes: map[string]any{
				"name": "syscol",
			},
		}},
	})
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `"78733010-2a5b-469e-924e-50258db84db9"`)
}

func TestToHclTokensFile(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "TestToHclTokensFile")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if os.RemoveAll(tmpdir) != nil {
			t.Fatal(err)
		}
	}()
	err = os.WriteFile(tmpdir+"/hello.txt", []byte("Hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test relative path
	tokens, err := toHclTokens(`@FILE("./hello.txt")`, &GenerationContext{ManifestFilePath: tmpdir + "/manifest.yaml"})
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `"Hello world"`)

	// Test absolute path
	tokens, err = toHclTokens(`@FILE("`+tmpdir+`/hello.txt")`, &GenerationContext{ManifestFilePath: tmpdir + "/manifest.yaml"})
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `"Hello world"`)

	// Test that a trailing newline gets stripped
	err = os.WriteFile(tmpdir+"/hello.txt", []byte("Hello world\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err = toHclTokens(`@FILE("./hello.txt")`, &GenerationContext{ManifestFilePath: tmpdir + "/manifest.yaml"})
	assert.NoErr(t, err)
	assert.Equal(t, string(hclwrite.Format(tokens.Bytes())), `"Hello world"`)
}
