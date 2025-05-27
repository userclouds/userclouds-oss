#!/usr/bin/env bash

set -euo pipefail

# This script should not be used normamly since the deployment is done via ArgoCD.
# Even when running it, it will deploy, but ArgoCD will override the deploy and re-deploy the previous version.
# If you need to use this for some reason, auto-sync must be disabled on the ArgoCD app.

img_tag="${1:-}"
if [ -z "$img_tag" ]; then
  echo "Error: image tag is not defined."
  exit 1
fi

echo "Install UserClouds On Prem into  debug-us-east-1 k8s cluster under uc-on-prem namespace. version: ${img_tag}"
helm upgrade --install uc-on-prem public-repos/helm-charts/charts/userclouds-on-prem \
  --values helm/userclouds-on-prem/values_on_prem_userclouds_io.yaml --set image.tag="$img_tag" \
  --debug --wait --kube-context debug-us-east-1 --namespace uc-on-prem
