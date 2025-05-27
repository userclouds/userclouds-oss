#!/usr/bin/env bash

set -euo pipefail

STATUS=$(brew services list | grep postgresql@14 | awk '{print $2}')

if [ "$STATUS" != "started" ]; then
  echo "Starting Postgres..."
  brew services restart postgresql@14 || true

  for _ in {0..4}; do
    set +e
    psql postgres -c "SELECT NOW();" &>/dev/null
    RES=$?
    set -e

    [ $RES -eq 0 ] && echo "Connected to Postgres..." && exit 0

    sleep 1
  done

  # if we got this far, we failed to connect after 5 seconds
  exit 1
fi
