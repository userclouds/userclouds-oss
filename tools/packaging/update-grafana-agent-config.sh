#!/usr/bin/env bash

set -euo pipefail
EB_ENV_TYPE=$1
DEFAULT_PROCESS=$2

sed -i -E "s~\{ ENV \}~$ENV~g" .ebextensions/grafana-agent.config
sed -i -E "s~\{ REGION \}~$REGION~g" .ebextensions/grafana-agent.config
sed -i -E "s~\{ EB_ENV_TYPE \}~$EB_ENV_TYPE~g" .ebextensions/grafana-agent.config
sed -i -E "s~\{ DEFAULT_PROCESS \}~$DEFAULT_PROCESS~g" .ebextensions/grafana-agent.config
