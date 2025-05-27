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

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: tools/db-migrate.sh <environment>"
  echo "  environment: dev | prod | staging | debug"
}

UCENV="${1:-}"
source tools/check-env.sh
validate_environment "$UCENV" print_usage

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

TMPFILE=$(mktemp /tmp/migrate."$UCENV".XXXXXX)
FLAGS="${2:-}"

# ensure-deploy-eks is an easy way to make sure we are connected to the right VPN (needed in order to access aurora cluster)
UC_UNIVERSE=$UCENV tools/ensure-deploy-eks.sh

# This order is intentional; rootdb should always be first
# as it may change users, permissions, or create new DBs.
# We bypass shellcheck here because quoting FLAGS means we get
# two positional args when FLAGS is empty
# TODO: should we use another env var instead to manage this? Neither feels great.
# shellcheck disable=SC2086
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --logfile="$TMPFILE" $FLAGS rootdb
# shellcheck disable=SC2086
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --logfile="$TMPFILE" $FLAGS rootdbstatus
# Console (which should be renamed companyconfig) should be before
# tenant DBs because it contains the list of all tenants.
# shellcheck disable=SC2086
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --logfile="$TMPFILE" $FLAGS companyconfig
# shellcheck disable=SC2086
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --logfile="$TMPFILE" $FLAGS tenantdb
# shellcheck disable=SC2086
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --logfile="$TMPFILE" $FLAGS status
