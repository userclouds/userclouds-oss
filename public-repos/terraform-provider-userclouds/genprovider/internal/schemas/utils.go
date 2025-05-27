package schemas

import (
	"regexp"

	"github.com/swaggest/openapi-go/openapi3"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/stringutils"

	"userclouds.com/infra/ucerr"
)

var schemaRefRegex = regexp.MustCompile(`^#\/components\/schemas\/([^\/]+)$`)

// SchemaNameFromRef returns a schema name from an OpenAPI ref string.
func SchemaNameFromRef(ref string) string {
	match := schemaRefRegex.FindStringSubmatch(ref)
	if match == nil {
		return ""
	}
	return match[1]
}

// GetTFModelStructName generates the name of the codegen'ed Terraform model struct.
func GetTFModelStructName(schemaName string) string {
	return stringutils.ToUpperCamel(schemaName) + "TFModel"
}

// GetJSONClientModelStructName generates the name of the codegen'ed struct for use for
// sending/receiving jsonclient requests/responses
func GetJSONClientModelStructName(schemaName string) string {
	return stringutils.ToUpperCamel(schemaName) + "JSONClientModel"
}

// GetAttrTypeMapName generates the name of the codegen'ed map of Terraform schema types (attr.Type).
func GetAttrTypeMapName(schemaName string) string {
	return stringutils.ToUpperCamel(schemaName) + "AttrTypes"
}

// GetAttributesMapName generates the name of the codegen'ed map of Terraform schema attributes.
func GetAttributesMapName(schemaName string) string {
	return stringutils.ToUpperCamel(schemaName) + "Attributes"
}

// GetTFModelToJSONClientFuncName generates the name of the codegen'ed function for converting the
// TF model struct to a jsonclient model struct.
func GetTFModelToJSONClientFuncName(schemaName string) string {
	return GetTFModelStructName(schemaName) + "ToJSONClient"
}

// GetJSONClientModelToTFFuncName generates the name of the codegen'ed function for converting the
// jsonclient model struct to a TF model struct.
func GetJSONClientModelToTFFuncName(schemaName string) string {
	return GetJSONClientModelStructName(schemaName) + "ToTF"
}

// ResolveSchema resolves an openapi SchemaOrRef to a Schema.
func ResolveSchema(spec *openapi3.Spec, schemaOrRef *openapi3.SchemaOrRef) (*openapi3.Schema, error) {
	if schemaOrRef.Schema != nil {
		return schemaOrRef.Schema, nil
	}
	schemaName := SchemaNameFromRef(schemaOrRef.SchemaReference.Ref)
	if schemaName == "" {
		return nil, ucerr.Errorf("invalid schema reference: %s", schemaOrRef.SchemaReference.Ref)
	}
	resolved := spec.Components.Schemas.MapOfSchemaOrRefValues[schemaName]
	schema, err := ResolveSchema(spec, &resolved)
	if err != nil {
		return nil, ucerr.Errorf("error while resolving schema reference %s: %v", schemaOrRef.SchemaReference.Ref, err)
	}
	return schema, nil
}
