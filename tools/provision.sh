#!/usr/bin/env bash

# Because this connects to the prod/staging databases, we need to be able to
# resolve DB credentials via AWS Secrets Manager. That in turn uses our
# AWS creds from the environment (because that's how it bootstraps in AWS),
# and so we need to ensure that our local env creds are valid. Because of our
# MFA policy, that means we need to MFA-auth locally if we haven't already.
# This is done by calling tools/ensure-aws-auth.sh

# TODO: should we save these creds in a local creds/profile file for slightly
#   nicer cross-shell behavior?

set -euo pipefail
IFS=$'\n\t'

UCENV="${1:-}"

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: tools/provision.sh <environment>"
  echo "  environment: dev | prod | staging | debug"
}

source tools/check-env.sh
validate_environment "$UCENV" print_usage

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

TMPFILE=$(mktemp /tmp/provision."$UCENV".XXXXXX)

echo "connecting to $UCENV for provisioning"
UC_UNIVERSE=$UCENV bin/provision --logfile="$TMPFILE" provision company "config/provisioning/$UCENV/company.json"
UC_UNIVERSE=$UCENV bin/provision --logfile="$TMPFILE" provision tenant "config/provisioning/$UCENV/tenant_console.json"

# Re-provision all existing companies/tenants
UC_UNIVERSE=$UCENV bin/provision --logfile="$TMPFILE" provision company all
UC_UNIVERSE=$UCENV bin/provision --logfile="$TMPFILE" provision tenant all
UC_UNIVERSE=$UCENV bin/provision --logfile="$TMPFILE" provision events all
