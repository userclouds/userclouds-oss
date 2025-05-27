#!/usr/bin/env bash

# set up our git hooks to automatically handle eg tests / lint

set -euo pipefail
IFS=$'\n\t'

mkdir -p .git/hooks
pushd .git/hooks
ln -f -s ../../tools/git/pre-commit.sh pre-commit
ln -f -s ../../tools/git/pre-push.sh pre-push
ln -f -s ../../tools/git/post-checkout.sh post-checkout
ln -f -s ../../tools/git/post-merge.sh post-merge
ln -f -s ../../tools/git/post-commit.sh post-commit
popd
