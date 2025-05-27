package manifest

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Manifest is the top-level object storing a parsed ucconfig manifest file
type Manifest struct {
	Resources []Resource `json:"resources" yaml:"resources"`
}

// Resource stores the config for a single instantiation of a Terraform resource
type Resource struct {
	// Terraform resource type suffix, e.g. "userstore_column"
	TerraformTypeSuffix string `json:"uc_terraform_type" yaml:"uc_terraform_type"`
	// A unique ID for this resource that will be stable across tenants and time. This is used
	// to produce a Terraform resource path. This can be an arbitrary ID, does not need to be a
	// UUID.
	ManifestID string `json:"manifest_id" yaml:"manifest_id"`
	// A map of fully-qualified-tenant-name to resource UUID within that tenant. The key
	// "__DEFAULT" can be used to set a default UUID when creating this resource in a new
	// tenant.
	ResourceUUIDs map[string]string `json:"resource_uuids" yaml:"resource_uuids"`
	// A map of attributes to set on the Terraform resource.
	Attributes map[string]any `json:"attributes" yaml:"attributes"`
}

func fromLiveResource(live *liveresource.Resource, fqtn string) Resource {
	// ManifestID should default to "{TerraformTypeSuffix}_{ResourceName}" if the resource has a
	// Name field. Otherwise, fall back to using the resource UUID.
	manifestID := live.ManifestID
	if manifestID == "" {
		if name, ok := live.Attributes["name"]; ok {
			manifestID = fmt.Sprintf("%s_%s", live.TerraformTypeSuffix, name)
		} else {
			manifestID = live.ResourceUUID
		}
	}
	return Resource{
		TerraformTypeSuffix: live.TerraformTypeSuffix,
		ManifestID:          manifestID,
		ResourceUUIDs: map[string]string{
			"__DEFAULT": live.ResourceUUID,
			fqtn:        live.ResourceUUID,
		},
		Attributes: live.Attributes,
	}
}

func (r *Resource) getResourceType() resourcetypes.ResourceType {
	return *resourcetypes.GetByTerraformTypeSuffix(r.TerraformTypeSuffix)
}

// ExternValuesDirConfig stores paths for storing attribute values in an
// external directory (e.g. JS function definition strings).
type ExternValuesDirConfig struct {
	// AbsolutePath stores the path to the directory so that we can write files there
	AbsolutePath string
	// RelativePathFromManifest stores the relative path from where the manifest
	// will be written. This affects the "@FILE(path)" function calls that will
	// be written to the manifest, so that anyone applying the manifest can get
	// these extern values relative to the manifest.
	RelativePathFromManifest string
}

type functionGenerationContext struct {
	Manifest        *Manifest
	FQTN            string
	LiveResources   *[]liveresource.Resource
	ExternValuesDir *ExternValuesDirConfig
}

