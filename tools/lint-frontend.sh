#!/usr/bin/env bash

set -euo pipefail

# lint/prettier all projects with a package.json that are NOT 3rd party node modules
# Using -mindepth 2 to ignore the repo root package.json
# Currently samples/auth0nodejs is not part of our yarn workspaces, so we skip it.
WEB_LINT_DIRS=$(find . -name "package.json" -mindepth 2 | sed -E 's/package.json//' | grep -v "node_modules" | grep -v ".next")
NODE_ENV="${NODE_ENV:=development}"
for DIR in $WEB_LINT_DIRS; do
  pushd "$DIR" >/dev/null
  if [[ $DIR =~ ^"./public-repos/" || $DIR =~ ^"./samples/" ]]; then
    echo "Yarn install in $DIR..."
    yarn install # these dirs aren't part of our yarn workspaces, so we need to make sure their node_modules are up to date
  fi

  echo "Linting JS/TS/CSS/etc in $DIR..."
  NODE_ENV=$NODE_ENV yarn run lint
  # sadly there is no way to show what `prettier` actually wants you to fix; the devs
  # seem oddly opinionated that they only want to list the files with issues OR fix them,
  # but there's no way to dump a diff of proposed fixes without applying it.
  # see https://github.com/prettier/prettier/issues/6885
  PRETTIER_EXITCODE=0
  NODE_ENV=$NODE_ENV yarn run format -c || PRETTIER_EXITCODE=$?
  if [ $PRETTIER_EXITCODE -ne 0 ]; then
    echo "NOTE: running 'make lintfix' should address issues from prettier"
    exit "$PRETTIER_EXITCODE"
  fi
  popd >/dev/null
done
