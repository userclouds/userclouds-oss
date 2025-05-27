#!/usr/bin/env bash

set -euo pipefail

echo "Generate route file"
go run cmd/exportrouting/main.go docker/userclouds-headless/routing.yaml idp plex authz

echo "Build binaries for container"
BIN_ARCHITECTURE=amd64 time ./tools/packaging/build-go-binary-linux.sh migrate provision idp plex authz containerrunner ucconfig
CXX=aarch64-linux-gnu-g++ CC=aarch64-linux-gnu-gcc BIN_ARCHITECTURE=arm64 time ./tools/packaging/build-go-binary-linux.sh migrate provision idp plex authz containerrunner ucconfig
ls -lah bin/linux-amd64
ls -lah bin/linux-arm64
