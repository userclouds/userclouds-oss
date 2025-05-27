#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

function print_usage() {
  echo "usage: $0 <sdk_version> <tenant_url> <client_id> <client_secret> <tenant_id>"
}

SDK_VERSION="${1:-}"
TENANT_URL="${2:-}"
CLIENT_ID="${3:-}"
CLIENT_SECRET="${4:-}"
TENANT_ID="${5:-}"

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
    echo "Typescript SDK Test failed with exit code ${EXIT_CODE}"
  fi
  rm -rf "$WORK_DIR"
}

# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

pushd "$WORK_DIR" >/dev/null
{
  echo TENANT_URL="$TENANT_URL"
  echo CLIENT_ID="$CLIENT_ID"
  echo CLIENT_SECRET="$CLIENT_SECRET"
} >.env
npm install @userclouds/sdk-typescript@"$SDK_VERSION" >/dev/null 2>&1
npm install dotenv >/dev/null 2>&1
npm install @types/uuid >/dev/null 2>&1
npm install -D typescript@5.1.3 >/dev/null 2>&1
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-typescript/main/samples/authz_sample.ts >authz_sample.ts 2>/dev/null
npx ts-node authz_sample.ts
popd >/dev/null
