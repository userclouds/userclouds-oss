package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// Int represents an integer value.
type Int struct {
	Schema *openapi3.Schema
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *Int) TFModelType() string {
	return "types.Int64"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *Int) TFSchemaAttributeType() string {
	return "types.Int64Type"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *Int) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	maps.Copy(fields, extraFields)
	return `schema.Int64Attribute{
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *Int) JSONClientModelType() string {
	return "int64"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *Int) TFModelToJSONClientFunc() string {
	return `func (val *types.Int64) (*int64, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted := val.ValueInt64()
		return &converted, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *Int) JSONClientModelToTFFunc() string {
	return `func (val *int64) (types.Int64, error) {
		return types.Int64PointerValue(val), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *Int) GetTFPlanModifierType() string {
	return "Int64"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *Int) GetTFPlanModifierPackageName() string {
	return "int64planmodifier"
}
