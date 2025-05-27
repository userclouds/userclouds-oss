#!/usr/bin/env bash
set -euo pipefail

# Validate OS is MacOS
if [[ "$(uname)" != "Darwin" ]]; then
  echo "Error: This script can only be run on macOS."
  echo "Current OS: $(uname)"
  exit 1
fi

output_dir="bin/linux-amd64"
rm -rf "${output_dir}"
mkdir -p "${output_dir}"

source ./tools/packaging/helpers.sh

echo "Building ${#} binaries in mac using docker: $*"
# Use a docker volume to cache stuff between runs (go modules, etc)
cache=cache
if [[ -z $(docker volume ls -q --filter name=${cache}) ]]; then
  echo "Creating volume ${cache}"
  docker volume create ${cache}
fi

time docker buildx build --platform=linux/amd64 -t userclouds-go-build --file docker/builders/Dockerfile.go-builder .
for binary in "$@"; do
  local_binary_name="${output_dir}/${binary}"
  package="$(get_package "$binary")"
  echo "Build ${binary} (${package}) binary into ${local_binary_name} in Docker"
  time docker run --platform=linux/amd64 --rm \
    --mount type=bind,source="$(pwd)",target=/userclouds/host_repo \
    --mount type=volume,src="${cache}",dst=/userclouds/.cache/ \
    -it userclouds-go-build \
    go build -o "${local_binary_name}" \
    -ldflags \
    "-X userclouds.com/infra/service.buildHash=$(git rev-parse HEAD) \
     -X userclouds.com/infra/service.buildTime=$(TZ=UTC git show -s --format=%cd --date=iso-strict-local HEAD)" \
    "${package}"
done
