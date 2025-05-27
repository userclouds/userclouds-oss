#!/usr/bin/env bash

# Not user-facing! Run `make lintfix` instead.
# $1 contains a string of space-delimited shell scripts to lint

set -euo pipefail
IFS=$'\n\t'

echo "Fixing Go formatting with gofmt..."
gofmt -s -w .

echo "Fixing Go import grouping with goimports..."
goimports -w -local userclouds.com .

echo "Fixing Go style with modernize..."
modernize -fix ./...

echo "Fixing shell formatting with shfmt..."
shellscripts=$(echo "$1" | tr " " "\n")
# this next comment specifically prevents shellcheck asking us to quote $shellscripts, which wouldn't work
# shellcheck disable=SC2086
shfmt -i 2 -s -w $shellscripts

./tools/lintfix-python.sh

# lint/prettier all projects with a package.json that are NOT 3rd party node modules
# Using -mindepth 2 to ignore the repo root package.json
# Currently samples/auth0nodejs is not part of our yarn workspaces, so we skip it.
WEB_LINT_DIRS=$(find . -name "package.json" -mindepth 2 | sed -E 's/package.json//' | grep -v "node_modules" | grep -v ".next")

# and cache IFS so we can split on newlines only to parse find's results (yay Bash)
# note that we call the binaries with a specific path since we're changing wd as we go
OLDIFS=$IFS
IFS=$'\n'
NODE_ENV="${NODE_ENV:=development}"
for DIR in $WEB_LINT_DIRS; do
  pushd "$DIR" >/dev/null
  echo "Fixing JS/TS/CSS/etc in $DIR..."
  if [[ $DIR =~ ^"./public-repos/" || $DIR =~ ^"./samples/" ]]; then
    yarn install # these dirs aren't part of our yarn workspaces, so we need to make sure their node_modules are up to date
  fi
  NODE_ENV=$NODE_ENV yarn run lintfix
  NODE_ENV=$NODE_ENV yarn run format --write
  popd >/dev/null
done
IFS=$OLDIFS
