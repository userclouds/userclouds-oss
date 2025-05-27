package testconfig

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/uchelm"
)

func getConfigMapsFromChart(uv universe.Universe) (map[string]string, error) {
	releaseManifest, err := uchelm.RenderUCChartForUniverse(uv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	configMaps, err := getConfigMaps(releaseManifest)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	configFiles := make(map[string]string)
	for _, cm := range configMaps {
		for key, val := range cm.Data {
			filename := getConfigFileNameFromKey(key, uv)
			configFiles[filename] = val
		}
	}
	return configFiles, nil
}
func getConfigFileNameFromKey(key string, uv universe.Universe) string {
	if key == "base" {
		// base env config, i.e. base_onprem.yaml, base_debug.yaml, etc.
		return fmt.Sprintf("%s_%v.yaml", key, uv)
	}
	if strings.HasSuffix(key, "_base") {
		// base config for a given service (common for all envs/universes) i.e. worker/base.yaml, authz/base.yaml, etc.
		serviceName := strings.TrimSuffix(key, "_base")
		return fmt.Sprintf("%s/base.yaml", serviceName)
	}
	// service specific config,authz/debug.yaml, worker/onprem.yaml, etc.
	return fmt.Sprintf("%s/%v.yaml", key, uv)
}

func getConfigMaps(yamlData string) ([]corev1.ConfigMap, error) {
	rawConfigMapManifests, err := uchelm.GetManifests(yamlData, "ConfigMap")
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	configMaps := make([]corev1.ConfigMap, 0, len(rawConfigMapManifests))
	for _, rawConfigMap := range rawConfigMapManifests {
		var configMap corev1.ConfigMap
		if err := ucerr.Wrap(yaml.Unmarshal(rawConfigMap, &configMap)); err != nil {
			return nil, ucerr.Wrap(err)
		}
		configMaps = append(configMaps, configMap)
	}
	return configMaps, nil
}
