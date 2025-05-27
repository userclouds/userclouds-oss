#!/usr/bin/env bash

set -euo pipefail

# This script pushes a github tag which triggers the Github Actions Workflow:
# .github/workflows/build-on-prem-containers.yml to build and push on-prem containers to ghcr.io/userclouds

if [ "$(git status --porcelain)" != "" ]; then
  echo "working directory is dirty, please commit or stash your changes"
  exit 1
fi

git fetch origin

if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/master)" ]; then
  echo "ERROR: HEAD doesn't appear to match origin/master"
  echo "Only versions that are tested in staging should be tagged for UC On Prem release"
  exit 1
fi

if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/deploy/staging)" ]; then
  echo "ERROR: HEAD doesn't appear to match origin/deploy/staging."
  echo "Only versions that are tested in staging should be tagged for UC On Prem release"
  exit 1
fi

TAG="on-prem/$(date -u +%Y%m%d-%H%M%S)"
echo "tagging release $TAG"
git tag "$TAG" HEAD
git push origin "$TAG"
