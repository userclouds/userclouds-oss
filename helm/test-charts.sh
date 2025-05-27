#!/usr/bin/env bash
set -euxo pipefail
helm version
mkdir -p .k8s-manifests
# TODO: consolidate those loops once we have us-east-1 & eu-west-1 in staging and prod
DEBUG_EKS_DEPLOY_REGIONS=("us-east-1" "us-west-2" "eu-west-1")
UVS=("debug" "staging" "prod")

# Default to verbose mode in CI, otherwise non-verbose unless specified
if [ -n "${CI:-}" ]; then
  VERBOSE=true
else
  VERBOSE=${VERBOSE:-false}
fi

for uv in "${UVS[@]}"; do
  for region in "${DEBUG_EKS_DEPLOY_REGIONS[@]}"; do
    MANIFESTS=.k8s-manifests/userclouds-${uv}-${region}.yaml
    helm template helm/userclouds --values "helm/userclouds/values-${uv}.yaml" --values "helm/userclouds/values-${uv}-${region}.yaml" --debug --set image.tag=fake >"${MANIFESTS}"
    if [ "${VERBOSE}" = true ]; then
      yq "${MANIFESTS}"
    fi
    ./tools/evaluate-k8s-manifests.sh "${MANIFESTS}"
  done
done

MANIFESTS=.k8s-manifests/userclouds-on-prem.yaml
helm template public-repos/helm-charts/charts/userclouds-on-prem --values helm/userclouds-on-prem/values_on_prem_userclouds_io.yaml --set image.tag=fake --debug --namespace uc-on-prem >"${MANIFESTS}"
if [ "${VERBOSE}" = true ]; then
  yq "${MANIFESTS}"
fi
./tools/evaluate-k8s-manifests.sh "${MANIFESTS}"
# helm lint helm/userclouds --values helm/userclouds/values_debug.yaml
