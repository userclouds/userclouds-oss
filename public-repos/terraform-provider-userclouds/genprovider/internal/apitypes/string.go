package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// String represents a string value.
type String struct {
	Schema *openapi3.Schema
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *String) TFModelType() string {
	return "types.String"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *String) TFSchemaAttributeType() string {
	return "types.StringType"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *String) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	maps.Copy(fields, extraFields)
	return `schema.StringAttribute{
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *String) JSONClientModelType() string {
	return "string"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *String) TFModelToJSONClientFunc() string {
	return `func (val *types.String) (*string, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted := val.ValueString()
		return &converted, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *String) JSONClientModelToTFFunc() string {
	return `func (val *string) (types.String, error) {
		return types.StringPointerValue(val), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *String) GetTFPlanModifierType() string {
	return "String"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *String) GetTFPlanModifierPackageName() string {
	return "stringplanmodifier"
}
