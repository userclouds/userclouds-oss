#!/usr/bin/env bash

set -euo pipefail

# Validate that we are running on Linux
if [[ "$(uname)" != "Linux" ]]; then
  echo "Error: This script must be run on Linux." >&2
  exit 1
fi

# If BIN_ARCHITECTURE is not defined, set it to amd64
: "${BIN_ARCHITECTURE:=amd64}"
output_dir="bin/linux-${BIN_ARCHITECTURE}"
rm -rf "${output_dir}"
mkdir -p "${output_dir}"

source ./tools/packaging/helpers.sh

echo "Building ${#} binaries in linux ${BIN_ARCHITECTURE}: $*"

for binary in "$@"; do
  local_binary_name="${output_dir}/${binary}"
  package="$(get_package "$binary")"
  CGO_ENABLED=1 GOOS=linux GOARCH=${BIN_ARCHITECTURE} go build -o "${local_binary_name}" \
    -ldflags \
    "-X userclouds.com/infra/service.buildHash=$(git rev-parse HEAD) \
     -X userclouds.com/infra/service.buildTime=$(TZ=UTC git show -s --format=%cd --date=iso-strict-local HEAD)" \
    "${package}"
done
