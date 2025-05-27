#!/usr/bin/env bash

command -v git-lfs >/dev/null 2>&1 || {
  printf >&2 "\nThis repository is configured for Git LFS but 'git-lfs' was not found on your path. If you no longer wish to use Git LFS, remove this hook by deleting .git/hooks/post-checkout.\n"
  exit 2
}
git lfs post-checkout "$@"

FILEV=$(cat .go-version)
RUNV=$(go env | grep GOVERSION | sed -E 's/^GOVERSION=.*go([0-9\.]*).*$/\1/')

if [ "$FILEV" != "$RUNV" ]; then
  echo ".go-version ($FILEV) differs from go env ($RUNV) ... running direnv allow to reload"
  direnv allow
fi

# if any relevant yarn files changed between the last HEAD ($1) and the new HEAD ($2)
# automatically run `yarn install`
if git diff "$1".."$2" --name-only | grep -q -E "package.json|yarn.lock"; then
  echo "Running yarn install"
  yarn install --immutable
fi
