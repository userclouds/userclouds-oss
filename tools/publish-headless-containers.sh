#!/usr/bin/env bash

set -euo pipefail

if [ "$(git status --porcelain)" != "" ]; then
  echo "working directory is dirty, please commit or stash your changes"
  exit 1
fi

DOCKER_UPSTREAM_REPO="userclouds/uc-headless"
VERSION=${1:-}
if [ -z "$VERSION" ]; then
  echo "ERROR: Version argument is required"
  exit 1
fi

if [[ ! $VERSION =~ ^v ]]; then
  echo "ERROR: Version must start with 'v'"
  exit 1
fi

# Check if the tag already exists in Docker Hub
if docker manifest inspect "${DOCKER_UPSTREAM_REPO}:${VERSION}" >/dev/null 2>&1; then
  echo "ERROR: Version ${VERSION} already exists in Docker Hub"
  exit 1
fi

git fetch origin

if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/master)" ]; then
  echo "ERROR: HEAD doesn't appear to match origin/master"
  echo "Only versions that are tested in staging should be tagged for UserClouds Headless Container release"
  exit 1
fi

if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/deploy/staging)" ]; then
  echo "ERROR: HEAD doesn't appear to match origin/deploy/staging."
  echo "Only versions that are tested in staging should be tagged for UserClouds Headless Container release"
  exit 1
fi

TAG="headless-container/${VERSION}"
echo "tagging release $TAG"
git tag "$TAG" HEAD
git push origin "$TAG"
