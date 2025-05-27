#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

UCENV="$1"
if [ "$UCENV" == "prod" ]; then
  echo "deploy-status.sh: is not compatible with prod, use tools/deploy-with-argocd.sh for prod env"
  exit 1
fi

UC_UNIVERSE="$UCENV" ./tools/ensure-argocd-auth.sh
# For debug and staging, we wait for the GHA workflow .github/workflows/build-uc-containers.yml to finish building images and push them to ECR
UC_UNIVERSE="$UCENV" tools/wait-for-uc-image-for-eks.sh
# Wait for ArgoCD to finish rolling out the deployment before running the deploy tests
UC_UNIVERSE=$UCENV ./tools/wait-for-argocd-deploy.sh
