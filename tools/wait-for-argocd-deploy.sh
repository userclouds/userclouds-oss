#!/usr/bin/env bash

set -euo pipefail

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

ARGO_CTX="$(argocd_context)"

if ! argocd account get-user-info --argocd-context "${ARGO_CTX}" | grep -q "Logged In: true"; then
  ./tools/ensure-argocd-auth.sh
else
  echo "Logged in to ArgoCD server ${ARGO_CTX}"
fi

ARGOCD_APPS=$(get_argocd_apps)

echo "Waiting for app be synced + healthy status : ${ARGOCD_APPS}"

if [ -n "$ARGOCD_APPS" ]; then
  echo "Waiting for apps to sync and become healthy: ${ARGOCD_APPS}"
  # we want word splitting on ARGOCD_APPS
  # shellcheck disable=SC2086
  argocd app wait --output tree --sync --health --timeout 600 --argocd-context "${ARGO_CTX}" ${ARGOCD_APPS}
else
  echo "ERROR: No apps to sync"
  exit 1
fi
