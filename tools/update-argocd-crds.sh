#!/bin/bash
set -euo pipefail

# https://github.com/argoproj/argo-cd/tree/master/manifests/crds
ARGOCD_VERSION=v2.13.2
TARGET_PATH=./terraform/modules/userclouds/argocd/crds
BASE_URL=https://raw.githubusercontent.com/argoproj/argo-cd/refs/tags/${ARGOCD_VERSION}/manifests/crds
declare -a CRDS=(
  "application"
  "applicationset"
  "appproject"
)

for crd in "${CRDS[@]}"; do
  crd_file="${crd}-crd.yaml"
  echo "Downloading ${crd_file}..."
  if ! curl -f -s -S -o "${TARGET_PATH}/${crd_file}" "${BASE_URL}/${crd_file}"; then
    echo "Error downloading ${crd_file}" >&2
    exit 1
  fi
done
echo "All CRDs downloaded successfully!"
