#!/usr/bin/env bash

set -eou pipefail

source ./tools/packaging/logging.sh
if [ "$UC_UNIVERSE" == "dev" ]; then
  PUSH_TO_ECR_PRIVATE="false"
elif [ "$UC_UNIVERSE" != "debug" ] && [ "$UC_UNIVERSE" != "staging" ] && [ "$UC_UNIVERSE" != "onprem" ]; then
  error "invalid environment '$UC_UNIVERSE': use  staging, debug, onprem or dev - define the UC_UNIVERSE environment variable. prod is not allowed (prod should use images built for staging)"
  exit 1
fi

# TODO: make sure we are on a clean branch for debug. and on the master branch for staging (this should not run in prod)
source ./tools/packaging/helpers.sh
CURRENT_VERSION="$(current_version)"
# sets IMAGE_BASE_NAME
set_userclouds_service_image_base_name

info "Build Userclouds Service Container"
time ./tools/packaging/build-go-binary.sh userclouds
LOCAL_ONLY_IMAGE_TAG="userclouds:latest" # Useful when running this script locally without pushing to ECR
IMAGE_REMOTE_TAG_VERSIONED="${IMAGE_BASE_NAME}userclouds:${CURRENT_VERSION}"
IMAGE_REMOTE_TAG_LATEST="${IMAGE_BASE_NAME}userclouds:latest"
time make console/consoleui/build
time make plex/plexui/build

time docker build --platform=linux/amd64 \
  --tag "${IMAGE_REMOTE_TAG_VERSIONED}" \
  --tag "${IMAGE_REMOTE_TAG_LATEST}" \
  --tag "${LOCAL_ONLY_IMAGE_TAG}" \
  --progress plain \
  --build-arg CONSOLE_UI_ASSETS_DIR=console/consoleui/build \
  --build-arg PLEX_UI_ASSETS_DIR=plex/plexui/build \
  --build-arg CURRENT_REPO_VERSION="$(current_version)" \
  --file docker/service/Dockerfile .

# Skip ECR/GHCR push by default (used for local development and CI testing)
if [ "${PUSH_TO_ECR_PRIVATE:-}" == "true" ] || [ "${PUSH_TO_GHCR:-}" == "true" ]; then
  info "Pushing image ${IMAGE_REMOTE_TAG_VERSIONED}"
  docker push "${IMAGE_REMOTE_TAG_VERSIONED}"
fi
