#!/usr/bin/env bash

set -euo pipefail

# Keep in sync with terraform/configurations/aws/aws-regions.hcl eks_regions
EKS_REGIONS="us-west-2 us-east-1 eu-west-1"
for env in debug staging prod; do
  for region in $EKS_REGIONS; do
    if ! aws eks update-kubeconfig --profile $env --region "$region" --name "$env-$region-eks" --alias "$env-$region"; then
      echo "Failed to update kubeconfig for environment: $env, region: $region" >&2
      exit 1
    fi
  done
done
