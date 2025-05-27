#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# This is a bash helper library used by tools which need to set up AWS creds.

source tools/check-env.sh

function print_usage() {
  echo "use a valid environment, UC_UNIVERSE must be set"
}
# ensure_aws_env_vars takes 1 param, the environment/universe.
# If "prod", "staging" or "debug", then ensure AWS auth is properly configured. otherwise do nothing.

function ensure_aws_env_vars() {
  validate_environment "$UC_UNIVERSE" print_usage

  RES=$(cloud_environment "$UC_UNIVERSE")
  if [ "$RES" == 1 ]; then
    echo "In ${UC_UNIVERSE} universe, no need for AWS auth..."
    return 0
  fi

  if [[ -f "${HOME}/.aws/credentials" ]]; then
    echo "************************************************* WARNING **********************************************************"
    echo "An AWS credentials file exists under ~/.aws/credentials, It might cause issues when trying to access AWS resources."
    echo "Please consider removing it."
    echo "********************************************************************************************************************"
  fi

  if [[ -d ~/.aws/sso ]]; then
    local EXPIRY
    EXPIRY="$(aws configure get sso_start_url | xargs -I {} grep -h {} ~/.aws/sso/cache/*.json | jq -r .expiresAt || echo '')"
    if [[ -n $EXPIRY ]]; then
      AWS_TOKEN_EXPIRY=$EXPIRY
    fi
  fi

  # Check if there is an unexpired AWS CLI session.
  local AWS_AUTHED=0
  if [[ -v AWS_TOKEN_EXPIRY ]]; then
    # AWS uses ISO-8601 for expiry times, so generate our local equiv for comparison
    local NOW
    NOW=$(date -u +%Y-%m-%dT%H:%M:%S%z)

    if [[ $AWS_TOKEN_EXPIRY > $NOW ]]; then
      echo "AWS auth already valid"
      AWS_AUTHED=1
    else
      echo "AWS auth expired ($AWS_TOKEN_EXPIRY), re-authorizing..."
    fi
  else
    echo "no AWS auth env vars set, authorizing..."
  fi

  # if not, prompt for one
  if [[ $AWS_AUTHED == 0 ]]; then
    aws sso login
  fi
}

ensure_aws_env_vars
