#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ -z "${USERCLOUDS_TENANT_URL-}" ] || [ -z "${USERCLOUDS_CLIENT_ID-}" ] || [ -z "${USERCLOUDS_CLIENT_SECRET-}" ]; then
  echo "Env variables required to access UserClouds are not defined"
  echo "Add them to an .envrc file and try again (see envrc.template for reference)"
  exit 1
fi
