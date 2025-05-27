#!/usr/bin/env bash

set -euo pipefail

# using this script is not advised as the kubernetes objects will be overridden by ArgoCD.

img_tag="${1:-}"
if [ -z "$img_tag" ]; then
  echo "Error: image tag is not defined."
  exit 1
fi
if [ -z "${UC_UNIVERSE:-}" ]; then
  echo "Error: UC_UNIVERSE is not defined."
  exit 1
elif [ "${UC_UNIVERSE}" != "prod" ] && [ "${UC_UNIVERSE}" != "staging" ] && [ "${UC_UNIVERSE}" != "debug" ]; then
  echo "Error: UC_UNIVERSE is must be defined as prod,staging or debug."
  exit 1

fi

EKS_DEPLOY_REGIONS=("us-east-1" "us-west-2" "eu-west-1")

for region in "${EKS_DEPLOY_REGIONS[@]}"; do
  echo "Install UserClouds services into K8S cluster: ${UC_UNIVERSE}-${region} ${img_tag}"
  # for UC_HELM_EXTRA_ARGS we actually want word splitting
  # shellcheck disable=SC2086
  time helm upgrade --install "uc-${UC_UNIVERSE}" helm/userclouds \
    --values helm/userclouds/values-${UC_UNIVERSE}-${region}.yaml --set image.tag="$img_tag" \
    --wait --kube-context ${UC_UNIVERSE}-${region} --namespace userclouds ${UC_HELM_EXTRA_ARGS:-}
done
