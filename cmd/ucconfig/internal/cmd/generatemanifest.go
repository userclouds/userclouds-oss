package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"userclouds.com/cmd/ucconfig/internal/manifest"
	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// GenerateNewManifest implements a "ucconfig gen-manifest" subcommand that generates a new manifest.
func GenerateNewManifest(ctx context.Context, idpClient *idp.Client, fqtn string, manifestPath string) error {
	uclog.Infof(ctx, "Generating new manifest from live resource state...")

	manifestBasename := filepath.Base(manifestPath)
	externValuesDirName := manifestBasename[:len(manifestBasename)-len(filepath.Ext(manifestBasename))] + "_values"
	externValuesDirPath, err := filepath.Abs(filepath.Dir(manifestPath) + "/" + externValuesDirName)
	if err != nil {
		return ucerr.Friendlyf(err, "failed to get absolute path for storing attribute values externally")
	}

	// Clear out the target directory if it already exists
	if err := os.RemoveAll(externValuesDirPath); err != nil {
		return ucerr.Friendlyf(err, "failed to clear directory %s for storing attribute values externally", externValuesDirPath)
	}
	if err := os.MkdirAll(externValuesDirPath, 0755); err != nil {
		return ucerr.Friendlyf(err, "failed to create directory %s for storing attribute values externally", externValuesDirPath)
	}

	mfest, err := manifest.GenerateNewManifest(ctx, idpClient, fqtn, &manifest.ExternValuesDirConfig{
		AbsolutePath:             externValuesDirPath,
		RelativePathFromManifest: "./" + externValuesDirName,
	})
	if err != nil {
		return ucerr.Friendlyf(err, "failed to generate manifest")
	}

	var serialized []byte
	switch filepath.Ext(manifestPath) {
	case ".json":
		serialized, err = json.MarshalIndent(mfest, "", "  ")
	case ".yaml":
		serialized, err = yaml.Marshal(mfest)
	default:
		return ucerr.Friendlyf(nil, "manifest path must have .json or .yaml extension")
	}
	if err != nil {
		return ucerr.Friendlyf(err, "failed to serialize manifest")
	}

	if err := os.WriteFile(manifestPath, serialized, 0644); err != nil {
		return ucerr.Friendlyf(err, "failed to write manifest")
	}

	uclog.Infof(ctx, "Wrote %d resources into manifest: %s", len(mfest.Resources), manifestPath)
	return nil
}
