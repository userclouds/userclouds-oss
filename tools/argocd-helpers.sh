#!/usr/bin/env bash

set -euo pipefail

source ./tools/packaging/logging.sh

function get_argocd_apps() {
  ARGO_CTX="$(argocd_context)"
  APPS=$(argocd app list --project "uc-${UC_UNIVERSE}" --argocd-context "${ARGO_CTX}" -o json | jq -r '.[].metadata.name' | tr '\n' ' ')
  echo "${APPS}"
}

function argocd_context() {
  echo "argocd.${UC_UNIVERSE}.userclouds.tools"
}
