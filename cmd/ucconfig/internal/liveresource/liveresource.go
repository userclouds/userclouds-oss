package liveresource

import (
	"context"
	"reflect"
	"strings"

	"github.com/gofrs/uuid"
	"golang.org/x/exp/slices"

	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// Resource stores a live resource, which we can then import into the Terraform state or use
// to generate a new ucconfig manifest.
type Resource struct {
	TerraformTypeSuffix string
	ManifestID          string
	ResourceUUID        string
	IsSystem            bool
	Attributes          map[string]any
}

// TerraformResourceName returns the name that should be used for this resource
// in the Terraform configuration and state
func (r *Resource) TerraformResourceName() string {
	if r.ManifestID != "" {
		return "manifestid-" + r.ManifestID
	}
	return "unmatched-" + r.ResourceUUID
}

// transformValue takes some value from a UC API model and applies any transformations needed to store
// that value in the manifest or in TF state.
func transformValue(data any, omitAttributes ...string) (any, error) {
	v := reflect.ValueOf(data)

	// If this is a userstore.ResourceID, grab just the UUID and omit the name. In the
	// terraform provider, we use apitypes.UserstoreResourceID to transform the string back into a
	// userstore.ResourceID struct.
	if v.Type() == reflect.TypeOf(userstore.ResourceID{}) {
		id := v.Interface().(userstore.ResourceID).ID
		name := v.Interface().(userstore.ResourceID).Name
		if name != "" && id.IsNil() {
			return nil, ucerr.Errorf("userstore.ResourceID has a nil ID")
		}
		return id.String(), nil
	}

	// Serialize IDs as strings
	if v.Type() == reflect.TypeOf(uuid.UUID{}) {
		return data.(uuid.UUID).String(), nil
	}

	// For collections, we may need to apply transformations to collection members, so we need to
	// iterate through and rebuild these collections.
	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		var items []any
		for i := 0; i < v.Len(); i++ {
			transformed, err := transformValue(v.Index(i).Interface(), omitAttributes...)
			if err != nil {
				return nil, ucerr.Errorf("error transforming value at array index %v: %v", i, err)
			}
			items = append(items, transformed)
		}
		return items, nil
	}
	if v.Kind() == reflect.Map {
		items := map[string]any{}
		for _, key := range v.MapKeys() {
			transformed, err := transformValue(v.MapIndex(key).Interface(), omitAttributes...)
			if err != nil {
				return nil, ucerr.Errorf("error transforming value at map key %s: %v", key, err)
			}
			items[key.String()] = transformed
		}
		return items, nil
	}
	// Similarly, for objects/structs, we need to iterate over the fields and apply transformations
	// to each value
	if v.Kind() == reflect.Struct {
		items := map[string]any{}
		for i := 0; i < v.NumField(); i++ {
			key := strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0]
			transformed, err := transformValue(v.Field(i).Interface(), omitAttributes...)
			if err != nil {
				return nil, ucerr.Errorf("error transforming struct field %s: %v", v.Type().Field(i).Name, err)
			}

			// Skip fields that should be omitted
			if slices.Contains(omitAttributes, key) {
				continue
			}

			// Skip optional fields that are empty
			required := v.Type().Field(i).Tag.Get("required") == "true"
			if !required && v.Field(i).IsZero() {
				continue
			}

			items[key] = transformed
		}
		return items, nil
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		transformed, err := transformValue(v.Elem().Interface(), omitAttributes...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return &transformed, nil
	}

	// General case: no transformation needed
	return data, nil
}

