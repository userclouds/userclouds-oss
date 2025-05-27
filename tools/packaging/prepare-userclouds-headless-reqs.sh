#!/usr/bin/env bash

set -euo pipefail

echo "Build binaries for container"
./tools/packaging/build-go-binary.sh migrate provision idp plex authz ucconfig

echo "Generate route file"
go run cmd/exportrouting/main.go docker/userclouds-headless/routing.yaml idp plex authz
