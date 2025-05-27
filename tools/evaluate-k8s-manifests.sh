#!/usr/bin/env bash

set -euo pipefail

mkdir -p "${HOME}/.cache/kubeconform"
K8S_VERSION="1.31"
kubeconform -output tap -strict -debug \
  -kubernetes-version "${K8S_VERSION}.0" \
  -schema-location default \
  -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
  -cache "${HOME}/.cache/kubeconform" \
  "$@"
