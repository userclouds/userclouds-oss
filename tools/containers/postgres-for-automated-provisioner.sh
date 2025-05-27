#!/usr/bin/env bash

set -uxo pipefail
docker compose -f docker/onprem/automatedprovisioner/docker-compose.yaml down
docker volume rm automatedprovisioner_postgres-on-prem-data
docker compose -f docker/onprem/automatedprovisioner/docker-compose.yaml up --detach
