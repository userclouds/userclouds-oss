package testconfig

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"userclouds.com/infra"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/kubernetes"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/yamlconfig"
)

// RunConfigTestTool tests config for the given tool across all possible universes
func RunConfigTestTool(ctx context.Context, t *testing.T, toolName string, cfg infra.Validateable, devOnly bool) {
	excludes := make([]universe.Universe, 0)
	if devOnly {
		for _, uv := range universe.AllUniverses() {
			if !uv.IsDev() {
				excludes = append(excludes, uv)
			}
		}
	} else {
		excludes = append(excludes, universe.Container, universe.OnPrem)
	}
	RunConfigTestForAll(ctx, t, toolName, cfg, false, false, false, excludes...)
}

// RunConfigTestForDB tests config for the given DB across all possible universes
func RunConfigTestForDB(ctx context.Context, t *testing.T, dbName string, cfg infra.Validateable, loadBaseEnvConfig bool) {
	RunConfigTestForAll(ctx, t, dbName, cfg, loadBaseEnvConfig, true, false)
}

// RunConfigTestForService tests config for the given service across all possible universes
func RunConfigTestForService(ctx context.Context, t *testing.T, service serviceNamespace.Service, cfg infra.Validateable, excludeUniverses ...universe.Universe) {
	validateNoCloudExclusions(t, excludeUniverses...)
	RunConfigTestForAll(ctx, t, string(service), cfg, true, false, true, excludeUniverses...)
}

func validateNoCloudExclusions(t *testing.T, excludeUniverses ...universe.Universe) {
	for _, uv := range excludeUniverses {
		assert.False(t, uv.IsCloud(), assert.Errorf("cloud universe %s should not be excluded", uv))
	}
}

func writeChartConfigFiles(t *testing.T, uv universe.Universe) string {
	files, err := getConfigMapsFromChart(uv)
	assert.NoErr(t, err)
	tempDir, err := os.MkdirTemp("", "")
	t.Cleanup(func() {
		assert.NoErr(t, os.RemoveAll(tempDir))
	})
	assert.NoErr(t, err)
	for name, content := range files {
		fp := filepath.Join(tempDir, name)
		if subdir := filepath.Dir(fp); subdir != tempDir {
			assert.NoErr(t, os.MkdirAll(subdir, 0755))
		}
		err = os.WriteFile(fp, []byte(content), 0644)
		assert.NoErr(t, err)
	}
	return tempDir
}

// RunConfigTestForAll tests config for the given config across all possible universes
func RunConfigTestForAll(ctx context.Context, t *testing.T, cfgName string, cfg infra.Validateable, loadBaseEnvConfig, allowUnknownFields, isService bool, excludeUniverses ...universe.Universe) {
	baseDirs := yamlconfig.GetBaseDirs()
	for _, uv := range universe.AllUniverses() {
		if containsUniverse(uv, excludeUniverses) || uv.IsUndefined() {
			continue
		}
		t.Setenv(universe.EnvKeyUniverse, string(uv))
		params := yamlconfig.LoadParams{
			BaseDirs:           baseDirs,
			Universe:           uv,
			LoadBaseEnvConfig:  loadBaseEnvConfig,
			AllowUnknownFields: allowUnknownFields,
		}

		if uv.IsKubernetes() {
			t.Setenv(kubernetes.EnvKubernetesNamespace, "jambalaya")
			// Order matters here. The chart configs dir must be loaded first.
			// see: public-repos/helm-charts/charts/userclouds-on-prem/templates/_helpers.tpl userclouds.envVars UC_CONFIG_DIR
			chartConfigsDir := writeChartConfigFiles(t, uv)
			params.BaseDirs = append([]string{chartConfigsDir}, baseDirs...)
		} else {
			t.Setenv(kubernetes.EnvKubernetesNamespace, "")
		}
		runConfigTestForUniverse(ctx, t, cfgName, uv, cfg, params)
	}
}

func runConfigTestForUniverse(ctx context.Context, t *testing.T, cfgName string, univ universe.Universe, cfg infra.Validateable, params yamlconfig.LoadParams) {
	// Reset cfg to its default values.
	// see: https://stackoverflow.com/a/29169727/38265
	p := reflect.ValueOf(cfg).Elem()
	p.Set(reflect.Zero(p.Type()))
	err := yamlconfig.LoadEnv(ctx, cfgName, cfg, params)
	assert.NoErr(t, err, assert.Errorf("load failed for universe %s for '%v': %v", univ, cfgName, err))
}

func containsUniverse(uv universe.Universe, universes []universe.Universe) bool {
	return slices.Contains(universes, uv)
}
