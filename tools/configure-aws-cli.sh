#!/usr/bin/env bash

set -euo pipefail

AWS_CONFIG_FILE="${HOME}/.aws/config"
AWS_SSO_CONFIG_BASE=./config/local-dev/aws-cli-sso-config.ini

if [ -f "$AWS_CONFIG_FILE" ]; then
  echo "AWS CLI Config file ($AWS_CONFIG_FILE) already exists. Do you want to overwrite it with UserClouds AWS SSO config? (y/N)"
  read -r response
  if [[ $response == "y" ]]; then
    echo "Overwrting ${AWS_CONFIG_FILE} from ${AWS_SSO_CONFIG_BASE}"
    cp "${AWS_SSO_CONFIG_BASE}" "${AWS_CONFIG_FILE}"
  else
    echo "Not overwrting ${AWS_CONFIG_FILE}"
  fi
else
  echo "Creating ${AWS_CONFIG_FILE} from ${AWS_SSO_CONFIG_BASE}"
  cp "${AWS_SSO_CONFIG_BASE}" "${AWS_CONFIG_FILE}"
fi
