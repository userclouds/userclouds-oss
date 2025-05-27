#!/usr/bin/env bash

set -eou pipefail

UC_UNIVERSE=debug PUSH_TO_ECR_PRIVATE=false ./tools/packaging/build-and-publish-uc-service-container.sh
