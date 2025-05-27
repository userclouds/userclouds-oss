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
    echo "Go SDK Test failed with exit code ${EXIT_CODE}"
  fi
  rm -rf "$WORK_DIR"
}

# register the cleanup function to be called on the EXIT signal
trap cleanup EXIT

pushd "$WORK_DIR" >/dev/null
go mod init temp >/dev/null 2>&1 && go get userclouds.com@"$SDK_VERSION" >/dev/null 2>&1
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-golang/"$SDK_VERSION"/samples/basic/main.go >main.go 2>/dev/null
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-golang/"$SDK_VERSION"/samples/basic/filemanager.go >filemanager.go 2>/dev/null
curl --retry 5 --fail https://raw.githubusercontent.com/userclouds/sdk-golang/"$SDK_VERSION"/samples/basic/authzhelper.go >authzhelper.go 2>/dev/null
{
  echo USERCLOUDS_TENANT_URL="$TENANT_URL"
  echo USERCLOUDS_TENANT_ID="$TENANT_ID"
  echo USERCLOUDS_CLIENT_ID="$CLIENT_ID"
  echo USERCLOUDS_CLIENT_SECRET="$CLIENT_SECRET"
  # For backward compatibility we also define the env variables used in older SDK versions (1.0.0 and older)
  # Once we deprecate those versions we can remove them
  echo TENANT_URL="$TENANT_URL"
  echo TENANT_ID="$TENANT_ID"
  echo CLIENT_ID="$CLIENT_ID"
  echo CLIENT_SECRET="$CLIENT_SECRET"
} >.env
go get ./...
go run ./*.go >/dev/null

rm ./*.go

set +e
curl -f https://raw.githubusercontent.com/userclouds/sdk-golang/"$SDK_VERSION"/samples/userstore/main.go >main.go 2>/dev/null
rc=$?
set -e

if [[ $rc == "0" ]]; then
  go get ./...
  go run ./*.go >/dev/null
fi
popd >/dev/null
