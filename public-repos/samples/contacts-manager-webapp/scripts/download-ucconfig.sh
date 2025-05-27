#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

mkdir -p bin/
_UCONFIG_VERSION="0.1.12"
_ARCH=$(uname -m)
_OS="$(uname)"
_PLATFORM=$(echo "${_OS}_${_ARCH}" | tr '[:upper:]' '[:lower:]')

wget -q "https://github.com/userclouds/ucconfig/releases/download/v${_UCONFIG_VERSION}/ucconfig_${_UCONFIG_VERSION}_${_PLATFORM}.tar.gz" -O bin/ucconfig.tar.gz
tar -xf bin/ucconfig.tar.gz -C bin/ ucconfig
rm bin/ucconfig.tar.gz
