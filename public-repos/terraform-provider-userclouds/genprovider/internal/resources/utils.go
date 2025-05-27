package resources

import "github.com/userclouds/terraform-provider-userclouds/genprovider/internal/stringutils"

// TFTypeNameSuffixToResourceName returns the name of the generated resource type, e.g.
// UserstoreColumnResource.
func TFTypeNameSuffixToResourceName(typeNameSuffix string) string {
	return stringutils.ToUpperCamel(typeNameSuffix) + "Resource"
}

// TFTypeNameSuffixToNewResourceFuncName returns the name of the generated function that returns a
// new instance of a resource, e.g. NewUserstoreColumnResource()
func TFTypeNameSuffixToNewResourceFuncName(typeNameSuffix string) string {
	return "New" + TFTypeNameSuffixToResourceName(typeNameSuffix)
}
