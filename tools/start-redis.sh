#!/usr/bin/env bash

set -euo pipefail

STATUS=$(brew services list | grep redis | awk '{print $2}')

if [ "$STATUS" != "started" ]; then
  echo "Starting Redis..."
  brew services restart redis || true
fi
