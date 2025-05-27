#!/usr/bin/env bash
set -euo pipefail

OS="$(uname)"
if [[ ${OS} == "Linux" ]]; then
  # Call the Linux-specific build script
  "$(dirname "$0")/build-go-binary-linux.sh" "$@"
elif [[ ${OS} == "Darwin" ]]; then
  # Call the macOS-specific build script
  "$(dirname "$0")/build-go-binary-mac.sh" "$@"
else
  echo "Unsupported OS ${OS} for building go binary"
  exit 1
fi
