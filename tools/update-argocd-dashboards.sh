#!/bin/bash
set -euo pipefail

# https://github.com/adinhodovic/argo-cd-mixin/tree/main/dashboards_out
TARGET_PATH=./terraform/modules/userclouds/grafana-dashboards/dashboards/argocd
BASE_URL="https://raw.githubusercontent.com/adinhodovic/argo-cd-mixin/refs/heads/main/dashboards_out/"
declare -a DASHBOARDS=(
  "application-overview"
  "notifications-overview"
  "operational-overview"
)

for ds in "${DASHBOARDS[@]}"; do
  ds_file="${ds}.json"
  echo "Downloading ${ds_file}..."
  if ! curl --fail --silent --show-error --location --output "${TARGET_PATH}/${ds_file}" "${BASE_URL}/argo-cd-${ds_file}"; then
    echo "Error downloading ${ds_file}" >&2
    exit 1
  fi
done
echo "All Dashboards downloaded successfully!"
