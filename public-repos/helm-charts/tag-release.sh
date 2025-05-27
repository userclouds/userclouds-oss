#!/usr/bin/env bash
set -euo pipefail

CHART_VERSION=$(yq e '.version' charts/userclouds-on-prem/Chart.yaml)

echo "Tagging release v${CHART_VERSION}"
git tag "v${CHART_VERSION}" && git push origin "v${CHART_VERSION}"
