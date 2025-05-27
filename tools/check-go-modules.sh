#!/usr/bin/env bash

set -euo pipefail

# Sync go workspace package versions, ensuring that child module dependency
# versions match the monorepo versions
go work sync

# Note: skip public-repos/sdk-golang/ since additional files get synced in
# there as part of the SDK release process, so it can't be properly checked
# as-is. We check that separately in CI
GO_MOD_DIRS=$(find . -name "go.mod" | sed -E 's/go.mod//' | grep -v "public-repos/sdk-golang/")
for DIR in $GO_MOD_DIRS; do
  echo "Check go modules in $DIR"
  pushd "$DIR" >/dev/null
  go mod tidy
  popd >/dev/null
done

RES=$(git status --porcelain)
if [ -n "$RES" ]; then
  echo "ERROR: running 'go mod tidy' changed files in:"
  echo "${GO_MOD_DIRS}"
  exit 1
fi

exit 0
