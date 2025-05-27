#!/bin/bash
set -eoxu pipefail

bin_names=("$@")
regions="us-east-1 us-west-2 eu-west-1"

for AWS_REGION in $regions; do
  # shellcheck disable=SC2128
  for rn in $bin_names; do
    imageDigests=$(aws ecr list-images --no-cli-pager --repository-name "userclouds/ecr/v1/${rn}" --region "${AWS_REGION}" --query 'imageIds[*].imageDigest' --output text)
    if [ -z "$imageDigests" ]; then
      echo "No images found in repository userclouds/ecr/v1/${rn} for region ${AWS_REGION}"
      continue
    fi
    for digest in $imageDigests; do
      aws ecr batch-delete-image --no-cli-pager --repository-name "userclouds/ecr/v1/${rn}" --region "${AWS_REGION}" --image-ids imageDigest="${digest}"
    done
  done
done
