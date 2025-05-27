#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

function print_usage() {
  echo "usage: $0 <tenant_url> <client_id> <client_secret> <tenant_id>"
}

function should_skip_test {
  if test -f "bypass_${1}_sdk_tests"; then
    echo "bypass_${1}_sdk_tests exists, skipping running the ${1} SDK tests..."
    rm "bypass_${1}_sdk_tests"
    return 0
  else
    return 1
  fi
}

TENANT_URL="${1:-}"
CLIENT_ID="${2:-}"
CLIENT_SECRET="${3:-}"
TENANT_ID="${4:-}"

if [ -z "$TENANT_ID" ]; then
  print_usage
  exit 1
fi

TEMP_DIR=$(mktemp -d)
pushd "$TEMP_DIR" >/dev/null
IFS=$' '
read -r -a GO_SDK_VERS <<<"$(go list -m -versions userclouds.com | sed -e 's/^userclouds.com //')"
IFS=$'\n\t'
popd >/dev/null
rm -rf "$TEMP_DIR"
if [[ ! ${GO_SDK_VERS[*]} ]]; then
  echo "No Go SDK versions found"
  exit 1
fi

if should_skip_test go; then
  : # No-op, test is skipped
else
  go version
  for GO_SDK_VER in "${GO_SDK_VERS[@]}"; do
    echo "Testing Go SDK version $GO_SDK_VER"
    tools/test-sdk-golang.sh "$GO_SDK_VER" "$TENANT_URL" "$CLIENT_ID" "$CLIENT_SECRET" "$TENANT_ID"
  done
fi

mapfile -t TYPESCRIPT_SDK_VERS < <(npm view @userclouds/sdk-typescript versions --json 2>/dev/null | grep '"' | sed -e 's/,//;s/ //g;s/"//g')
if [[ ! ${TYPESCRIPT_SDK_VERS[*]} ]]; then
  echo "No TypeScript SDK versions found"
  exit 1
fi

nodejs_ver=$(node --version)
if should_skip_test typescript; then
  : # No-op, test is skipped
else
  for TYPESCRIPT_SDK_VER in "${TYPESCRIPT_SDK_VERS[@]}"; do
    DEPRECATED=$(npm view @userclouds/sdk-typescript@"$TYPESCRIPT_SDK_VER" deprecated)
    if [ "$DEPRECATED" ]; then
      continue
    fi
    echo "Testing TypeScript SDK version $TYPESCRIPT_SDK_VER using NodeJS ${nodejs_ver}"
    tools/test-sdk-typescript.sh "$TYPESCRIPT_SDK_VER" "$TENANT_URL" "$CLIENT_ID" "$CLIENT_SECRET" "$TENANT_ID"
  done
fi

IFS=$' '
read -r -a PYTHON_SDK_VERS <<<"$(pip3 index versions usercloudssdk 2>/dev/null | grep "^Available versions:" | sed -e 's/^Available versions: //;s/,//g')"
IFS=$'\n\t'
if [[ ! ${PYTHON_SDK_VERS[*]} ]]; then
  echo "No Python SDK versions found"
  exit 1
fi

python_ver=$(python3 --version)
MIN_PYTHON_SDK_VERSION="1.8.0"
if should_skip_test python; then
  : # No-op, test is skipped
else
  for PYTHON_SDK_VER in "${PYTHON_SDK_VERS[@]}"; do
    if [[ $(echo -e "$PYTHON_SDK_VER\n$MIN_PYTHON_SDK_VERSION" | sort -V | head -n 1) != "$MIN_PYTHON_SDK_VERSION" ]]; then
      echo "Skip testing Python SDK version $PYTHON_SDK_VER because it is less than $MIN_PYTHON_SDK_VERSION"
      continue
    fi
    echo "Testing Python SDK version $PYTHON_SDK_VER using ${python_ver}"
    tools/test-sdk-python.sh "$PYTHON_SDK_VER" "$TENANT_URL" "$CLIENT_ID" "$CLIENT_SECRET" "$TENANT_ID"
  done
fi
