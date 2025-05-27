package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// Bool represents a boolean value.
type Bool struct {
	Schema *openapi3.Schema
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *Bool) TFModelType() string {
	return "types.Bool"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *Bool) TFSchemaAttributeType() string {
	return "types.BoolType"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *Bool) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	maps.Copy(fields, extraFields)
	return `schema.BoolAttribute{
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *Bool) JSONClientModelType() string {
	return "bool"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *Bool) TFModelToJSONClientFunc() string {
	return `func (val *types.Bool) (*bool, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted := val.ValueBool()
		return &converted, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *Bool) JSONClientModelToTFFunc() string {
	return `func (val *bool) (types.Bool, error) {
		return types.BoolPointerValue(val), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *Bool) GetTFPlanModifierType() string {
	return "Bool"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *Bool) GetTFPlanModifierPackageName() string {
	return "boolplanmodifier"
}
