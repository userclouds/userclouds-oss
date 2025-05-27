#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

set +e
keys=$(git grep "PRIMARY KEY (.*id.*deleted.*)" ./**/schema_generated.go)
set -e

fail=0
if [[ -n $keys ]]; then
  fail=1
  echo "deleted should always come before id in primary keys for performance"
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
