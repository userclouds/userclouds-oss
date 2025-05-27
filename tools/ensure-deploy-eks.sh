#!/usr/bin/env bash

set -euo pipefail

source ./tools/eks-helpers.sh

if [ -z "${UC_UNIVERSE:-}" ]; then
  echo "Error: UC_UNIVERSE is not defined."
  exit 1
fi

if [ "${UC_UNIVERSE}" == "prod" ] || [ "${UC_UNIVERSE}" == "staging" ]; then
  EKS_REGIONS=("us-west-2")
elif [ "${UC_UNIVERSE}" == "debug" ]; then
  EKS_REGIONS=("us-east-1" "us-west-2" "eu-west-1")
else
  exit 0
fi

check_command helm
check_command kubectl

# Loop through each cluster and check
for region in "${EKS_REGIONS[@]}"; do
  check_cluster "${UC_UNIVERSE}-${region}"
done

echo "Helm and kubectl are installed, and all specified Kubernetes clusters are configured and accessible."
