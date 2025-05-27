#!/usr/bin/env bash

set -euo pipefail

PULL_IMAGE=false
IMAGE_NAME="userclouds/uc-headless:latest"

# Parse arguments
for arg in "$@"; do
  case $arg in
  --pull)
    PULL_IMAGE=true
    shift
    ;;
  *)
    # First non-flag argument is treated as image name
    if [[ $arg != --* ]]; then
      IMAGE_NAME="$arg"
      shift
    fi
    ;;
  esac
done

# Used in the docker compose file
export IMAGE_NAME
COMPOSE_FILE="docker/userclouds-headless/docker-compose.yaml"
# Forces postgres DB init from scratch
docker volume rm -f userclouds-headless_postgres-db-data-headless
source ./tools/packaging/helpers.sh
docker compose -f $COMPOSE_FILE down

if [ "$PULL_IMAGE" = false ]; then
  ./tools/packaging/build-go-binary.sh migrate provision idp plex authz containerrunner ucconfig
  go run cmd/exportrouting/main.go docker/userclouds-headless/routing.yaml idp plex authz
  docker compose -f $COMPOSE_FILE build --build-arg CURRENT_REPO_VERSION="$(current_version)"
  docker image inspect --format='{{json .Config.Labels}}' "${IMAGE_NAME}"
  docker compose -f $COMPOSE_FILE up
else
  docker compose -f $COMPOSE_FILE up --pull always
fi
