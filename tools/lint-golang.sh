#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# NB: for some reason I don't fully understand, AWS CodeBuild output logging breaks
# after we use `tee` to make these tests simpler (as in `test -z "$(gofmt -s -l . | tee /dev/stderr)"`).
# So we are explicitly capturing the output of each command, checking it for empty, and if needed,
# printing it out ourselves so that CI lint failures are visible.

echo "Checking Go formatting with gofmt..."
GOFMT=$(gofmt -s -l .)
test -z "$GOFMT" || (echo "gofmt needs running on:" && echo "$GOFMT" && exit 1)

echo "Checking Go import grouping with goimports..."
GOIMPS=$(goimports -d -local userclouds.com -l .)
test -z "$GOIMPS" || (echo "goimports needs running on:" && echo "$GOIMPS" && exit 1)

echo "Checking Go style grouping with modernize"
GOMIDERNIZE=$(modernize ./...)
test -z "$GOMIDERNIZE" || (echo "run ./tools/lintfix.sh" && echo "$GOMIDERNIZE" && exit 1)

# it turns out that the go pathspec ./... excludes modules in subdirectories,
# and we'd like to validate everything (samples, plex, clients, etc), so we'll
# find each submodule and lint there as well (note this includes ./go.mod).
# Skip public-repos/sdk-golang because it only includes SDK-specific files and
# is missing all the other golang sources that get synced in later -- we can run
# lint on the golang sdk separately in the future.
GO_MOD_DIRS=$(find . -name "go.mod" | sed -E 's/go.mod//' | grep -v "public-repos/sdk-golang/")

# we need to capture our current wd so we can run correctly-pathed tools in CI (without PATH set)
# this is the root of the repo, and is probably a CODEBUILD env var too, but this also works locally
BASE=$(pwd)

# locally, we install everything to GOBIN (set in .envrc) and GOPATH isn't set, but in CodeBuild,
# GOBIN isn't defined (default is used) but GOPATH is, so we recreate the default value to ensure
# we can access the tools we installed in the Makefile. For clarify, GOBIN defaults to GOPATH[0]/bin
# and we do the usual IFS-preservation BS for bash. See https://pkg.go.dev/cmd/go
if [[ ! -v GOBIN && -v GOPATH ]]; then
  OLDIFS=$IFS
  IFS=:
  # shellcheck disable=SC2206
  GOPATH_ARRAY=($GOPATH)
  IFS=$OLDIFS
  GOBIN="${GOPATH_ARRAY[0]}/bin"
fi

# and cache IFS so we can split on newlines only to parse find's results (yay Bash)
# note that we call the binaries with a specific path since we're changing wd as we go
OLDIFS=$IFS
IFS=$'\n'
for DIR in $GO_MOD_DIRS; do
  echo "Linting go module in $DIR"
  pushd "$DIR" >/dev/null

  # go is obviously on the path so unqualified
  echo "- Linting Go with vet..."
  go vet ./...

  # we installed golint via `go install` -> $GOBIN
  echo "- Linting Go with revive..."
  "$GOBIN/revive" -config "$BASE/tools/revive.toml" -set_exit_status ./...

  # likewise we installed staticcheck via `go install`
  echo "- Linting Go with staticcheck..."
  "$GOBIN/staticcheck" ./...

  # we build uclint specifically to [repo root]/bin in Makefile
  echo "- Running Userclouds linters..."
  "$BASE/bin/uclint" ./...

  popd >/dev/null
done

# TODO: don't run this on samples for now
# errcheck is also `go install`ed, and the relative path to exclude changes
echo "Running errcheck on main userclouds project only (not samples for now), which checks for unhandled errors..."
"$GOBIN/errcheck" -blank -exclude "$BASE/tools/errcheck_exclude.txt" ./...
IFS=$OLDIFS

# these scripts are grep-based, so doesn't use ./...
echo "Userclouds lint check for accidentally-included fmt.Print, etc"
tools/check-prints.sh
