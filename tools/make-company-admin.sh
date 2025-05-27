#!/usr/bin/env bash

# This is a helper script to make a user an admin of an company,, which is a required
# step when setting up a new environment if you want your account to be able to
# see or modify any companies. However, since you need a user ID to do this
# and we don't yet have a stable "root" user provisioned with a fixed name/ID,
# you need to get your user ID somehow. The easiest way right now is to get it
# from the upper right profile widget in Console (there's a handy 'copy' feature).

set -euo pipefail
IFS=$'\n\t'

function print_usage() {
  echo "usage: tools/make-company-admin.sh <environment> <userid> [COMPANYID]"
  echo "  environment: dev | prod | staging | debug"
  echo "  userid: username of the user to make an admin on company (TODO: make it a UUID)"
  echo "  COMPANYID: optional, ID of company (if omitted, then userclouds company)"
}

UCENV="${1:-}"

source tools/check-env.sh
validate_environment "$UCENV" print_usage

USERID="${2:-}"
if [ -z "$USERID" ]; then
  print_usage
  exit 1
fi

COMPANYID="${3:-}"
if [ -z "$COMPANYID" ]; then
  COMPANYFILE=config/provisioning/$UCENV/company.json
  COMPANYID=$(jq -r .company.id "$COMPANYFILE")
fi

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

make bin/provision # ensure this is up to date
echo "setting user '$USERID' as admin of company '$COMPANYID' for universe '$UCENV'..."
# region is not really impprtant it but the config loader requires it
UC_REGION=aws-us-west-2 UC_UNIVERSE=$UCENV bin/provision --owner "$USERID" setowner company "$COMPANYID"
