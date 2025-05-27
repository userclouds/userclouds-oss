#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

UCENV="${1:-}"

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: tools/provision-events.sh <environment>"
  echo "  environment: dev | prod | staging | debug"
}

source tools/check-env.sh
validate_environment "$UCENV" print_usage

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

# ensure-deploy-eks is an easy way to make sure we are connected to the right VPN (needed in order to access aurora cluster
UC_UNIVERSE=$UCENV tools/ensure-deploy-eks.sh

echo "connecting to $UCENV for event provisioning"
UC_UNIVERSE=$UCENV bin/provision provision events all
