#!/usr/bin/env bash
set -euo pipefail

# This script will push the current branch to Github and create a tag.
# Which will trigger a Github actions workflow to build and push new container images to ECR.
# Which in turn will trigger a deploy to the UC on prem env on debug-us-east-1 cluster (via ArgoCD and specifically ArgoCD Image updater)

if [ "$(git status --porcelain)" != "" ]; then
  echo "working directory is dirty, please commit or stash your changes"
  exit 1
fi

# Must be the same as terraform/modules/userclouds/applications/uc-on-prem/main.tf local.on_prem_deploy_branch
ONPREM_DEBUG_DEPLOY_BRANCH="deploy/on-prem/debug"

git push -f origin HEAD:$ONPREM_DEBUG_DEPLOY_BRANCH
echo "deploying to on-prem environment..."
source ./tools/packaging/helpers.sh
# shellcheck disable=SC2155
export CURRENT_VERSION="$(current_version)"
TAG="on-prem/debug/$CURRENT_VERSION"
echo "tagging on prem debug release: $TAG"
git tag "$TAG" HEAD && git push origin "$TAG"
