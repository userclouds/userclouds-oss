#!/usr/bin/env bash

set -euo pipefail

tools/ensure-python-venv.sh

if [ -z "${VIRTUAL_ENV:-}" ]; then
  echo "Not already inside a Python virtual environment, activating it"
  # shellcheck disable=SC1091
  source .venv/bin/activate
fi

tools/ensure-deploy-test-env.sh
if [ "$1" == "debug" ]; then
  echo "Running automated post deploy tests on debug"
  # shellcheck disable=SC2034
  USERCLOUDS_TENANT_URL="https://usercloudsdebug-deploytests.tenant.debug.userclouds.com" \
    USERCLOUDS_CLIENT_ID="baa83fe1166497d87c3a2629021dd2e8" \
    USERCLOUDS_CLIENT_SECRET=$UC_CONFIG_TEST_DEBUG_CLIENT_SECRET \
    tools/test-last-released-ucconfig.py

  tools/test-all-available-sdks.sh \
    https://usercloudsdebug-deploytests.tenant.debug.userclouds.com \
    "baa83fe1166497d87c3a2629021dd2e8" \
    "${SDK_TEST_DEBUG_CLIENT_SECRET}" \
    "7041608a-00b9-4cd0-9951-e3e0ecdbc1d1"

elif [ "$1" == "staging" ]; then
  echo "Running automated post deploy tests on staging"
  # shellcheck disable=SC2034
  USERCLOUDS_TENANT_URL="https://stagingtests-stagingucconfigtests.tenant.staging.userclouds.com" \
    USERCLOUDS_CLIENT_ID="e57c48f410623e491119221b5a652274" \
    USERCLOUDS_CLIENT_SECRET=$UC_CONFIG_TEST_STAGING_CLIENT_SECRET \
    tools/test-last-released-ucconfig.py

  tools/test-all-available-sdks.sh \
    https://stagingtests-stagingdeployapitests.tenant.staging.userclouds.com \
    "711693b9a9d1914a2f484f4c8dee2dd6" \
    "${SDK_TEST_STAGING_CLIENT_SECRET}" \
    "7abc087b-17e8-441d-8e80-88e2088f108f"

  tools/test-sqlshim.sh staging

elif [ "$1" == "prod" ]; then
  echo "Running automated post deploy tests on prod"
  # shellcheck disable=SC2034
  USERCLOUDS_TENANT_URL="https://usercloudstests-proddeploymentucconfig.tenant.userclouds.com" \
    USERCLOUDS_CLIENT_ID="cfafd76f54374f8cc3286cf307bb37d9" \
    USERCLOUDS_CLIENT_SECRET=$UC_CONFIG_TEST_PROD_CLIENT_SECRET ./tools/test-last-released-ucconfig.py

  tools/test-all-available-sdks.sh \
    https://usercloudstests-deploy-sdk-tests.tenant.userclouds.com \
    ecd7ad5874c5a82a31d5335874e540cb "${SDK_TEST_PROD_CLIENT_SECRET}" \
    26c523d9-b23b-452a-a795-d99445196ea9

  tools/test-sqlshim.sh prod

else
  echo "usage: tools/deploy-tests.sh environment"
  exit 1
fi
