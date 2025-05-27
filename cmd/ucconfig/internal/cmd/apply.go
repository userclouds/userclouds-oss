package cmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"userclouds.com/cmd/ucconfig/internal/liveresource"
	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/cmd/ucconfig/internal/tfconfig"
	"userclouds.com/cmd/ucconfig/internal/tfstate"
	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func writeTerraformRC(ctx context.Context, rcPath string, tfProviderDevDirPath string) error {
	abspath, err := filepath.Abs(tfProviderDevDirPath)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to get absolute path to terraform-provider-userclouds dev directory %v", tfProviderDevDirPath)
	}
	config := `
		provider_installation {
			dev_overrides {
				"userclouds/userclouds" = "` + abspath + `"
			}
			direct {}
		}
	`
	return ucerr.Wrap(os.WriteFile(rcPath, []byte(config), 0644))
}

func genTerraform(ctx context.Context, mfestPath string, mfest *manifest.Manifest, fqtn string, resources *[]liveresource.Resource, tfDir string, tfProviderVersionConstraint string) error {
	if tfProviderVersionConstraint == "" {
		// Require at least v0.1.8 for support for column search indexing
		tfProviderVersionConstraint = ">= 0.1.8"
	}
	tfText, err := tfconfig.GenConfig(&tfconfig.GenerationContext{
		ManifestFilePath:            mfestPath,
		Manifest:                    mfest,
		FQTN:                        fqtn,
		LiveResources:               resources,
		TFProviderVersionConstraint: tfProviderVersionConstraint,
	})
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to generate Terraform config")
	}

	err = os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte(tfText), 0644)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to write generated Terraform config")
	}

	// Generate Terraform state for existing resources
	state, err := tfstate.CreateState(resources)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to generate Terraform state")
	}
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to marshal Terraform state")
	}
	return ucerr.Wrap(os.WriteFile(filepath.Join(tfDir, "terraform.tfstate"), stateBytes, 0644))
}

// Apply implements a "ucconfig apply" subcommand that applies a manifest.
func Apply(ctx context.Context, dryRun, autoApprove bool, idpClient *idp.Client, fqtn string, tenantURL string, clientID string, clientSecret string, manifestPath string, tfProviderVersionConstraint string, tfProviderDevDirPath string) error {
	if dryRun && autoApprove {
		return ucerr.Friendlyf(nil, "dry run and auto approve flags are mutually exclusive")
	}

	uclog.Infof(ctx, "Reading manifest from %s...", manifestPath)
	manifestText, err := os.ReadFile(manifestPath)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to read manifest file")
	}

	mfest := manifest.Manifest{}
	switch filepath.Ext(manifestPath) {
	case ".json":
		if err := json.Unmarshal(manifestText, &mfest); err != nil {
			return ucerr.Friendlyf(err, "Failed to decode JSON")
		}
	case ".yaml":
		if err := yaml.Unmarshal(manifestText, &mfest); err != nil {
			return ucerr.Friendlyf(err, "Failed to decode YAML")
		}
	default:
		return ucerr.Friendlyf(nil, "Manifest path must have .json or .yaml extension")
	}
	if err := mfest.Validate(fqtn); err != nil {
		return ucerr.Friendlyf(err, "Failed to validate manifest")
	}

	uclog.Infof(ctx, "Fetching live resources...")
	resources, err := liveresource.GetLiveResources(ctx, idpClient)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to fetch live resources")
	}
	err = mfest.MatchLiveResources(ctx, &resources, fqtn)
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to match manifest entries to live resources")
	}

	uclog.Infof(ctx, "Generating Terraform...")
	dname, err := os.MkdirTemp("", "ucconfig-terraform")
	if err != nil {
		return ucerr.Friendlyf(err, "Failed to create temporary directory")
	}
	uclog.Infof(ctx, "Terraform files will be generated in %s", dname)

	err = genTerraform(ctx, manifestPath, &mfest, fqtn, &resources, dname, tfProviderVersionConstraint)
	if err != nil {
		return ucerr.Friendlyf(err, "Error during Terraform generation")
	}

	env := os.Environ()
	if tfProviderDevDirPath != "" {
		terraformRCPath := dname + "/.terraformrc"
		if err := writeTerraformRC(ctx, terraformRCPath, tfProviderDevDirPath); err != nil {
			return ucerr.Friendlyf(err, "Failed to write Terraform RC file")
		}
		uclog.Infof(ctx, "Setting TF_CLI_CONFIG_FILE=%v to enable usage of local dev build of UC TF provider", terraformRCPath)
		env = append(env, "TF_CLI_CONFIG_FILE="+terraformRCPath)
	}

	uclog.Infof(ctx, "Running terraform init...")
	cmd := exec.Command("terraform", "init")
	cmd.Dir = dname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return ucerr.Friendlyf(err, "Failed to run terraform init. Generated terraform files are in %s", dname)
	}

	cmdArgs := make([]string, 0, 3)
	if dryRun {
		cmdArgs = append(cmdArgs, "plan")
	} else {
		cmdArgs = append(cmdArgs, "apply")
		if autoApprove {
			cmdArgs = append(cmdArgs, "-auto-approve")
		}
	}
	uclog.Infof(ctx, "Running terraform %s...", strings.Join(cmdArgs, " "))
	cmd = exec.Command("terraform", cmdArgs...)
	cmd.Dir = dname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Env = append(cmd.Env, "USERCLOUDS_TENANT_URL="+tenantURL)
	cmd.Env = append(cmd.Env, "USERCLOUDS_CLIENT_ID="+clientID)
	cmd.Env = append(cmd.Env, "USERCLOUDS_CLIENT_SECRET="+clientSecret)
	if err := cmd.Run(); err != nil {
		return ucerr.Friendlyf(err, "Failed to run terraform apply. Generated terraform files are in %s", dname)
	}

	// TODO: it could be a nice feature to prompt the user to ask whether to
	// write the manifest with updated matched resource<->manifest ID mappings.
	// However, if they have added comments to the manifest or made any
	// formatting customizations, those would get overwritten, so I have been
	// waiting to see whether anyone asks for this.

	return nil
}