// rewriteManifestAttribute takes a resource attribute value and returns a
// rewritten value if it should use a function call instead (e.g. if the value
// is a reference to another resource, which should be a UC_MANIFEST_ID function
// call instead). Restated, if the supplied value has no references to another
// resource or other special desired behavior, the value will be returned
// unchanged, but otherwise this function will return the text of a
// `@FUNCTIONNAME()` function call to implement the desired behavior.
func rewriteManifestAttribute(data any, currAttrPath string, forResource *Resource, ctx *functionGenerationContext) (any, error) {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Pointer {
		return rewriteManifestAttribute(v.Elem().Interface(), currAttrPath, forResource, ctx)
	}

	if v.Kind() == reflect.String && forResource.getResourceType().References[currAttrPath] != "" {
		// This should be a reference to another resource. Rewrite the value to reference the other
		// resource. Note: we are using mfest.Resources instead of liveResources here, because we
		// want to get the manifest IDs, which are guaranteed to be set properly in the manifest
		// struct
		ref := v.String()
		for _, r := range ctx.Manifest.Resources {
			if r.TerraformTypeSuffix == forResource.getResourceType().References[currAttrPath] && r.ResourceUUIDs[ctx.FQTN] == ref {
				return `@UC_MANIFEST_ID("` + r.ManifestID + `").id`, nil
			}
		}
		// If we didn't find a match to a resource in the manifest, try seeing if we can match a
		// system object
		for _, r := range *ctx.LiveResources {
			if r.TerraformTypeSuffix == forResource.getResourceType().References[currAttrPath] && r.IsSystem && r.ResourceUUID == ref {
				return `@UC_SYSTEM_OBJECT("` + r.TerraformTypeSuffix + `", "` + r.Attributes["name"].(string) + `")`, nil
			}
		}
		return nil, ucerr.Errorf("this should be a reference to a %s resource, but the live resource state we fetched doesn't contain such a resource with UUID %s", forResource.getResourceType().References[currAttrPath], ref)
	}

	// Rewrite external attributes
	if extension, ok := forResource.getResourceType().WriteAttributesExternally[currAttrPath]; ok && ctx.ExternValuesDir != nil {
		if v.Kind() != reflect.String {
			return nil, ucerr.Errorf("attribute %s is marked as an external attribute, but the value is not a string", currAttrPath)
		}
		val := v.String()

		nameOrID := forResource.ManifestID
		if name, ok := forResource.Attributes["name"]; ok && reflect.TypeOf(name).Kind() == reflect.String {
			nameOrID = name.(string)
		}
		targetFileName := forResource.TerraformTypeSuffix + "_" + nameOrID + "_" + strings.ReplaceAll(currAttrPath, ".", "_") + extension
		targetPath := ctx.ExternValuesDir.AbsolutePath + "/" + targetFileName
		// Write file with trailing newline (since most text editors will insert
		// one upon save)
		if err := os.WriteFile(targetPath, []byte(val+"\n"), 0644); err != nil {
			return nil, ucerr.Errorf("error writing external attribute value to %s for attribute %s: %v", targetPath, currAttrPath, err)
		}
		return `@FILE("` + ctx.ExternValuesDir.RelativePathFromManifest + "/" + targetFileName + `")`, nil
	}

	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		var out []any
		for i := 0; i < v.Len(); i++ {
			val := v.Index(i).Interface()
			rewritten, err := rewriteManifestAttribute(val, currAttrPath, forResource, ctx)
			if err != nil {
				return nil, ucerr.Errorf("error rewriting value at array index %v: %v", i, err)
			}
			out = append(out, rewritten)
		}
		return out, nil
	}

	if v.Kind() == reflect.Map {
		out := map[string]any{}
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key).Interface()
			rewritten, err := rewriteManifestAttribute(val, currAttrPath+"."+key.String(), forResource, ctx)
			if err != nil {
				return nil, ucerr.Errorf("error rewriting value at map key %s: %v", key, err)
			}
			out[key.String()] = rewritten
		}
		return out, nil
	}

	// Many types won't need rewriting
	return data, nil
}

// MatchLiveResources compares live resources to resources declared in the manifest, setting the
// correct ManifestID on matched live resources. For resources that could not be matched to the
// manifest, the ManifestID is left blank. If a manifest entry ends up matching a resource by name
// (but not by UUID), the manifest entry ResourceUUIDs will also be updated to include the resource
// ID.
func (mfest *Manifest) MatchLiveResources(ctx context.Context, liveResources *[]liveresource.Resource, fqtn string) error {
	unmatchedLiveResourceIndexes := map[string]int{}
	for i, resource := range *liveResources {
		// System objects should not be matched to the manifest
		if resource.IsSystem {
			continue
		}
		unmatchedLiveResourceIndexes[resource.ResourceUUID] = i
	}

	unmatchedManifests := map[string]*Resource{}
	for i, manifest := range mfest.Resources {
		unmatchedManifests[manifest.ManifestID] = &mfest.Resources[i]
	}

	// First pass: match live resources to manifest IDs using resource UUIDs
	for manifestID, manifest := range unmatchedManifests {
		targetResourceID := manifest.ResourceUUIDs[fqtn]
		if targetResourceID == "" {
			targetResourceID = manifest.ResourceUUIDs["__DEFAULT"]
		}
		// If the manifest's resource UUID matches an unmatched resource with the same resource
		// type...
		if resourceIndex, ok := unmatchedLiveResourceIndexes[targetResourceID]; ok && manifest.TerraformTypeSuffix == (*liveResources)[resourceIndex].TerraformTypeSuffix {
			(*liveResources)[resourceIndex].ManifestID = manifestID
			delete(unmatchedLiveResourceIndexes, targetResourceID)
			delete(unmatchedManifests, manifestID)
			// If we used the __DEFAULT UUID, update the manifest to include the
			// tenant-specific UUID we found
			if manifest.ResourceUUIDs[fqtn] == "" {
				manifest.ResourceUUIDs[fqtn] = targetResourceID
			}
		}
	}

	// Second pass: match live resources to manifest IDs using resource names where possible
	for manifestID, manifest := range unmatchedManifests {
		// Skip entries where the manifest explicitly specified a UUID for this tenant. If the
		// UUID was explicit but didn't match a live resource in the previous step, we shouldn't try
		// to fill it in with something else
		if manifest.ResourceUUIDs[fqtn] != "" {
			continue
		}
		// Skip entries where no name is present; matching by name is impossible.
		if manifest.Attributes["name"] == nil {
			continue
		}
		manifestName := manifest.Attributes["name"].(string)
		for _, resourceIndex := range unmatchedLiveResourceIndexes {
			if (*liveResources)[resourceIndex].Attributes["name"] == nil {
				continue
			}
			resourceID := (*liveResources)[resourceIndex].ResourceUUID
			resourceName := (*liveResources)[resourceIndex].Attributes["name"].(string)
			if manifestName == resourceName && manifest.TerraformTypeSuffix == (*liveResources)[resourceIndex].TerraformTypeSuffix {
				uclog.Warningf(ctx, "Live resource %s (id %s) does not match a resource ID in the manifest, but the name matches the resource manifest with manifest ID %s. Assuming that these are intended to be the same resource...", resourceName, resourceID, manifestID)
				(*liveResources)[resourceIndex].ManifestID = manifestID
				manifest.ResourceUUIDs[fqtn] = resourceID
				delete(unmatchedManifests, manifestID)
				delete(unmatchedLiveResourceIndexes, resourceID)
				break
			}
		}
	}

	// Warn for unmatched resources
	for resourceID, resourceIndex := range unmatchedLiveResourceIndexes {
		var description string
		if name := (*liveResources)[resourceIndex].Attributes["name"]; name != nil {
			description = fmt.Sprintf("%s (id %s)", name.(string), resourceID)
		} else {
			description = resourceID
		}
		uclog.Warningf(ctx, "Live %s resource %s could not be matched to any resources in the manifest. This resources will be deleted if the configuration is applied.", (*liveResources)[resourceIndex].TerraformTypeSuffix, description)
	}

	return nil
}

