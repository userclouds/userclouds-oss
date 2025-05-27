#!/usr/bin/env bash

set -euo pipefail

source ./tools/packaging/logging.sh

function get_package() {
  binary="$1"
  if [[ -f "./${binary}/cmd/main.go" ]]; then
    echo "./${binary}/cmd"
  elif [[ -f "./cmd/${binary}/main.go" ]]; then
    echo "./cmd/${binary}"
  else
    echo "Can't find binary ${binary} is in ./cmd or ./${binary}/cmd"
    exit 1
  fi
}

function current_version() {
  # We prefix the git hash with the commit's timestamp, so that the versions sort chronologically.
  # This makes it easier to clean up old versions
  #
  # For example:
  # git show: 2023-08-10T17:18:13+00:00@82a682898a310964d733e6091c5355d044315f25
  # final:    2023-08-10.17-18-13-82a682898a31
  #
  TZ=UTC git show -s --format=%cd@%H --date=iso-strict-local HEAD |
    sed -E -f <(
      cat <<EOF
# Drop the UTC timezone
s|\+00:00||

# <date>.<time> instead of <date>T<time>
s|T|.|

# H-M-S instead of H:M:S
s|:|-|g

# Just the leading 12 characters of the full commit sha.
s|@(.{12}).*$|-\1|
EOF
    )
}

function current_branch() {
  echo "${CODEBUILD_WEBHOOK_HEAD_REF}" | sed -e "s/^refs\/heads\///"
}

function set_userclouds_service_image_base_name() {
  if [ "${PUSH_TO_ECR_PRIVATE:-}" == "true" ] && [ "${CI:-}" != "true" ]; then
    # in CI, AWS Account ID is already set
    AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
  fi

  if [ -z "${AWS_ACCOUNT_ID:-}" ]; then
    info "AWS_ACCOUNT_ID is not set, not pushing to ECR"
    PUSH_TO_ECR_PRIVATE="false"
  fi

  if [ "${PUSH_TO_ECR_PRIVATE:-}" == "true" ]; then
    AWS_REGION="us-west-2"
    PRIVATE_IMAGE_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
    info "Login to ECR registry for ${UC_UNIVERSE}: ${PRIVATE_IMAGE_REGISTRY}"
    aws ecr get-login-password --region "${AWS_REGION}" | docker login --username AWS --password-stdin "${PRIVATE_IMAGE_REGISTRY}"
    if [ "$UC_UNIVERSE" == "onprem" ]; then
      IMAGE_BASE_NAME="${PRIVATE_IMAGE_REGISTRY}/onprem/"
    else
      # see: terraform/configurations/aws/ecr-repos.hcl userclouds-single-binary
      IMAGE_BASE_NAME="${PRIVATE_IMAGE_REGISTRY}/"
    fi
  elif [ "${PUSH_TO_GHCR:-}" == "true" ] && [ "$UC_UNIVERSE" == "onprem" ]; then
    if [ -z "${GITHUB_TOKEN:-}" ] || [ -z "${GITHUB_ACTOR:-}" ]; then
      echo "Error: GITHUB_TOKEN and GITHUB_ACTOR must be set to push to GitHub Container Registry"
      exit 1
    fi
    echo "Authenticating to GitHub Container Registry"
    echo "${GITHUB_TOKEN}" | docker login ghcr.io -u "${GITHUB_ACTOR}" --password-stdin
    IMAGE_BASE_NAME="ghcr.io/userclouds/"
  else
    # shellcheck disable=SC2034
    IMAGE_BASE_NAME="local-"
  fi
}
