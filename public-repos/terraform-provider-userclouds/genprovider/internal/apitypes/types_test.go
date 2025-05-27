package apitypes

import (
	"testing"
)

func makePlanModifiers(t APIType) string {
	return `[]planmodifier.` + t.GetTFPlanModifierType() + `{
		` + t.GetTFPlanModifierPackageName() + `.RequiresReplace(),
	}`
}

func TestString(t *testing.T) {
	apitype := &String{}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `"hello"`,
		SampleTFModelValue:    `types.StringValue("hello")`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestInt(t *testing.T) {
	apitype := &Int{}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `int64(123)`,
		SampleTFModelValue:    `types.Int64Value(123)`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestFloat(t *testing.T) {
	apitype := &Float{}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `float64(1.23)`,
		SampleTFModelValue:    `types.Float64Value(1.23)`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestBool(t *testing.T) {
	apitype := &Bool{}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `true`,
		SampleTFModelValue:    `types.BoolValue(true)`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

var objectSampleCode = `
		type ObjTFModel struct {
			Name types.String ` + "`tfsdk:\"name\"`" + `
		}

		type ObjJSONClientModel struct {
			Name *string ` + "`json:\"name\"`" + `
		}

		var ObjAttributes = map[string]schema.Attribute {
			"name": schema.StringAttribute{},
		}

		var ObjSchemaAttrTypes = map[string]attr.Type {
			"name": types.StringType,
		}

		func ObjTFModelToJSONClient(val *ObjTFModel) (*ObjJSONClientModel, error) {
			return &ObjJSONClientModel{Name: val.Name.ValueStringPointer()}, nil
		}

		func ObjJSONClientModelToTF(val *ObjJSONClientModel) (ObjTFModel, error) {
			return ObjTFModel{Name: types.StringPointerValue(val.Name)}, nil
		}

		// define sample string so that we can get a pointer to it
		var sampleString = "hello"
		var sampleJSONClientValue = ObjJSONClientModel{Name: &sampleString}
		var sampleTFModelAttrTypes = map[string]attr.Type{"name": basetypes.StringType{}}
		var sampleTFModelValue = types.ObjectValueMust(sampleTFModelAttrTypes, map[string]attr.Value{"name": types.StringValue(sampleString)})
`

func TestObject(t *testing.T) {
	apitype := &Object{
		TFModelStructName:           "ObjTFModel",
		TFSchemaAttributesMapName:   "ObjAttributes",
		TFSchemaAttrTypeMapName:     "ObjSchemaAttrTypes",
		JSONClientModelStructName:   "ObjJSONClientModel",
		TFModelToJSONClientFuncName: "ObjTFModelToJSONClient",
		JSONClientModelToTFFuncName: "ObjJSONClientModelToTF",
	}
	runTestProgram(t, data{
		ExtraCode:             objectSampleCode,
		T:                     apitype,
		SampleJSONClientValue: "sampleJSONClientValue",
		SampleTFModelValue:    "sampleTFModelValue",
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestStringArray(t *testing.T) {
	apitype := &Array{ChildType: &String{}}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `[]string{"hello", "world"}`,
		SampleTFModelValue:    `types.ListValueMust(types.StringType, []attr.Value{types.StringValue("hello"), types.StringValue("world")})`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestObjectArray(t *testing.T) {
	runTestProgram(t, data{
		ExtraCode: objectSampleCode,
		T: &Array{
			ChildType: &Object{
				TFModelStructName:           "ObjTFModel",
				TFSchemaAttributesMapName:   "ObjAttributes",
				TFSchemaAttrTypeMapName:     "ObjSchemaAttrTypes",
				JSONClientModelStructName:   "ObjJSONClientModel",
				TFModelToJSONClientFuncName: "ObjTFModelToJSONClient",
				JSONClientModelToTFFuncName: "ObjJSONClientModelToTF",
			},
		},
		SampleJSONClientValue: "[]ObjJSONClientModel{sampleJSONClientValue}",
		SampleTFModelValue:    `types.ListValueMust(sampleTFModelValue.Type(context.Background()), []attr.Value{sampleTFModelValue})`,
	})
}

func TestStringMap(t *testing.T) {
	apitype := &Map{ValueType: &String{}}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `map[string]string{"a": "A", "b": "B"}`,
		SampleTFModelValue:    `types.MapValueMust(types.StringType, map[string]attr.Value{"a": types.StringValue("A"), "b": types.StringValue("B")})`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestObjectMap(t *testing.T) {
	runTestProgram(t, data{
		ExtraCode: objectSampleCode,
		T: &Map{
			ValueType: &Object{
				TFModelStructName:           "ObjTFModel",
				TFSchemaAttributesMapName:   "ObjAttributes",
				TFSchemaAttrTypeMapName:     "ObjSchemaAttrTypes",
				JSONClientModelStructName:   "ObjJSONClientModel",
				TFModelToJSONClientFuncName: "ObjTFModelToJSONClient",
				JSONClientModelToTFFuncName: "ObjJSONClientModelToTF",
			},
		},
		SampleJSONClientValue: `map[string]ObjJSONClientModel{"a": sampleJSONClientValue}`,
		SampleTFModelValue:    `types.MapValueMust(sampleTFModelValue.Type(context.Background()), map[string]attr.Value{"a": sampleTFModelValue})`,
	})
}

func TestUUID(t *testing.T) {
	apitype := &UUID{}
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000"))`,
		SampleTFModelValue:    `types.StringValue("123e4567-e89b-12d3-a456-426655440000")`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

func TestStringEnum(t *testing.T) {
	apitype := &StringEnum{Values: []string{"a", "b", "c"}}
	// NOTE: this just tests that the generated StringEnum code compiles. It
	// doesn't test the enum validation logic, since that's specified as a
	// Terraform attribute validator, and that's harder to test.
	runTestProgram(t, data{
		T:                     apitype,
		SampleJSONClientValue: `"hello"`,
		SampleTFModelValue:    `types.StringValue("hello")`,
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}

var userstoreResourceIDSampleCode = `
		type UserstoreResourceIDJSONClientModel struct {
			ID   *uuid.UUID
			Name *string
		}

		// define sample string so that we can get a pointer to it
		var sampleID = uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000"))
		var sampleJSONClientValue = UserstoreResourceIDJSONClientModel{ID: &sampleID}
		var sampleTFModelValue = types.StringValue(sampleID.String())
`

func TestUserstoreResourceID(t *testing.T) {
	apitype := &UserstoreResourceID{
		JSONClientModelStructName: "UserstoreResourceIDJSONClientModel",
	}
	runTestProgram(t, data{
		ExtraCode:             userstoreResourceIDSampleCode,
		T:                     apitype,
		SampleJSONClientValue: "sampleJSONClientValue",
		SampleTFModelValue:    "sampleTFModelValue",
		ExtraTFAttributeFields: map[string]string{
			"Required":      "true",
			"PlanModifiers": makePlanModifiers(apitype),
		},
	})
}
