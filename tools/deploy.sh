#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

if [ "$(git status --porcelain)" != "" ]; then
  echo "working directory is dirty, please commit or stash your changes"
  exit 1
fi

UCENV="${1:-}"

source tools/check-env.sh

# this function is passed as a callable param to validate_environment, but
# SC isn't that sophisticated, so we need to disable the warning
# shellcheck disable=SC2317
function print_usage() {
  echo "usage: make deploy-[env]"
}

validate_environment "$UCENV" print_usage

if [ "$UCENV" == "prod" ]; then
  LOCALBRANCH=deploy/prod
elif [ "$UCENV" == "staging" ]; then
  LOCALBRANCH=deploy/staging
elif [ "$UCENV" == "debug" ]; then
  LOCALBRANCH=deploy/debug
else
  echo "usage: tools/deploy.sh environment"
  echo "  environment: prod | staging | debug"
  exit 1
fi

UC_UNIVERSE="$UCENV" tools/ensure-aws-auth.sh
UC_UNIVERSE="$UCENV" tools/ensure-deploy-eks.sh

if [ "$UCENV" != "debug" ]; then
  tools/ensure-deploy-test-env.sh
  # For prod & staging deploy, we need to make sure user has the ArgoCD CLI and is authenticated w/ ArgoCD
  # since for prod we use the ArgoCD CLI to trigger the deployment/sync and
  # for staging we use the ArgoCD CLI to wait for the deployment to finish before starting the deploy tests.
  UC_UNIVERSE="$UCENV" ./tools/ensure-argocd-auth.sh
fi

echo "deploying to $UCENV environment..."

# NB: this script assumes that our github "master" repo is the remote called "origin"
# and that we only deploy using the branch deploy/prod or deploy/staging
git fetch origin

DEPLOYED=origin/$LOCALBRANCH

python3 tools/deploy_prs.py "$DEPLOYED" HEAD

if [ "$UCENV" == "prod" ]; then
  # in prod, we should normally be deploying whatever bits are on staging
  if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/deploy/staging)" ]; then
    REPLY="n"
    read -p "HEAD doesn't appear to match origin/deploy/staging. Continue? [y/N] " -r REPLY
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
  fi
elif [ "$UCENV" == "staging" ]; then
  # in staging, we should normally be deploying master
  if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/master)" ]; then
    REPLY="n"
    read -p "HEAD doesn't appear to match origin/master Continue? [y/N] " -r REPLY
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
  fi
fi

# Currently in EB we don't have any way to run different builds on different services,
# so the naive way (just idp/userstore) will work. Will need to fix this soon enough.
# We only get the first line from the response since we may return more info in subsequent lines
# TODO: should we get this URL from config?
DEPLOYED_URL=https://idp."$UCENV".userclouds.com/deployed
if ! DEPLOYED_ACTUAL=$(curl -s "${DEPLOYED_URL}" | head -n 1); then
  echo "Failed to fetch deployed hash from $DEPLOYED_URL"
  exit 1
fi

if [ "$DEPLOYED_ACTUAL" != "$(git rev-parse $DEPLOYED)" ]; then
  REPLY="n"
  read -p "The currently deployed hash doesn't appear to match $DEPLOYED. Maybe a deploy failed or is in progress? Continue? [y/N] " -r REPLY
  [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

set +e
ANCESTOR=0
git merge-base --is-ancestor $DEPLOYED HEAD && ANCESTOR=1
set -e

if [ $ANCESTOR -eq 0 ]; then
  REPLY="n"
  read -p "The currently deployed tag doesn't appear to be an ancestor of HEAD. Continue? [y/N] " -r REPLY
  [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
fi

# ensure AWS env vars are set for this environment
UC_UNIVERSE=$UCENV tools/ensure-aws-auth.sh

SUCCESS=0
set +e
# TODO: rootdb?
UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/migrate --checkDeployed tenantdb companyconfig status && SUCCESS=1
set -e

if [ $SUCCESS -eq 0 ]; then
  REPLY="n"
  read -p "$UCENV migration check failed...run migrate-$UCENV? [yN] " -r REPLY
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    make migrate-"$UCENV"
    make deploy-"$UCENV" # and re-run ourselves for convenience
    exit 0               # terminate this since we re-entered ourselves above
  else
    REPLY="n"
    read -p "Continuing will probably break $UCENV...are you sure? [yN] " -r REPLY
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
  fi
fi

if [ "$UCENV" != "debug" ]; then
  TAG="deploy-$UCENV-$(date -u +%Y%m%d-%H%M%S)"
  echo "tagging release $TAG"
  git tag "$TAG" HEAD
  git push origin "$TAG"
fi

echo "pushing HEAD to $LOCALBRANCH"
git push -f origin HEAD:$LOCALBRANCH

if [ "$UCENV" == "prod" ]; then
  # For prod, ArgoCD autosync is disabled, so we need to manually sync the apps. the images we use are replicated from the AWC ECR in staging (so we use the same images we deployed to staging)
  # Note that for debug and staging, ArgoCD will automatically deploy when the new image is available in the ECR. which is not the behavior we want/have in prod (otherwise prod will be deployed when staging is deployed)
  # Hence this step to manually sync the apps in prod
  echo "deploying to prod..."
  UC_UNIVERSE="$UCENV" tools/deploy-with-argocd.sh
else
  tools/deploy-status.sh "$UCENV"
fi

if [ "$UCENV" != "debug" ]; then
  tools/deploy-tests.sh "$UCENV"
fi