// RewriteWithFunctionCalls updates the attribute values for this resource:
// where attribute values should have special behavior (e.g. a UUID that is a
// reference to a different resource ID), those values will be replaced with
// string `@FUNCTIONNAME()` function invocations.
func (r *Resource) RewriteWithFunctionCalls(ctx *functionGenerationContext) error {
	for key, value := range r.Attributes {
		rewritten, err := rewriteManifestAttribute(value, key, r, ctx)
		if err != nil {
			return ucerr.Errorf("error rewriting manifest reference for %s attribute %s: %v", r.TerraformTypeSuffix, key, err)
		}
		r.Attributes[key] = rewritten
	}
	return nil
}

// Validate returns an error if the manifest is malformed.
func (mfest *Manifest) Validate(fqtn string) error {
	for i, resource := range mfest.Resources {
		if !resourcetypes.ValidateTerraformTypeSuffix(resource.TerraformTypeSuffix) {
			return ucerr.Errorf("error validating resource at index %v: uc_terraform_type \"%s\" is not a valid userclouds resource type suffix", i, resource.TerraformTypeSuffix)
		}
		if resource.ManifestID == "" {
			return ucerr.Errorf("error validating resource at index %v: manifest_id is required", i)
		}
		if resource.ResourceUUIDs[fqtn] == "" && resource.ResourceUUIDs["__DEFAULT"] == "" {
			return ucerr.Errorf("error validating resource at index %v: resource_uuids either must include a UUID for tenant \"%s\", or it must include a __DEFAULT entry.", i, fqtn)
		}
	}
	return nil
}

func generateFromLiveResources(ctx context.Context, liveResources *[]liveresource.Resource, fqtn string, externValuesDir *ExternValuesDirConfig) (Manifest, error) {
	var resourceManifests []Resource
	for _, r := range *liveResources {
		if r.IsSystem {
			// Omit system resources from the manifest, since they can't be changed and just add
			// clutter.
			continue
		}
		resourceManifests = append(resourceManifests, fromLiveResource(&r, fqtn))
	}
	mfest := Manifest{
		Resources: resourceManifests,
	}
	for i := range mfest.Resources {
		if err := mfest.Resources[i].RewriteWithFunctionCalls(&functionGenerationContext{
			Manifest:        &mfest,
			LiveResources:   liveResources,
			FQTN:            fqtn,
			ExternValuesDir: externValuesDir,
		}); err != nil {
			return Manifest{}, ucerr.Wrap(err)
		}
	}
	return mfest, nil
}

// GenerateNewManifest fetches the live resources using the UC API and returns
// a new Manifest struct describing those resources.
//
// The optional externAttrValuesDir specifies a directory where attribute values
// can be stored for attributes that we'd prefer to not specify inline in the
// manifest (e.g. Javascript function definitions).
func GenerateNewManifest(ctx context.Context, client *idp.Client, fqtn string, externValuesDir *ExternValuesDirConfig) (Manifest, error) {
	liveResources, err := liveresource.GetLiveResources(ctx, client)
	if err != nil {
		return Manifest{}, ucerr.Wrap(err)
	}
	return generateFromLiveResources(ctx, &liveResources, fqtn, externValuesDir)
}
