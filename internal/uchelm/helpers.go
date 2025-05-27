package uchelm

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"sigs.k8s.io/yaml"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

// YamlDoc represents a YAML document as a map with string keys and any values.
type YamlDoc map[string]any

const (
	onPremChartPath     = "public-repos/helm-charts/charts/userclouds-on-prem"
	onPremValuesPath    = "helm/userclouds-on-prem/values_on_prem_userclouds_io.yaml"
	userCloudsChartPath = "helm/userclouds"
	chartRegion         = "us-west-2" // This is arbitrary, we probably want to cover more regions in the future, especially for the ConfigTests for testing
)

// RenderUCChartForUniverse renders a Helm chart for a given universe and returns the rendered manifest as a string.
func RenderUCChartForUniverse(uv universe.Universe) (string, error) {
	if uv.IsOnPrem() {
		return renderUserCloudsHelmChart(onPremChartPath, onPremValuesPath)
	} else if uv.IsCloud() {
		return renderUserCloudsHelmChart(userCloudsChartPath,
			filepath.Join(userCloudsChartPath, fmt.Sprintf("values-%s.yaml", uv)),
			filepath.Join(userCloudsChartPath, fmt.Sprintf("values-%s-%s.yaml", uv, chartRegion)),
		)
	}
	return "", ucerr.Errorf("universe %s is not supported for RenderUCChartForUniverse", uv)
}

// renderUserCloudsHelmChart renders a Helm chart and returns the rendered manifest as a string.
func renderUserCloudsHelmChart(chartPath string, valueFilesPaths ...string) (string, error) {
	ns := "userclouds"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(cli.New().RESTClientGetter(), ns, "", log.Printf); err != nil {
		return "", ucerr.Wrap(err)
	}
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	// Set up Helm install options
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = "uc-testing"
	client.Namespace = ns
	client.ClientOnly = true
	valueOpts := &values.Options{
		ValueFiles:   valueFilesPaths,
		StringValues: []string{"image.tag=fake"},
	}
	vs, err := valueOpts.MergeValues(getter.Providers{})
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	release, err := client.Run(chart, vs)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return release.Manifest, nil
}

// GetFirstManifest parses a YAML document and unmarshals the first occurrence of a specified kind into the provided manifest.
func GetFirstManifest(yamlData, apiKind string, manifest any) error {
	rawManifests, err := GetManifests(yamlData, apiKind)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(yaml.Unmarshal(rawManifests[0], manifest))
}

// GetManifests parses a YAML document and returns all occurrences of a specified kind as a slice of byte slices.
func GetManifests(yamlData, apiKind string) ([][]byte, error) {
	manifests := make([][]byte, 0)
	for rawDoc := range strings.SplitSeq(yamlData, "\n---\n") {
		bytesDoc := []byte(rawDoc)
		var yamlDoc map[string]any
		if err := yaml.Unmarshal(bytesDoc, &yamlDoc); err != nil {
			return nil, ucerr.Wrap(err)
		}
		if kind, ok := yamlDoc["kind"].(string); !ok || kind != apiKind {
			continue
		}
		manifests = append(manifests, bytesDoc)

	}
	if len(manifests) == 0 {
		return nil, ucerr.Errorf("no %s found in YAML data", apiKind)
	}
	return manifests, nil
}
