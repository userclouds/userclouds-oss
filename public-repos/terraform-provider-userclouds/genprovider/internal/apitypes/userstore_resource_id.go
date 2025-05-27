package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// UserstoreResourceID represents a "userstore.ResourceID" struct. This struct has an ID field and a
// Name field, where only one field is required, so that customers making API requests can use
// either the UUID or the Name to refer to a resource. However, in Terraform, we want to drop the
// Name field, since the Name field is not guaranteed to be stable. If we include Name in Terraform
// state, then e.g. if someone were to change a column name, they would get diffs everywhere that
// references that column, even though those other resources did not change.
type UserstoreResourceID struct {
	Schema *openapi3.Schema
	// JSONClientModelStructName is the name of the codegen'ed struct that stores a
	// userstore.ResourceID value for use in sending/receiving API requests/responses
	JSONClientModelStructName string
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *UserstoreResourceID) TFModelType() string {
	// Store only the UUID in Terraform state
	return "types.String"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *UserstoreResourceID) TFSchemaAttributeType() string {
	return "types.StringType"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *UserstoreResourceID) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	maps.Copy(fields, extraFields)
	return `schema.StringAttribute{
		Validators: []validator.String{
			stringvalidator.RegexMatches(
				regexp.MustCompile("` + uuidRegex + `"),
				"invalid UUID format",
			),
		},
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *UserstoreResourceID) JSONClientModelType() string {
	return t.JSONClientModelStructName
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *UserstoreResourceID) TFModelToJSONClientFunc() string {
	return `func (val *types.String) (*` + t.JSONClientModelType() + `, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted, err := uuid.FromString(val.ValueString())
		if err != nil {
			return nil, ucerr.Errorf("failed to parse uuid: %v", err)
		}
		s := ` + t.JSONClientModelStructName + `{
			ID: &converted,
		}
		return &s, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *UserstoreResourceID) JSONClientModelToTFFunc() string {
	return `func (val *` + t.JSONClientModelType() + `) (types.String, error) {
		if val == nil {
			return types.StringNull(), nil
		}
		// We should only need to convert jsonclient models to TF models when receiving API
		// responses, and API responses should always have the ID set.
		// Sometimes we receive nil UUIDs here because of how the server
		// serializes empty values, so we should only freak out if we see a
		// name provided but not an ID.
		if val.Name != nil && *val.Name != "" && (val.ID == nil || val.ID.IsNil()) {
			return types.StringNull(), ucerr.Errorf("got nil ID field in UserstoreResourceID. this is an issue with the UserClouds Terraform provider")
		}
		if val.ID == nil || val.ID.IsNil() {
			return types.StringNull(), nil
		}
		return types.StringValue(val.ID.String()), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *UserstoreResourceID) GetTFPlanModifierType() string {
	return "String"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *UserstoreResourceID) GetTFPlanModifierPackageName() string {
	return "stringplanmodifier"
}
