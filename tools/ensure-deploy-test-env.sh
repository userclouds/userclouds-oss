#!/usr/bin/env bash

set -euo pipefail

# See: tools/deploy-tests.sh and cmd/ensuresecrets/main.go
variables=(UC_CONFIG_TEST_STAGING_CLIENT_SECRET UC_CONFIG_TEST_PROD_CLIENT_SECRET SDK_TEST_STAGING_CLIENT_SECRET SDK_TEST_PROD_CLIENT_SECRET UC_CONFIG_TEST_DEBUG_CLIENT_SECRET SDK_TEST_DEBUG_CLIENT_SECRET SQLSHIM_TEST_STAGING_CLIENT_SECRET SQLSHIM_TEST_PROD_CLIENT_SECRET SQLSHIM_TEST_MYSQL_PASSWORD)

error=false

for var in "${variables[@]}"; do
  if [[ -z ${!var:-} ]]; then
    echo "Environment variable $var is undefined or empty."
    error=true
  fi
done

if [[ $error == true ]]; then
  echo "One or more environment variables needed to run deploy tests are undefined."
  echo "Run 'make ensure-secrets-dev' followed by 'direnv allow' to populate them via the .envrc.private file."
  exit 1
fi
