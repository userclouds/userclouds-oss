#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# shellcheck disable=SC2317  # Don't warn about unreachable commands in this function
function trap_unstash() {
  unstash
  exit 1
}
function unstash() {
  if [ "$stashed" != "No local changes to save" ]; then
    git stash pop --index -q || echo -e "\nAutomatic git stash pop failed, check for conflicts"
  fi
}

if test -f bypass_lint; then
  echo "bypass_lint exists, skipping pre-commit lint"
  rm bypass_lint
else
  echo "Running make lint before committing"
  stashed=$(git stash --keep-index -u)
  trap trap_unstash INT

  set +e
  make lint
  RESULT=$?
  set -e

  unstash
  # OK to exit here because bypass_tests must not have existed
  [ $RESULT -ne 0 ] && exit 1
fi

exit 0