// MakeLiveResource takes a live resource object fetched from the UC API, extracts the ID and
// attributes, and constructs a LiveResource struct for that resource. ManifestID will be left
// blank, since we don't perform any matching against a manifest at this stage.
func MakeLiveResource(ctx context.Context, resourceType resourcetypes.ResourceType, resource any) (Resource, error) {
	// Gather the attributes of the resource for the Attributes map
	attributes := map[string]any{}
	v := reflect.ValueOf(resource)
	resourceID := v.FieldByName("ID").Interface().(uuid.UUID)
	isSystem := false
	for i := 0; i < v.NumField(); i++ {
		jsonKey := strings.Split(v.Type().Field(i).Tag.Get("json"), ",")[0]
		// We have a small handful of model that are used both for the database
		// and for the API. May need to extract IsSystem from those models.
		if v.Type().Field(i).Type == reflect.TypeOf(ucdb.SystemAttributeBaseModel{}) {
			isSystem = v.Field(i).Interface().(ucdb.SystemAttributeBaseModel).IsSystem
			continue
		}
		// Skip internal fields not used in the API
		if jsonKey == "" {
			continue
		}
		// Skip the resource UUID, which is stored in the LiveResource as a separate field
		if jsonKey == "id" {
			continue
		}
		// Omit the is_system field from the attributes. It can't be changed, and we only show
		// resources where is_system=false in the manifest, so leaving it in is just adding clutter.
		if jsonKey == "is_system" {
			isSystem = v.Field(i).Bool()
			continue
		}
		// Omit the version field from the attributes. It functions as an etag
		// for versioned resources, and the Terraform provider won't allow it to
		// be set
		if jsonKey == "version" {
			continue
		}
		if slices.Contains(resourceType.OmitAttributes, jsonKey) {
			continue
		}
		// Skip optional fields that are empty
		required := v.Type().Field(i).Tag.Get("required") == "true"
		if !required && v.Field(i).IsZero() {
			continue
		}
		// Native data types should be treated as system resources
		if jsonKey == "is_native" {
			isSystem = v.Field(i).Bool()
		}

		transformed, err := transformValue(v.Field(i).Interface(), resourceType.OmitAttributes...)
		if err != nil {
			return Resource{}, ucerr.Errorf("error marshalling %s field %s: %v", resourceType.TerraformTypeSuffix, jsonKey, err)
		}
		attributes[jsonKey] = transformed
	}

	return Resource{
		// Note: leaving ManifestID blank, since we don't do any matching against a manifest at this
		// level of abstraction. The caller can set the ManifestID if needed
		TerraformTypeSuffix: resourceType.TerraformTypeSuffix,
		ResourceUUID:        resourceID.String(),
		IsSystem:            isSystem,
		Attributes:          attributes,
	}, nil
}

func validateResourceType(resourceType resourcetypes.ResourceType, liveResources *[]any) error {
	// Validate that the resource model type has an ID field. Currently ucconfig doesn't support
	// resources without an ID (or with an ID that goes by a different name in the struct).
	v := reflect.ValueOf((*liveResources)[0])
	idField, found := v.Type().FieldByName("ID")
	if !found {
		return ucerr.Errorf("resource type %s does not have an ID field. this is an error with the ucconfig code.", resourceType.TerraformTypeSuffix)
	}
	if idField.Type != reflect.TypeOf(uuid.UUID{}) {
		return ucerr.Errorf("resource type %s has an ID field, but it is not of type uuid.UUID. this is an error with the ucconfig code.", resourceType.TerraformTypeSuffix)
	}
	return nil
}

// GetLiveResourcesForType fetches all resources of a given type from the UC API and returns a list
// of LiveResource structs for those resources.
func GetLiveResourcesForType(ctx context.Context, client *idp.Client, resourceType resourcetypes.ResourceType) ([]Resource, error) {
	liveResources, err := resourceType.ListResources(ctx, client)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(liveResources) == 0 {
		return nil, nil
	}

	if err = validateResourceType(resourceType, &liveResources); err != nil {
		return nil, ucerr.Wrap(err)
	}

	var out []Resource
	for _, resource := range liveResources {
		s, err := MakeLiveResource(ctx, resourceType, resource)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		out = append(out, s)
	}

	return out, nil
}

// GetLiveResources fetches all live resources from the UC API and returns a list of LiveResource
// structs.
func GetLiveResources(ctx context.Context, client *idp.Client) ([]Resource, error) {
	var out []Resource
	for _, resourceType := range resourcetypes.ResourceTypes {
		liveResources, err := GetLiveResourcesForType(ctx, client, resourceType)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		out = append(out, liveResources...)
	}
	return out, nil
}
