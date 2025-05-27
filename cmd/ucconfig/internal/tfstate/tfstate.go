package tfstate

import (
	"reflect"

	"github.com/gofrs/uuid"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/resourcetypes"
	"userclouds.com/infra/ucerr"
)

// The terraform tfstate file format is internal to Terraform, and there aren't any libraries (that
// I found) for working with the format. It's subject to change with different Terraform versions.
// There is some useful documentation here:
// https://discuss.hashicorp.com/t/terraform-state-schema/48660/2
// As an internal file format, we are subject to breakage here. However, the only other alternative
// for producing the state file is to run `terraform import` for each resource we want to import,
// which has two problems:
//
// * It's going to be extremely slow, since we have to run `terraform import` on every individual
//   resource (may have 10s or 100s of resources), and `terraform import` has to initialize the
//   provider from scratch, then the provider makes GET requests to fetch the full resource state
//   (even though we may already have it from our IDP client List* calls). Each import takes on the
//   order of 1+ second.
// * You can only run `terraform import` on resources that exist in the terraform config. That means
//   we'd need to generate terraform configuration even for resources we want to delete (i.e.
//   resources that exist live, but do not exist in the manifest). Then we'd need to delete those
//   resources from the tf config before running `terraform apply`.
//
// In practice, the tfstate file doesn't change format often, and since ucconfig generates the
// config *and* invokes Terraform, we should be able to write tests that catch any issues.

// State is the top-level struct for a terraform.tfstate file
type State struct {
	// Version contains the state file schema version
	Version int `json:"version" yaml:"version"`
	// TerraformVersion contains the version of Terraform that was used to create this state file
	TerraformVersion string `json:"terraform_version" yaml:"terraform_version"`
	// Serial is a monotonically increasing number that is incremented each time the state file is
	// modified
	Serial int `json:"serial" yaml:"serial"`
	// Lineage is a UUID that is generated when the state file is created for the first time. Good
	// explanation of `serial` and `lineage` here:
	// https://discuss.hashicorp.com/t/terraform-state-schema/48660/2
	Lineage string `json:"lineage" yaml:"lineage"`
	// Outputs contains the outputs from the terraform configuration
	Outputs map[string]Output `json:"outputs" yaml:"outputs"`
	// Resources contains information about the resources being managed by Terraform
	Resources []Resource `json:"resources" yaml:"resources"`
	// CheckResults contains the results of any condition checks such as preconditions,
	// postconditions, and variable validation
	CheckResults []CheckResult `json:"check_results" yaml:"check_results"`
}

// Output contains information about Terraform output values
type Output struct {
	// TODO: we currently don't generate any outputs, so leaving this blank
}

// Resource contains information about Terraform resource
type Resource struct {
	// Module contains the address to the module being used to provision this resource (blank for
	// the root module)
	Module string `json:"module,omitempty" yaml:"module,omitempty"`
	// Mode contains the mode of the resource (managed or data)
	Mode string `json:"mode" yaml:"mode"`
	// Type contains the type of the resource (e.g. "userclouds_userstore_column")
	Type string `json:"type" yaml:"type"`
	// Name contains the name of the resource (e.g. "address")
	Name string `json:"name" yaml:"name"`
	// Provider contains the full path for the provider that manages this resource
	Provider string `json:"provider" yaml:"provider"`
	// Instances contains one or more instances of this resource
	Instances []Instance `json:"instances" yaml:"instances"`
}

// Instance contains information about a specific instantiation of a Terraform resource
type Instance struct {
	// Status contains the status of the resource (e.g. "tainted")
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	// SchemaVersion contains the resource-specific schema version within the provider
	SchemaVersion int `json:"schema_version" yaml:"schema_version"`
	// Attributes contains the attributes of the resource as defined by the provider schema
	Attributes map[string]any `json:"attributes" yaml:"attributes"`
	// SensitiveAttributes contains an array of paths within `attributes` to mark as sensitive
	SensitiveAttributes []string `json:"sensitive_attributes" yaml:"sensitive_attributes"`
	// Dependencies contains an array of resource addresses that this resource is dependent on
	Dependencies []string `json:"dependencies" yaml:"dependencies"`
}

