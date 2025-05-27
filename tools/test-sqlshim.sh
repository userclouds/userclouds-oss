#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

function print_usage() {
  echo "usage: $0 <environment>"
  echo "  <environment> - staging | prod"
}

ENVIRONMENT="${1:-}"

if [ -z "$ENVIRONMENT" ]; then
  print_usage
  exit 1
fi

tools/ensure-deploy-test-env.sh

if [ "$ENVIRONMENT" == "staging" ]; then
  CLIENT_ID="24850b3897393491f18965e4f812de2a"
  CLIENT_SECRET="$SQLSHIM_TEST_STAGING_CLIENT_SECRET"
  TENANT_URL="https://stagingtests-stagingsqlshimtest.tenant.staging.userclouds.com"
  # See: helm/userclouds/values-staging-us-west-2.yaml dbproxy.ingress.domain
  PROXY_HOST="dbproxy.staging.userclouds.com"
  PROXY_PORT="3309"
elif [ "$ENVIRONMENT" == "prod" ]; then
  CLIENT_ID="d6e1937abcf1a1acc044b08769f4d8c0"
  CLIENT_SECRET="$SQLSHIM_TEST_PROD_CLIENT_SECRET"
  TENANT_URL="https://usercloudstests-sqlshimtest.tenant.userclouds.com"
  # See: helm/userclouds/values-prod-us-west-2.yaml dbproxy.ingress.domain
  PROXY_HOST="dbproxy.userclouds.com"
  PROXY_PORT="3308"
else
  print_usage
  exit 1
fi

# shellcheck disable=SC2317 # Don't warn about unreachable commands in this function
function cleanup {
  EXIT_CODE=$?
  if [ "${EXIT_CODE}" -ne 0 ]; then
    echo "SQL Shim Tests failed with exit code ${EXIT_CODE}"
  fi
  rm bin/ucconfig
  rm output.yaml
}

# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

make bin/auditlogview bin/ucconfig

set +e
bin/ucconfig apply --client-id="$CLIENT_ID" --client-secret="$CLIENT_SECRET" --tenant-url="$TENANT_URL" --auto-approve tools/sqlshim-testfiles/clean.yaml
set -e

ACCESSOR_NAME="custom_accessor_$RANDOM"
mysql -A -h "$PROXY_HOST" -P "$PROXY_PORT" -u admin -p"$SQLSHIM_TEST_MYSQL_PASSWORD" test -e "/*accessor_name=$ACCESSOR_NAME refresh_schemas=true*/SELECT * from users"

bin/ucconfig gen-manifest --client-id="$CLIENT_ID" --client-secret="$CLIENT_SECRET" --tenant-url="$TENANT_URL" output.yaml

set +e
NUM_COLUMNS_CREATED=$(diff output.yaml tools/sqlshim-testfiles/clean.yaml | grep -c "uc_terraform_type: userstore_column")
NUM_ACCESSORS_CREATED=$(diff output.yaml tools/sqlshim-testfiles/clean.yaml | grep -c "uc_terraform_type: userstore_accessor")
CUSTOM_ACCESSOR_SEARCH=$(bin/auditlogview "$TENANT_URL" "$CLIENT_ID" "$CLIENT_SECRET" list 5 | grep "$ACCESSOR_NAME")
set -e

if [ -n "$CUSTOM_ACCESSOR_SEARCH" ]; then
  CUSTOM_ACCESSOR_FOUND=1
else
  CUSTOM_ACCESSOR_FOUND=0
fi

set +e
bin/ucconfig apply --client-id="$CLIENT_ID" --client-secret="$CLIENT_SECRET" --tenant-url="$TENANT_URL" --auto-approve tools/sqlshim-testfiles/clean.yaml
set -e

echo ""
echo "-----------------------------"
echo "Number of columns created: $NUM_COLUMNS_CREATED"
echo "Number of accessors created: $NUM_ACCESSORS_CREATED"

if [ "$NUM_COLUMNS_CREATED" -gt 0 ] && [ "$NUM_ACCESSORS_CREATED" -gt 0 ] && [ "$CUSTOM_ACCESSOR_FOUND" -eq 1 ]; then
  echo "Test passed"
  exit 0
else
  echo "SQL Shim tests failed:"
  echo "  - Number of columns created: $NUM_COLUMNS_CREATED (should be > 0)"
  echo "  - Number of accessors created: $NUM_ACCESSORS_CREATED (should be > 0)"
  echo "  - Custom accessor found: $CUSTOM_ACCESSOR_FOUND (should be 1)"
  exit 1
fi
