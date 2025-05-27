package apitypes

import (
	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

const uuidRegex = `(?i)^[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}$`

// UUID represents a UUID string.
type UUID struct {
	Schema *openapi3.Schema
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *UUID) TFModelType() string {
	return "types.String"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *UUID) TFSchemaAttributeType() string {
	return "types.StringType"
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *UUID) TFSchemaAttributeText(extraFields map[string]string) string {
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
func (t *UUID) JSONClientModelType() string {
	return "uuid.UUID"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *UUID) TFModelToJSONClientFunc() string {
	return `func (val *types.String) (*uuid.UUID, error) {
		if val.IsNull() || val.IsUnknown() {
			return nil, nil
		}
		converted, err := uuid.FromString(val.ValueString())
		if err != nil {
			return nil, ucerr.Errorf("failed to parse uuid: %v", err)
		}
		return &converted, nil
	}`
}

// JSONClientModelToTFFunc returns the text of a function for converting a jsonclient model struct
// to a Terraform model struct.
func (t *UUID) JSONClientModelToTFFunc() string {
	return `func (val *uuid.UUID) (types.String, error) {
		if val == nil {
			return types.StringNull(), nil
		}
		return types.StringValue(val.String()), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *UUID) GetTFPlanModifierType() string {
	return "String"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *UUID) GetTFPlanModifierPackageName() string {
	return "stringplanmodifier"
}
