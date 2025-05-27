#!/usr/bin/env bash

set -euo pipefail

# Passowrd must match PG_PASSWORD in docker/onprem/automatedprovisioner/docker-compose.yaml
PG_PASSWORD=vileweed ADMIN_USER_EMAIL=jerry@seinfeld.com GOOGLE_CLIENT_ID=newman COMPANY_NAME=davis CUSTOMER_DOMAIN=bob.jerry.io UC_CONFIG_DIR=docker/onprem/automatedprovisioner/config UC_UNIVERSE=onprem go run ./cmd/automatedprovisioner
