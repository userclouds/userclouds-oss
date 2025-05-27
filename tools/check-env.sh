#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

function validate_environment() {
  local UCENV="$1"

  if [ "$UCENV" != "dev" ] && [ "$UCENV" != "prod" ] && [ "$UCENV" != "staging" ] && [ "$UCENV" != "debug" ]; then
    echo "invalid environment '$UCENV': use dev, prod, staging, debug"
    $2
    exit 1
  fi
}

function cloud_environment() {
  local UCENV="$1"

  if [ "$UCENV" == "prod" ] || [ "$UCENV" == "staging" ] || [ "$UCENV" == "debug" ]; then
    echo 0
    exit 0
  fi

  echo 1
}