// CheckResult contains information about the results of a Terraform configuration check
type CheckResult struct {
	// TODO: we currently aren't using any configuration checks, so leaving this blank
}

// getDependenciesFromAttribute examines an attribute of a resource and, based
// on the value of that attribute, infers dependencies on other resources.
// Dependencies are returned as a list of Terraform resource addresses.
func getDependenciesFromAttribute(attrVal any, currAttrPath string, forTfTypeSuffix string, allResources *[]liveresource.Resource) ([]string, error) {
	v := reflect.ValueOf(attrVal)

	if v.Kind() == reflect.Pointer {
		return getDependenciesFromAttribute(v.Elem().Interface(), currAttrPath, forTfTypeSuffix, allResources)
	}

	forType := resourcetypes.GetByTerraformTypeSuffix(forTfTypeSuffix)
	if v.Kind() == reflect.String && forType.References[currAttrPath] != "" && !uuid.Must(uuid.FromString(v.String())).IsNil() {
		var referenced *liveresource.Resource
		for i := range *allResources {
			if (*allResources)[i].TerraformTypeSuffix == forType.References[currAttrPath] && (*allResources)[i].ResourceUUID == v.String() {
				referenced = &(*allResources)[i]
				break
			}
		}
		if referenced == nil {
			return []string{}, ucerr.Errorf("could not find referenced live resource for attrPath %s with UUID %v", currAttrPath, v.String())
		}
		if referenced.IsSystem {
			// Don't write dependencies on system resources, since those are
			// excluded from the tf config
			return []string{}, nil
		}
		return []string{"userclouds_" + referenced.TerraformTypeSuffix + "." + referenced.TerraformResourceName()}, nil
	}

	if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
		var out []string
		for i := 0; i < v.Len(); i++ {
			deps, err := getDependenciesFromAttribute(v.Index(i).Interface(), currAttrPath, forTfTypeSuffix, allResources)
			if err != nil {
				return []string{}, ucerr.Wrap(err)
			}
			out = append(out, deps...)
		}
		return out, nil
	}

	if v.Kind() == reflect.Map {
		var out []string
		for _, k := range v.MapKeys() {
			deps, err := getDependenciesFromAttribute(v.MapIndex(k).Interface(), currAttrPath+"."+k.String(), forTfTypeSuffix, allResources)
			if err != nil {
				return []string{}, ucerr.Wrap(err)
			}
			out = append(out, deps...)
		}
		return out, nil
	}

	return []string{}, nil
}

// CreateState creates a State struct from a list of live resources
func CreateState(resources *[]liveresource.Resource) (State, error) {
	lineage, err := uuid.NewV4()
	if err != nil {
		return State{}, ucerr.Wrap(err)
	}

	var stateResources []Resource
	for _, resource := range *resources {
		// Omit system resources from state, since they are also omitted from the configuration
		if resource.IsSystem {
			continue
		}
		attributes := map[string]any{}
		dependencies := []string{}
		for k, v := range resource.Attributes {
			attributes[k] = v
			d, err := getDependenciesFromAttribute(v, k, resource.TerraformTypeSuffix, resources)
			if err != nil {
				return State{}, ucerr.Wrap(err)
			}
			dependencies = append(dependencies, d...)
		}
		attributes["id"] = resource.ResourceUUID
		stateResources = append(stateResources, Resource{
			Mode:     "managed",
			Type:     "userclouds_" + resource.TerraformTypeSuffix,
			Name:     resource.TerraformResourceName(),
			Provider: "provider[\"registry.terraform.io/userclouds/userclouds\"]",
			Instances: []Instance{
				{
					SchemaVersion:       0,
					Attributes:          attributes,
					SensitiveAttributes: []string{},
					Dependencies:        dependencies,
				},
			},
		})
	}

	state := State{
		Version:          4,
		TerraformVersion: "1.5.3",
		Serial:           1,
		Lineage:          lineage.String(),
		Outputs:          map[string]Output{},
		Resources:        stateResources,
		CheckResults:     []CheckResult{},
	}

	return state, nil
}
