#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

function print_usage() {
  echo "usage: $0 <sdk_version> <tenant_url> <client_id> <client_secret> <tenant_id>"
}

SDK_VERSION="${1:-}"
USERCLOUDS_TENANT_URL="${2:-}"
USERCLOUDS_CLIENT_ID="${3:-}"
USERCLOUDS_CLIENT_SECRET="${4:-}"
TENANT_ID="${5:-}"

# Deprecated, but keeping those around for usage in older SDK versions
TENANT_URL="$USERCLOUDS_TENANT_URL"
CLIENT_ID="$USERCLOUDS_CLIENT_ID"
CLIENT_SECRET="$USERCLOUDS_CLIENT_SECRET"

if [ -z "$TENANT_ID" ]; then
  print_usage
  exit 1
fi

WORK_DIR=$(mktemp -d)

# check if temp dir was created
if [[ ! $WORK_DIR || ! -d $WORK_DIR ]]; then
  echo "Could not create temp dir"
  exit 1
fi

# deletes the temp directory
function cleanup {
  EXIT_CODE=$?
  if [ "${EXIT_CODE}" -ne 0 ]; then
    echo "Python SDK Test failed with exit code ${EXIT_CODE}"
  fi
  deactivate
  rm -rf "$WORK_DIR"
}

# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

pushd "$WORK_DIR" >/dev/null
python3 -m venv .venv
# shellcheck source=/dev/null
source .venv/bin/activate
pip install usercloudssdk=="$SDK_VERSION" >/dev/null 2>&1
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-python/v"$SDK_VERSION"/src/authz_sample.py >authz_sample.py 2>/dev/null
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-python/v"$SDK_VERSION"/src/tokenizer_sample.py >tokenizer_sample.py 2>/dev/null
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-python/v"$SDK_VERSION"/src/userstore_sample.py >userstore_sample.py 2>/dev/null
FINDANDREPLACE='s|^client_id =.*$|client_id = "'"$CLIENT_ID"'"|;s|^client_secret =.*$|client_secret = "'"$CLIENT_SECRET"'"|;s|^url =.*$|url = "'"$TENANT_URL"'"|;s|\t|    |g'
sed -i '' "$FINDANDREPLACE" authz_sample.py
sed -i '' "$FINDANDREPLACE" tokenizer_sample.py
sed -i '' "$FINDANDREPLACE" userstore_sample.py
# Add machine name & user name to the session name so we can tell who ran the test
SESSION="test-sdk-python.sh $(uname -n)/${USER:-$(id -un)}"
if [[ $SDK_VERSION != "1.0.8" ]]; then # authz functions only added after 1.0.8
  UC_SESSION_NAME=$SESSION python3 authz_sample.py >/dev/null
fi
UC_SESSION_NAME=$SESSION python3 tokenizer_sample.py >/dev/null
UC_SESSION_NAME=$SESSION UC_REGION=aws-us-east-1 python3 userstore_sample.py >/dev/null
popd >/dev/null
