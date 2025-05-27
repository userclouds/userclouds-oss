#!/bin/bash
set -euo pipefail

PROMETHEUS_OPERATOR_VERSION=v0.82.2
TARGET_PATH=./terraform/modules/aws/eks-cluster/software/crds/prometheus-operator
BASE_URL=https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${PROMETHEUS_OPERATOR_VERSION}/example/prometheus-operator-crd
declare -a CRDS=(
  "alertmanagerconfigs"
  "alertmanagers"
  "podmonitors"
  "probes"
  "prometheusagents"
  "prometheuses"
  "prometheusrules"
  "scrapeconfigs"
  "servicemonitors"
  "thanosrulers"
)

for crd in "${CRDS[@]}"; do
  crd_file="monitoring.coreos.com_${crd}.yaml"
  echo "Downloading ${crd_file}..."
  if ! curl -f -s -S -o "${TARGET_PATH}/${crd_file}" "${BASE_URL}/${crd_file}"; then
    echo "Error downloading ${crd_file}" >&2
    exit 1
  fi
done
echo "All CRDs downloaded successfully!"
