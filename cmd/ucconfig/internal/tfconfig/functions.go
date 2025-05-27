package tfconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/infra/ucerr"
)

type functionInvocation struct {
	Name       string
	Params     []any
	PathSuffix []string
}

func (i *functionInvocation) dispatch(ctx *GenerationContext) (hclwrite.Tokens, error) {
	if i.Name == "UC_MANIFEST_ID" {
		return ucManifestID(i, ctx)
	}
	if i.Name == "UC_SYSTEM_OBJECT" {
		return ucSystemObject(i, ctx)
	}
	if i.Name == "FILE" {
		return readFile(i, ctx)
	}
	return []*hclwrite.Token{}, ucerr.Errorf("unknown function %s", i.Name)
}

func parseFunctionInvocation(invocation string) *functionInvocation {
	baseRegex := regexp.MustCompile(`^@(?P<funcname>[A-Z_]+)\((?P<params>.*?)?\)(?P<pathsuffix>(?:\.[A-Za-z0-9_-]+)*)$`)
	groups := baseRegex.FindStringSubmatch(invocation)
	if len(groups) == 0 {
		return nil
	}
	out := functionInvocation{
		Name: groups[1],
	}
	params := groups[2]
	for _, param := range strings.Split(params, ",") {
		param = strings.TrimSpace(param)
		if param[0] == '"' && param[len(param)-1] == '"' {
			out.Params = append(out.Params, param[1:len(param)-1])
		} else if b, err := strconv.ParseBool(param); err != nil {
			out.Params = append(out.Params, b)
		} else if u, err := strconv.ParseUint(param, 10, 64); err != nil {
			out.Params = append(out.Params, u)
		} else if i, err := strconv.ParseInt(param, 10, 64); err != nil {
			out.Params = append(out.Params, i)
		} else if f, err := strconv.ParseFloat(param, 64); err != nil {
			out.Params = append(out.Params, f)
		} else {
			return nil
		}
	}
	pathsuffix := groups[3]
	out.PathSuffix = strings.Split(pathsuffix, ".")[1:]
	return &out
}

func ucManifestID(invocation *functionInvocation, ctx *GenerationContext) (hclwrite.Tokens, error) {
	if len(invocation.Params) != 1 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_MANIFEST_ID takes exactly 1 parameter")
	}
	if reflect.ValueOf(invocation.Params[0]).Kind() != reflect.String {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_MANIFEST_ID takes a string parameter")
	}
	manifestID := invocation.Params[0].(string)
	var matchingResource *manifest.Resource
	for i := range ctx.Manifest.Resources {
		if ctx.Manifest.Resources[i].ManifestID == manifestID {
			matchingResource = &ctx.Manifest.Resources[i]
			break
		}
	}
	if matchingResource == nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not find resource with manifest ID %s for UC_MANIFEST_ID invocation", manifestID)
	}
	if len(invocation.PathSuffix) == 0 {
		return []*hclwrite.Token{}, ucerr.Errorf("expected manifest to access attributes of resource returned by UC_MANIFEST_ID")
	}
	traversal := hcl.Traversal{
		hcl.TraverseRoot{Name: "userclouds_" + matchingResource.TerraformTypeSuffix},
		hcl.TraverseAttr{Name: "manifestid-" + matchingResource.ManifestID},
	}
	for _, pathPart := range invocation.PathSuffix {
		traversal = append(traversal, hcl.TraverseAttr{Name: pathPart})
	}
	return hclwrite.TokensForTraversal(traversal), nil
}

func ucSystemObject(invocation *functionInvocation, ctx *GenerationContext) (hclwrite.Tokens, error) {
	if len(invocation.Params) != 2 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT takes exactly 2 parameters")
	}
	if reflect.ValueOf(invocation.Params[0]).Kind() != reflect.String ||
		reflect.ValueOf(invocation.Params[1]).Kind() != reflect.String {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT parameters must be strings")
	}
	terraformTypeSuffix := invocation.Params[0].(string)
	objectName := invocation.Params[1].(string)
	var matchingResource *liveresource.Resource
	for _, resource := range *ctx.LiveResources {
		if resource.TerraformTypeSuffix == terraformTypeSuffix && resource.IsSystem && resource.Attributes["name"].(string) == objectName {
			matchingResource = &resource
			break
		}
	}
	if matchingResource == nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not find system object with type %s and name %s for UC_SYSTEM_OBJECT invocation", terraformTypeSuffix, objectName)
	}
	if len(invocation.PathSuffix) != 0 {
		return []*hclwrite.Token{}, ucerr.Errorf("UC_SYSTEM_OBJECT returns a string, so path suffixes may not be used")
	}
	return hclwrite.TokensForValue(cty.StringVal(matchingResource.ResourceUUID)), nil
}

func readFile(invocation *functionInvocation, ctx *GenerationContext) (hclwrite.Tokens, error) {
	if len(invocation.Params) != 1 {
		return []*hclwrite.Token{}, ucerr.Errorf("FILE takes exactly 1 parameter")
	}
	if reflect.ValueOf(invocation.Params[0]).Kind() != reflect.String {
		return []*hclwrite.Token{}, ucerr.Errorf("FILE takes a string parameter")
	}
	filePath := invocation.Params[0].(string)
	if !strings.HasPrefix(filePath, "/") {
		filePath = filepath.Dir(ctx.ManifestFilePath) + "/" + filePath
	}
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return []*hclwrite.Token{}, ucerr.Errorf("error reading file %s: %v", filePath, err)
	}
	// Remove trailing newline if present (inserted by many editors, and by our
	// manifest generation)
	return hclwrite.TokensForValue(cty.StringVal(strings.TrimSuffix(string(contents), "\n"))), nil
}
