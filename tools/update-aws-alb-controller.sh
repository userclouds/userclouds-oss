#!/bin/bash
set -euo pipefail

# https://github.com/aws/eks-charts/tree/master/stable/aws-load-balancer-controller/crds
# https://github.com/aws/eks-charts/releases
# AWS_ALB_CONTROLLER_HELM_CHART_RELEASE=v0.0.195
# BASE_URL=https://raw.githubusercontent.com/aws/eks-charts/refs/tags/${AWS_ALB_CONTROLLER_HELM_CHART_RELEASE}/stable/aws-load-balancer-controller/crds
# temporary use master since this repo is having issues with the release tag
BASE_URL=https://raw.githubusercontent.com/aws/eks-charts/refs/heads/master/stable/aws-load-balancer-controller/crds
TARGET_PATH=./terraform/modules/aws/eks-cluster/software/crds/aws-alb-controller
mkdir -p "${TARGET_PATH}"

crd_url="${BASE_URL}/crds.yaml"
echo "Downloading and processing ${crd_url}..."
pushd "${TARGET_PATH}" >/dev/null
if ! curl -f -s -S "${crd_url}" | yq --split-exp '.metadata.name + ".yaml"'; then
  popd >/dev/null
  echo "Error downloading or processing ${crd_url}" >&2
  exit 1
fi
popd >/dev/null

echo "All CRDs downloaded and split successfully!"
