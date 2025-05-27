#!/bin/bash
set -euo pipefail

# https://github.com/dotdc/grafana-dashboards-kubernetes/releases
DASHBOARD_VERSION=v2.7.4
TARGET_PATH=./terraform/modules/userclouds/grafana-dashboards/dashboards/kubernetes
BASE_URL="https://raw.githubusercontent.com/dotdc/grafana-dashboards-kubernetes/refs/tags/${DASHBOARD_VERSION}/dashboards/"
declare -a DASHBOARDS=(
  "k8s-addons-prometheus"
  "k8s-addons-trivy-operator"
  "k8s-system-api-server"
  "k8s-system-coredns"
  "k8s-views-global"
  "k8s-views-namespaces"
  "k8s-views-nodes"
  "k8s-views-pods"
)

for ds in "${DASHBOARDS[@]}"; do
  ds_file="${ds}.json"
  echo "Downloading ${ds_file}..."
  if ! curl --fail --silent --show-error --location --output "${TARGET_PATH}/${ds_file}" "${BASE_URL}/${ds_file}"; then
    echo "Error downloading ${ds_file}" >&2
    exit 1
  fi
done
echo "All Dashboards downloaded successfully!"
