package tfconfig

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/infra/ucerr"
)

// GenerationContext stores info that is needed for generating the Terraform
// configuration
type GenerationContext struct {
	ManifestFilePath string
	Manifest         *manifest.Manifest
	FQTN             string // fully-qualified tenant name
	LiveResources    *[]liveresource.Resource
	// TFProviderVersionConstraint specifies the version constraint that should be used for the terraform-provider-userclouds provider instantiation
	TFProviderVersionConstraint string // e.g. "~> 1.0"
}

func genResourceConfig(resource *manifest.Resource, ctx *GenerationContext, body *hclwrite.Body) error {
	block := body.AppendNewBlock("resource", []string{"userclouds_" + resource.TerraformTypeSuffix, "manifestid-" + resource.ManifestID})
	var resourceUUID string
	if resource.ResourceUUIDs[ctx.FQTN] != "" {
		resourceUUID = resource.ResourceUUIDs[ctx.FQTN]
	} else {
		resourceUUID = resource.ResourceUUIDs["__DEFAULT"]
	}
	block.Body().SetAttributeValue("id", cty.StringVal(resourceUUID))
	var keys []string
	for key := range resource.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		tokens, err := toHclTokens(resource.Attributes[key], ctx)
		if err != nil {
			return ucerr.Errorf("Manifest ID %s: %v", resource.ManifestID, err)
		}
		block.Body().SetAttributeRaw(key, tokens)
	}
	body.AppendNewline()

	return nil
}

// GenConfig generates a terraform config file from a ucconfig manifest
func GenConfig(ctx *GenerationContext) (string, error) {
	// required providers
	file := hclwrite.NewEmptyFile()
	file.Body().
		AppendNewBlock("terraform", []string{}).Body().
		AppendNewBlock("required_providers", []string{}).Body().
		SetAttributeValue("userclouds", cty.ObjectVal(map[string]cty.Value{
			"source":  cty.StringVal("registry.terraform.io/userclouds/userclouds"),
			"version": cty.StringVal(ctx.TFProviderVersionConstraint),
		}))
	file.Body().AppendNewline()

	// provider initialization
	file.Body().AppendNewBlock("provider", []string{"userclouds"})
	file.Body().AppendNewline()

	// gen resources
	for _, resource := range ctx.Manifest.Resources {
		if err := genResourceConfig(&resource, ctx, file.Body()); err != nil {
			return "", ucerr.Wrap(err)
		}
	}

	var b strings.Builder
	if _, err := file.WriteTo(&b); err != nil {
		return "", ucerr.Wrap(err)
	}
	return b.String(), nil
}
