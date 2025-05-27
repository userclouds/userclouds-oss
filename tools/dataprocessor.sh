#!/usr/bin/env bash

# Because this connects to the prod/staging databases, we need to be able to
# resolve DB credentials via AWS Secrets Manager. That in turn uses our
# AWS creds from the environment (because that's how it bootstraps in AWS),
# and so we need to ensure that our local env creds are valid. Because of our
# MFA policy, that means we need to MFA-auth locally if we haven't already.
#  This is done by calling tools/ensure-aws-auth.sh
#
# TODO: should we save these creds in a local creds/profile file for slightly
#   nicer cross-shell behavior?

set -euo pipefail
IFS=$'\n\t'

UCENV="${1:-}"
if [ "$UCENV" == "dev" ] || [ "$UCENV" == "prod" ] || [ "$UCENV" == "staging" ]; then
  echo "connecting to $UCENV database to apply the command"
else
  echo "usage: tools/dataprocessor.sh <environment>"
  echo "  environment: dev | prod | staging"
  exit 1
fi

COMMAND="${2:-}"

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

echo "connecting to $UCENV to apply command"
UC_UNIVERSE=$UCENV bin/dataprocessor "$COMMAND"
