// Package yamlconfig provides helpers to load YAML configuration into any struct.
package yamlconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"

	"userclouds.com/infra"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Environment keys for config settings
// We use these instead of command line args because it works better with `go test`
const (
	EnvKeyConfigDir = "UC_CONFIG_DIR"
)

// LoadParams configures config loading behavior
type LoadParams struct {
	BaseDirs                 []string
	Universe                 universe.Universe
	LoadBaseEnvConfig        bool
	AllowUnknownFields       bool
	IsSpecificConfigOptional bool
}

//go:generate genvalidate LoadParams

// GetBaseDirs returns the list of directories to search for config files
func GetBaseDirs() []string {
	return strings.Split(os.Getenv(EnvKeyConfigDir), ",")
}

func (p LoadParams) extraValidate() error {
	if len(p.BaseDirs) == 0 {
		return ucerr.Errorf("Config base directories not set,'%s' must be set", EnvKeyConfigDir)
	}
	return nil
}

// GetLoadParams returns the default LoadParams for loading config files
func GetLoadParams(loadBaseEnv, allowUnknownFields, isSpecificConfigOptional bool) LoadParams {
	uv := universe.Current()
	return LoadParams{
		BaseDirs:           GetBaseDirs(),
		Universe:           uv,
		LoadBaseEnvConfig:  loadBaseEnv,
		AllowUnknownFields: allowUnknownFields,
		// is the <service/db>/<env/universe> config file optional?
		IsSpecificConfigOptional: isSpecificConfigOptional,
	}
}

func (p LoadParams) getDirs(suffix string) []string {
	dirs := make([]string, 0, len(p.BaseDirs))
	for _, dir := range p.BaseDirs {
		dirs = append(dirs, filepath.Join(dir, suffix))
	}
	return dirs
}

// LoadToolConfig loads the configuration for the specified tool
func LoadToolConfig(ctx context.Context, service string, cfg infra.Validateable) error {
	return ucerr.Wrap(LoadEnv(ctx, service, cfg, GetLoadParams(false, false, false)))
}

// LoadServiceConfig loads the configuration for the specified service
func LoadServiceConfig(ctx context.Context, service service.Service, cfg infra.Validateable) error {
	return ucerr.Wrap(LoadEnv(ctx, string(service), cfg, GetLoadParams(true, false, false)))
}

// LoadDatabaseConfig loads the configuration for the specified database
func LoadDatabaseConfig(ctx context.Context, service string, cfg infra.Validateable, loadBaseEnv bool) error {
	return ucerr.Wrap(LoadEnv(ctx, service, cfg, GetLoadParams(loadBaseEnv, true, true)))
}

// LoadEnv loads YAML configuration from the specified directory for the specified environment.
func LoadEnv(ctx context.Context, service string, cfg infra.Validateable, params LoadParams) error {
	if err := params.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	serviceDirs := params.getDirs(service)
	paths := []string{}
	if params.LoadBaseEnvConfig {
		baseEnvCfg, err := getConfigFilePath(params.BaseDirs, fmt.Sprintf("base_%v", params.Universe), true)
		if err != nil {
			return ucerr.Wrap(err)
		}
		paths = append(paths, baseEnvCfg)
	}

	// Load service-specific config last, base (if exist) + universe.
	baseConfigPath, err := getConfigFilePath(serviceDirs, "base", false)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if baseConfigPath != "" {
		paths = append(paths, baseConfigPath)
	}

	path, err := getConfigFilePath(serviceDirs, string(params.Universe), !params.IsSpecificConfigOptional)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if path != "" {
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return ucerr.Errorf("no config files found for %s in %v", service, serviceDirs)
	}
	uclog.Infof(ctx, "Loading %v for %+v config from: %+v", service, params, paths)
	for _, p := range paths {
		if err := LoadAndDecodeFromPath(p, cfg, params.AllowUnknownFields); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return ucerr.Wrap(cfg.Validate())
}

func getConfigFilePath(baseDirs []string, name string, mustExists bool) (string, error) {
	lookedUpPaths := make([]string, 0, len(baseDirs))
	for _, baseDir := range baseDirs {
		path := filepath.Join(baseDir, fmt.Sprintf("%s.yaml", name))
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		lookedUpPaths = append(lookedUpPaths, path)
	}
	if mustExists {
		return "", ucerr.Errorf("config file %q not found in %v (looked up: %v)", name, baseDirs, lookedUpPaths)
	}
	return "", nil
}

// LoadAndDecodeFromPath loads a YAML file into the provided struct.
func LoadAndDecodeFromPath(path string, cfg infra.Validateable, allowUnknownFields bool) error {
	f, err := os.Open(path)
	if err != nil {
		return ucerr.Errorf("can't read configuration file '%s': %w", path, err)
	}
	defer f.Close()
	return ucerr.Wrap(LoadAndDecode(f, path, cfg, allowUnknownFields))
}

// LoadAndDecode loads a YAML from a reader into the provided struct.
func LoadAndDecode(r io.Reader, filesrc string, cfg infra.Validateable, allowUnknownFields bool) error {
	cfgBytes, err := io.ReadAll(r)
	if err != nil {
		return ucerr.Wrap(err)
	}
	opts := make([]yaml.JSONOpt, 0, 1)
	if !allowUnknownFields {
		// error on unused keys if YAML file contains fields that don't map into structs
		opts = append(opts, yaml.DisallowUnknownFields)
	}

	// if Decode returns and io.EOF error, that means the file is empty, and we should just ignore it
	// note that a missing file (eg typo) would fail at os.Open() above, so this should be safe
	if err := yaml.Unmarshal(cfgBytes, cfg, opts...); err != nil && !errors.Is(err, io.EOF) {
		return ucerr.Errorf("can't unmarshal %q into a %T: %w", filesrc, cfg, err)
	}
	return nil
}
