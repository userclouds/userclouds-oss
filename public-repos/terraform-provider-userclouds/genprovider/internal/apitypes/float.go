package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// Float represents a float value.
type Float struct {
	Schema *openapi3.Schema
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *Float) TFModelType() string {
	return "types.Float64"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *Float) TFSchemaAttributeType() string {
	return "types.Float64Type"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *Float) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	maps.Copy(fields, extraFields)
	return `schema.Float64Attribute{
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *Float) JSONClientModelType() string {
	return "float64"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *Float) TFModelToJSONClientFunc() string {
	return `func (val *types.Float64) (*float64, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted := val.ValueFloat64()
		return &converted, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *Float) JSONClientModelToTFFunc() string {
	return `func (val *float64) (types.Float64, error) {
		return types.Float64PointerValue(val), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *Float) GetTFPlanModifierType() string {
	return "Float64"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *Float) GetTFPlanModifierPackageName() string {
	return "float64planmodifier"
}
