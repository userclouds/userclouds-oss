#!/usr/bin/env bash

set -euo pipefail

source ./tools/argocd-helpers.sh
# We only need this script for prod where we don't auto-sync the UserClouds services ArgoCD apps.
# The reset of the script is agnostic to the environment, so it can be easily reused in other environments, if needed.
if [ "${UC_UNIVERSE:-}" != "prod" ]; then
  echo "Error: UC_UNIVERSE is must be defined as prod. we only do manual sync of app in prod. other envs are have automated sync"
  exit 1
fi

ARGO_CTX="$(argocd_context)"

# Validate UC_UNIVERSE is not empty after substitution
if [ -z "${UC_UNIVERSE}" ]; then
  echo "Error: UC_UNIVERSE is empty"
  exit 1
fi

if ! argocd account get-user-info --argocd-context "${ARGO_CTX}" | grep -q "Logged In: true"; then
  ./tools/ensure-argocd-auth.sh
else
  echo "Logged in to ArgoCD server ${ARGO_CTX}"
fi

ARGOCD_APPS=$(argocd app list --project "uc-${UC_UNIVERSE}" --argocd-context "${ARGO_CTX}" -o json | jq -r '.[] | select(.spec.syncPolicy.automated == null) | .metadata.name' | sort | tr '\n' ' ')

echo "Apps to be synced: ${ARGOCD_APPS}"

if [ -n "$ARGOCD_APPS" ]; then
  for app in ${ARGOCD_APPS}; do
    echo "Syncing app: ${app}"
    argocd app sync --argocd-context "${ARGO_CTX}" "${app}"
    echo "Waiting for healthy status: ${app}"
    argocd app wait --output tree --sync --health --timeout 600 --argocd-context "${ARGO_CTX}" "${app}"
  done
else
  echo "ERROR: No apps to sync"
  exit 1
fi
