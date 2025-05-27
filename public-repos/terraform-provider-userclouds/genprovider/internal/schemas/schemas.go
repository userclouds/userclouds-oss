package schemas

import (
	"bytes"
	"context"
	"go/format"
	"log"
	"os"
	"sort"
	"text/template"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/apitypes"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/config"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/stringutils"
	"golang.org/x/exp/slices"

	"userclouds.com/infra/ucerr"
)

type fileData struct {
	Package string
}

var fileHeaderTemplate = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
	"context"
	"reflect"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/userclouds/terraform-provider-userclouds/internal/provider/planmodifiers"

	"userclouds.com/infra/ucerr"
)

// Note: revive is complaining about stuttering in the generated schema names (e.g. an OpenAPI
// schema might be called "UserstoreColumnTFModel", and then we generate it in the "userstore"
// package, so it becomes "userstore.UserstoreColumnTFModel"), but these names are coming from the
// OpenAPI spec and no one is going to be reading this generated code anyways, so we should just
// silence the rule.
//revive:disable:exported

// boolplanmodifier is used in userstore schemas but not tokenizer
var _ = boolplanmodifier.RequiresReplace
`

type fieldData struct {
	// StructName of field within golang structs (should be upper camel case)
	StructName string
	// Name of field within JSON (should be lower snake case)
	JSONKey string
	// Name of field within Terraform schema (should be lower snake case)
	SchemaName string
	// Logical type of this field
	Type apitypes.APIType
	// Attributes to add to the Terraform schema
	ExtraTFAttributeFields map[string]string
}

type schemaData struct {
	TFSchemaAttributesMapName   string
	TFSchemaAttrTypeMapName     string
	TFModelStructName           string
	JSONClientModelStructName   string
	TFModelToJSONClientFuncName string
	JSONClientModelToTFFuncName string
	Fields                      []fieldData
}

var modelTemplate = `
// << .TFModelStructName >> is a Terraform model struct for the << .TFSchemaAttributesMapName >> schema.
type << .TFModelStructName >> struct {
	<<- range $i, $field := .Fields >>
	<< $field.StructName >> << $field.Type.TFModelType >> ` + "`tfsdk:\"<< $field.SchemaName >>\"`" + `
	<<- end >>
}

// << .JSONClientModelStructName >> stores data for use with jsonclient for making API requests.
type << .JSONClientModelStructName >> struct {
	<<- range $i, $field := .Fields >>
	<< $field.StructName >> *<< $field.Type.JSONClientModelType >> ` + "`json:\"<< $field.JSONKey >>,omitempty\" yaml:\"<< $field.JSONKey >>,omitempty\"`" + `
	<<- end >>
}

// << .TFSchemaAttrTypeMapName >> defines the attribute types for the << .TFSchemaAttributesMapName >> schema.
var << .TFSchemaAttrTypeMapName >> = map[string]attr.Type{
	<<- range $i, $field := .Fields >>
	"<< $field.SchemaName >>": << $field.Type.TFSchemaAttributeType >>,
	<<- end >>
}

// << .TFSchemaAttributesMapName >> defines the Terraform attributes schema.
var << .TFSchemaAttributesMapName >> = map[string]schema.Attribute{
	<<- range $i, $field := .Fields >>
	"<< $field.SchemaName >>": << $field.Type.TFSchemaAttributeText $field.ExtraTFAttributeFields >>,
	<<- end >>
}

// << .TFModelToJSONClientFuncName >> converts a Terraform model struct to a jsonclient model struct.
func << .TFModelToJSONClientFuncName >>(in *<< .TFModelStructName >>) (*<< .JSONClientModelStructName >>, error) {
	out := << .JSONClientModelStructName >>{}
	<<- if .Fields >>
	var err error
	<<- end >>
	<<- range $i, $field := .Fields >>
	out.<< $field.StructName >>, err = << $field.Type.TFModelToJSONClientFunc >>(&in.<< $field.StructName >>)
	if err != nil {
		return nil, ucerr.Errorf("failed to convert \"<< $field.SchemaName >>\" field: %+v", err)
	}
	<<- end >>
	return &out, nil
}

// << .JSONClientModelToTFFuncName >> converts a jsonclient model struct to a Terraform model struct.
func << .JSONClientModelToTFFuncName >>(in *<< .JSONClientModelStructName >>) (<< .TFModelStructName >>, error) {
	out := << .TFModelStructName >>{}
	<<- if .Fields >>
	var err error
	<<- end >>
	<<- range $i, $field := .Fields >>
	out.<< $field.StructName >>, err = << $field.Type.JSONClientModelToTFFunc >>(in.<< $field.StructName >>)
	if err != nil {
		return << $.TFModelStructName >>{}, ucerr.Errorf("failed to convert \"<< $field.SchemaName >>\" field: %+v", err)
	}
	<<- end >>
	return out, nil
}
`

func inferAPIType(spec *openapi3.Spec, schemaName *string, schema *openapi3.Schema) (apitypes.APIType, error) {
	// Unspecified or "any" type:
	if schema.Type == nil {
		// Note: In the future, Terraform may support an "any" type, but
		// the plugin framework doesn't support this right now (even though
		// the wire format does). As of writing, we shouldn't need this for
		// anything anyways.
		return nil, ucerr.Errorf(`schema has no specified type and we don't support an "any" type`)
	}
	// UserstoreResourceID references: we want to store *only* the UUID and
	// omit the name.
	if *schema.Type == openapi3.SchemaTypeObject && schemaName != nil && *schemaName == "UserstoreResourceID" {
		return &apitypes.UserstoreResourceID{
			Schema:                    schema,
			JSONClientModelStructName: GetJSONClientModelStructName(*schemaName),
		}, nil
	}
	// Nested objects:
	if *schema.Type == openapi3.SchemaTypeObject && schema.Properties != nil {
		if schemaName == nil {
			// If the object properties are declared as a ref to another schema, then we'll
			// already be generating structs/functions for that schema, and we can just
			// reference them below. If the properties were declared inline, we would need to
			// codegen some more stuff here, but we aren't doing that in our OpenAPI spec today.
			return nil, ucerr.Errorf("this is a nested object, but is missing a schema name to refer to. we don't support nested objects with properties declared inline yet")
		}
		return &apitypes.Object{
			Schema:                      schema,
			TFModelStructName:           GetTFModelStructName(*schemaName),
			TFSchemaAttributesMapName:   GetAttributesMapName(*schemaName),
			TFSchemaAttrTypeMapName:     GetAttrTypeMapName(*schemaName),
			JSONClientModelStructName:   GetJSONClientModelStructName(*schemaName),
			TFModelToJSONClientFuncName: GetTFModelToJSONClientFuncName(*schemaName),
			JSONClientModelToTFFuncName: GetJSONClientModelToTFFuncName(*schemaName),
		}, nil
	}
	// Maps:
	if *schema.Type == openapi3.SchemaTypeObject && schema.AdditionalProperties != nil {
		valueTypeSchemaOrRef := schema.AdditionalProperties.SchemaOrRef
		if valueTypeSchemaOrRef == nil {
			return nil, ucerr.Errorf("this looks like a map, but we currently require additionalProperties to specify a schema (we don't have support for specifying the type inline)")
		}
		valueTypeSchema, err := ResolveSchema(spec, valueTypeSchemaOrRef)
		if err != nil {
			return nil, ucerr.Errorf("could not resolve additionalProperties schema: %v", err)
		}
		var valueTypeSchemaName *string
		if schema.AdditionalProperties.SchemaOrRef.SchemaReference != nil {
			n := SchemaNameFromRef(schema.AdditionalProperties.SchemaOrRef.SchemaReference.Ref)
			valueTypeSchemaName = &n
		}
		valueType, err := inferAPIType(spec, valueTypeSchemaName, valueTypeSchema)
		if err != nil {
			return nil, ucerr.Errorf("could not infer map value type from additionalProperties: %v", err)
		}
		return &apitypes.Map{Schema: schema, ValueType: valueType}, nil
	}
	// Arrays:
	if *schema.Type == openapi3.SchemaTypeArray {
		if schema.Items == nil {
			return nil, ucerr.Errorf("items schema is missing")
		}
		itemsSchema, err := ResolveSchema(spec, schema.Items)
		if err != nil {
			return nil, ucerr.Errorf("could not resolve items schema: %v", err)
		}
		var itemsSchemaName *string
		if schema.Items.SchemaReference != nil {
			n := SchemaNameFromRef(schema.Items.SchemaReference.Ref)
			itemsSchemaName = &n
		}
		itemsType, err := inferAPIType(spec, itemsSchemaName, itemsSchema)
		if err != nil {
			return nil, ucerr.Errorf("could not infer array items type from items schema: %v", err)
		}
		return &apitypes.Array{Schema: schema, ChildType: itemsType}, nil
	}
	// UUIDs (custom string type):
	if *schema.Type == openapi3.SchemaTypeString && schema.Format != nil && *schema.Format == "uuid" {
		return &apitypes.UUID{Schema: schema}, nil
	}
	// String enums:
	if *schema.Type == openapi3.SchemaTypeString && schema.Enum != nil {
		values := []string{}
		for _, v := range schema.Enum {
			if s, ok := v.(string); ok {
				values = append(values, s)
			} else {
				return nil, ucerr.Errorf("schema specified type \"string\", but enum value %v is not a string", v)
			}
		}
		return &apitypes.StringEnum{Schema: schema, Values: values}, nil
	}
	// Basic types:
	basicTypesMap := map[string]apitypes.APIType{
		"boolean": &apitypes.Bool{Schema: schema},
		"integer": &apitypes.Int{Schema: schema},
		"float":   &apitypes.Float{Schema: schema},
		"string":  &apitypes.String{Schema: schema},
	}
	return basicTypesMap[string(*schema.Type)], nil
}

func gatherSchemaProperties(spec *openapi3.Spec, schemaName string, overrideInfo *config.SchemaOverride) (schemaData, error) {
	schemaOrRef := spec.Components.Schemas.MapOfSchemaOrRefValues[schemaName]
	schema, err := ResolveSchema(spec, &schemaOrRef)
	if err != nil {
		log.Fatalf("error while resolving schema %s: %v", schemaName, err)
	}
	data := schemaData{
		TFSchemaAttributesMapName:   GetAttributesMapName(schemaName),
		TFSchemaAttrTypeMapName:     GetAttrTypeMapName(schemaName),
		TFModelStructName:           GetTFModelStructName(schemaName),
		JSONClientModelStructName:   GetJSONClientModelStructName(schemaName),
		TFModelToJSONClientFuncName: GetTFModelToJSONClientFuncName(schemaName),
		JSONClientModelToTFFuncName: GetJSONClientModelToTFFuncName(schemaName),
	}
	var propNames []string
	for name := range schema.Properties {
		if name == "is_system" {
			// System objects should never be managed by Terraform, and this property can't change,
			// so there is no point in exposing it in Terraform.
			continue
		}

		propNames = append(propNames, name)
	}
	sort.Strings(propNames)
	for _, name := range propNames {
		prop := schema.Properties[name]
		propSchema, err := ResolveSchema(spec, &prop)
		if err != nil {
			return schemaData{}, ucerr.Errorf("could not resolve schema reference for schema %s property %s: %v", schemaName, name, err)
		}

		// Some properties are declared by referencing other schemas, e.g. "id" properties often
		// reference the UuidUUID schema
		var propSchemaName *string
		if prop.SchemaReference != nil {
			n := SchemaNameFromRef(prop.SchemaReference.Ref)
			propSchemaName = &n
		}

		t, err := inferAPIType(spec, propSchemaName, propSchema)
		if err != nil {
			return schemaData{}, ucerr.Errorf("could not infer type for schema %s property %s: %v", schemaName, name, err)
		}
		fieldData := fieldData{
			StructName:             stringutils.ToUpperCamel(name),
			JSONKey:                name,
			SchemaName:             name,
			Type:                   t,
			ExtraTFAttributeFields: map[string]string{},
		}

		if schema.Required != nil && slices.Contains(schema.Required, name) {
			fieldData.ExtraTFAttributeFields["Required"] = "true"
		} else if name == "version" {
			// Version is like an etag -- should be set by server, not by practitioners
			fieldData.ExtraTFAttributeFields["Computed"] = "true"
		} else if overrideInfo != nil && slices.Contains(overrideInfo.ReadonlyProperties, name) {
			// Handle properties marked readonly by generation config
			fieldData.ExtraTFAttributeFields["Computed"] = "true"
		} else {
			// Computed: these values can be populated by the provider (e.g. on read) if unspecified
			// in the terraform config
			fieldData.ExtraTFAttributeFields["Computed"] = "true"
			fieldData.ExtraTFAttributeFields["Optional"] = "true"
		}

		if name == "id" {
			// IDs should be stable; if an ID is not explicitly set in
			// Terraform, keep using the value from previous state
			fieldData.ExtraTFAttributeFields["PlanModifiers"] = `[]planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			}`
		} else if name == "version" {
			// When sending update requests, we need to send the
			// last-received version number as an etag
			fieldData.ExtraTFAttributeFields["PlanModifiers"] = `[]planmodifier.Int64{
				planmodifiers.IncrementOnUpdate(),
			}`
		}
		if overrideInfo != nil && slices.Contains(overrideInfo.ImmutableProperties, name) {
			if name == "id" || name == "version" {
				// We have different PlanModifiers set for these attributes
				// above (don't want to override), and it doesn't make sense to
				// mark them immutable anyways
				return schemaData{}, ucerr.Errorf("id and version properties cannot be marked immutable")
			}
			fieldData.ExtraTFAttributeFields["PlanModifiers"] = `[]planmodifier.` + t.GetTFPlanModifierType() + `{
				` + t.GetTFPlanModifierPackageName() + `.RequiresReplace(),
			}`
		}

		data.Fields = append(data.Fields, fieldData)
	}
	return data, nil
}

// GenSchemas generates the code for Terraform/jsonclient schemas, attribute maps, and conversion
// functions.
func GenSchemas(ctx context.Context, outDir string, conf *config.Spec, spec *openapi3.Spec) {
	fileHeaderTemplate := template.Must(template.New("fileHeaderTemplate").Delims("<<", ">>").Parse(fileHeaderTemplate))
	modelTemplate := template.Must(template.New("modelTemplate").Delims("<<", ">>").Parse(modelTemplate))

	buf := bytes.NewBuffer([]byte{})

	if err := fileHeaderTemplate.Execute(buf, fileData{Package: conf.GeneratedFilePackage}); err != nil {
		log.Fatalf("error executing template: %v", err)
	}

	var schemaNames []string
	for name := range spec.Components.Schemas.MapOfSchemaOrRefValues {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	for _, name := range schemaNames {
		var overrideInfo *config.SchemaOverride
		if i, ok := conf.SchemaOverrides[name]; ok {
			overrideInfo = &i
		}
		data, err := gatherSchemaProperties(spec, name, overrideInfo)
		if err != nil {
			log.Printf("WARNING: skipping schema generation for %s: %v", name, err) // lint: ucwrapper-safe
			continue
		}

		if err := modelTemplate.Execute(buf, data); err != nil {
			log.Fatalf("error executing template: %v", err)
		}
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error formatting source: %v", err)
	}
	fh, err := os.Create(outDir + "/" + conf.GeneratedFilePackage + "/schemas_generated.go")
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	if _, err := fh.Write(formatted); err != nil {
		log.Fatalf("error writing output file: %v", err)
	}
	if err := fh.Close(); err != nil {
		log.Fatalf("error closing output file: %v", err)
	}
}
