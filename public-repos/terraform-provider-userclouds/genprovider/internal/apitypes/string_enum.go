package apitypes

import (
	"fmt"
	"strings"

	"github.com/swaggest/openapi-go/openapi3"
	"golang.org/x/exp/maps"
)

// StringEnum represents an enum of strings.
type StringEnum struct {
	Schema *openapi3.Schema
	Values []string
}

// TFModelType returns the type that should be used to represent this type in a Terraform model.
func (t *StringEnum) TFModelType() string {
	return "types.String"
}

// TFSchemaAttributeType returns an attr.Type, which is a struct representing the type of this
// object in the Terraform schema.
func (t *StringEnum) TFSchemaAttributeType() string {
	return "types.StringType"
}

func (t *StringEnum) description() string {
	desc := "Valid values: "
	for i, v := range t.Values {
		if len(v) == 0 {
			// Skip empty values. Otherwise, we generate `` in the documentation
			// markdown, which seems like it should be valid ("empty code
			// block"), but it messes up the parsing. I'm not entirely sure how
			// we should generate an empty code block, but let's just omit for
			// now.
			continue
		}
		desc += "`" + v + "`"
		if i < len(t.Values)-1 {
			desc += ", "
		}
	}
	if t.Schema != nil && t.Schema.Description != nil && *t.Schema.Description != "" {
		if !strings.HasSuffix(*t.Schema.Description, ".") {
			desc = ". " + desc
		}
		desc = *t.Schema.Description + desc
	}
	return desc
}

// TFSchemaAttributeText returns the text of the code for instantiating this type as a Terraform
// schema attribute.
func (t *StringEnum) TFSchemaAttributeText(extraFields map[string]string) string {
	fields := makeCommonFields(t.Schema)
	fields["Description"] = fmt.Sprintf("%#v", t.description())
	fields["MarkdownDescription"] = fmt.Sprintf("%#v", t.description())
	maps.Copy(fields, extraFields)
	return `schema.StringAttribute{
		Validators: []validator.String{
			stringvalidator.OneOf(` + fmt.Sprintf("%#v", t.Values) + `...),
		},
		` + fieldsToString(fields) + `
	}`
}

// JSONClientModelType returns the type that should be used to represent this type in a jsonclient
// request/response struct.
func (t *StringEnum) JSONClientModelType() string {
	return "string"
}

// TFModelToJSONClientFunc returns the text of a function for converting a Terraform model struct to
// a jsonclient model struct.
func (t *StringEnum) TFModelToJSONClientFunc() string {
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
func (t *StringEnum) JSONClientModelToTFFunc() string {
	return `func (val *string) (types.String, error) {
		return types.StringPointerValue(val), nil
	}`
}

// GetTFPlanModifierType returns the name of the
// terraform-plugin-framework/resource/schema/planmodifier type for this API
// type (e.g. String, Int64, etc.)
func (t *StringEnum) GetTFPlanModifierType() string {
	return "String"
}

// GetTFPlanModifierPackageName returns the name of the package
// (terraform-plugin-framework/resource/schema/RETURNVALUE) containing the plan
// modifiers for this type
func (t *StringEnum) GetTFPlanModifierPackageName() string {
	return "stringplanmodifier"
}
