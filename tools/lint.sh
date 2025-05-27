#!/usr/bin/env bash

# Not user-facing! Run `make lint` instead.
# $1 contains a string of space-delimited shell scripts to lint

set -euo pipefail
IFS=$'\n\t'

if [[ ${BASH_VERSINFO[0]} -lt 4 ]]; then
  echo "Your bash version is ${BASH_VERSION}, which is less than the required major version (4+)."
  echo "Please follow directions in the repo README.md to ensure you have set up your environment properly."
  if [ "$(uname)" == "Darwin" ]; then
    echo "On Mac OS X, you can upgrade bash by running 'brew install bash'."
  fi
  exit 1
fi

# NB: for some reason I don't fully understand, AWS CodeBuild output logging breaks
# after we use `tee` to make these tests simpler (as in `test -z "$(gofmt -s -l . | tee /dev/stderr)"`).
# So we are explicitly capturing the output of each command, checking it for empty, and if needed,
# printing it out ourselves so that CI lint failures are visible.

# we call back into make for all of these targets so we can defer build costs unless files have changed

# for now, we're always going to lint in CI even though it's a bit slower. I think that before too long,
# we'll want to separate out different GHActions (like already done for terraform) and run them all the time,
# but also all in parallel

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q '.sh$'; then
  echo "No shell scripts changed, skipping shell linting."
else
  make lint-shell
fi

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q '.go$'; then
  echo "No Go files changed, skipping Go linting."
else
  make lint-golang
fi

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q -E '(?:.ts$|.json$|consoleui|plexui|sharedui|ui-component-lib|nodejs|typescript)'; then
  echo "No Typescript files changed, skipping frontend linting."
else
  make lint-frontend
fi

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q -E '(.py|sdk-python)'; then
  echo "No Python files changes, skipping Python linting."
else
  make lint-python
fi

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q -E '^terraform'; then
  echo "No Terraform files changes, skipping Terraform linting."
else
  make lint-terraform
fi

if [ "$UC_UNIVERSE" != "ci" ] && ! git status -s | grep -q 'schema_generated.go$'; then
  echo "No SQL schema files changed, skipping SQL linting."
else
  make lint-sql
fi
