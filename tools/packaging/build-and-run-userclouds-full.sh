#!/usr/bin/env bash

set -euo pipefail

COMPOSE_FILE=docker/userclouds-full/docker-compose.yaml
source ./tools/packaging/helpers.sh
docker compose -f $COMPOSE_FILE down
./tools/packaging/build-go-binary.sh migrate provision idp plex authz logserver console worker containerrunner ucconfig
make plex/plexui/build console/consoleui/build
# Forces postgres DB init from scratch
docker volume rm -f userclouds-full_postgres-db-data-full
go run cmd/exportrouting/main.go docker/userclouds-full/routing.yaml idp plex authz console logserver
docker compose -f $COMPOSE_FILE build --build-arg CURRENT_REPO_VERSION="$(current_version)"
docker image inspect --format='{{json .Config.Labels}}' userclouds/userclouds:latest
docker compose -f $COMPOSE_FILE up
