#!/usr/bin/env bash

set -eou pipefail

source ./tools/packaging/helpers.sh
source ./tools/packaging/logging.sh
CURRENT_VERSION="$(current_version)"
# sets IMAGE_BASE_NAME
set_userclouds_service_image_base_name
UC_NAME="automatedprovisioner"
./tools/packaging/build-go-binary.sh automatedprovisioner
LOCAL_ONLY_IMAGE_TAG="userclouds-${UC_NAME}:latest"
IMAGE_REMOTE_TAG_VERSIONED="${IMAGE_BASE_NAME}${UC_NAME}:${CURRENT_VERSION}"
IMAGE_REMOTE_TAG_LATEST="${IMAGE_BASE_NAME}${UC_NAME}:latest"
docker build --platform=linux/amd64 \
  --tag "${IMAGE_REMOTE_TAG_VERSIONED}" \
  --tag "${IMAGE_REMOTE_TAG_LATEST}" \
  --tag "${LOCAL_ONLY_IMAGE_TAG}" \
  --build-arg CURRENT_REPO_VERSION="${CURRENT_VERSION}" \
  --file docker/onprem/automatedprovisioner/Dockerfile .

# Skip ECR/GHCR push by default (used for local development and CI testing)
if [ "${PUSH_TO_ECR_PRIVATE:-}" == "true" ] || [ "${PUSH_TO_GHCR:-}" == "true" ]; then
  info "Push ${UC_NAME} image to ECR ${IMAGE_REMOTE_TAG_VERSIONED}"
  docker push "${IMAGE_REMOTE_TAG_VERSIONED}"
  if [ "$UC_UNIVERSE" != "onprem" ]; then
    # On prem repos are immutable, so we don't push the latest tag
    docker push "${IMAGE_REMOTE_TAG_LATEST}"
  fi
else
  info "Not pushing ${UC_NAME} image to ECR"
fi
