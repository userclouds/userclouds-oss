#!/usr/bin/env bash

set -euo pipefail

source ./tools/eks-helpers.sh
source ./tools/argocd-helpers.sh

if [ -z "${UC_UNIVERSE:-}" ]; then
  echo "Error: UC_UNIVERSE is not defined."
  exit 1
else
  if [ "${UC_UNIVERSE}" != "prod" ] && [ "${UC_UNIVERSE}" != "staging" ] && [ "${UC_UNIVERSE}" != "debug" ]; then
    echo "Error: UC_UNIVERSE is must be defined as prod,staging or debug."
    exit 1
  fi
fi

check_command argocd
ARGO_CTX="$(argocd_context)"

if argocd account get-user-info --argocd-context "${ARGO_CTX}" | grep -q "Logged In: true"; then
  echo "Already logged into ArgoCD server ${ARGO_CTX}"
  exit 0
fi

# this is mostly to check that VPN is connected, we don't actually need to access the cluster to auth with argocd
check_cluster "${UC_UNIVERSE}-us-west-2"

if ! argocd login "${ARGO_CTX}" --sso; then
  echo "Error: Failed to login to ArgoCD server ${ARGO_CTX}"
  echo "Please ensure your SSO credentials are correct and try again"
  exit 1
fi
echo "Successfully logged into ArgoCD server ${ARGO_CTX}"
